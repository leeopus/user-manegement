package service

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ = auth.ErrTokenNotFound // 确保 auth 包被正确引用

const (
	refreshTokenTTL     = 30 * 24 * time.Hour
	maxRegisterAttempts = 10
)

var (
	dummyHash     string
	dummyHashOnce sync.Once
)

func getDummyHash() string {
	dummyHashOnce.Do(func() {
		h, err := utils.HashPassword("constant-time-dummy-value")
		if err != nil {
			dummyHash = "$2a$12$000000000000000000000uGYDq1WbOaJBgFBaFBaFBaFBaFBaFBa"
			return
		}
		dummyHash = h
	})
	return dummyHash
}

type AuthService interface {
	Register(email, password string, auditCtx dto.AuditContext) (*repository.User, error)
	Login(email, password, clientIP string, rememberMe bool) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, string, bool, error)
	Logout(userID uint, refreshToken, accessToken string) error
	LogoutAll(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
}

type authService struct {
	userRepo        repository.UserRepository
	auditLogger     *AuditLogger
	lockoutManager  *auth.AccountLockoutManager
	refreshTokenMgr *auth.RefreshTokenManager
	blacklistMgr    *auth.TokenBlacklistManager
}

func NewAuthService(userRepo repository.UserRepository, auditLogger *AuditLogger, redisClient *redis.Client, blacklistMgr *auth.TokenBlacklistManager, refreshTokenMgr *auth.RefreshTokenManager) AuthService {
	return &authService{
		userRepo:        userRepo,
		auditLogger:     auditLogger,
		lockoutManager:  auth.NewAccountLockoutManager(redisClient),
		refreshTokenMgr: refreshTokenMgr,
		blacklistMgr:    blacklistMgr,
	}
}

func (s *authService) Register(email, password string, auditCtx dto.AuditContext) (*repository.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if _, err := utils.ValidatePassword(password, ""); err != nil {
		return nil, apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if utils.IsDisposableEmail(email) {
		return nil, apperrors.ErrDisposableEmail
	}

	emailExists := false
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		emailExists = true
	}

	// 无论邮箱是否存在，都执行等量 bcrypt 工作保持恒定时间，防止时序侧信道枚举
	// 正常注册路径会执行 1 次 bcrypt hash；此处也执行 1 次 dummy check 对齐总耗时
	if emailExists {
		_ = utils.CheckPassword(password, getDummyHash())
		// 额外执行 1 次 dummy hash 以对齐正常路径的 HashPassword 调用
		_, _ = utils.HashPassword("constant-time-dummy-value")
		// 不返回"邮箱已存在"错误，防止邮箱枚举攻击
		// 返回通用成功消息，与正常注册一致
		return nil, apperrors.ErrRegisterSilent.WithDetails(map[string]interface{}{
			"hint": "if_this_email_is_available_you_will_receive_confirmation",
		})
	}

	username := utils.GenerateUsernameFromEmail(email)

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		zap.L().Error("Failed to hash password", zap.Error(err))
		return nil, apperrors.ErrInternalServer
	}

	var user *repository.User
	createErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		currentUsername := username

		for attempt := 0; attempt < maxRegisterAttempts; attempt++ {
			user = &repository.User{
				Username:     currentUsername,
				Email:        email,
				PasswordHash: passwordHash,
				Status:       "active",
			}

			if err := tx.Create(user).Error; err != nil {
				if isUniqueViolation(err) {
					if isUsernameConstraint(err) {
						suffix, sErr := utils.RandomSuffix(6)
						if sErr != nil {
							return apperrors.ErrInternalServer
						}
						currentUsername = username + suffix
						continue
					}
					return apperrors.ErrEmailAlreadyExists
				}
				return apperrors.ErrInternalServer
			}
			return nil
		}

		return apperrors.ErrInternalServer
	})

	if createErr != nil {
		if appErr, ok := apperrors.IsAppError(createErr); ok {
			return nil, appErr
		}
		zap.L().Error("Failed to create user", zap.String("email", email), zap.Error(createErr))
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    user.ID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "register", "user", map[string]interface{}{
		"ip":       auditCtx.IPAddress,
		"platform": extractPlatform(auditCtx.UserAgent),
	})

	return user, nil
}

func (s *authService) Login(email, password, clientIP string, rememberMe bool) (*repository.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	locked, remainingTime, err := s.lockoutManager.IsAccountLocked(email)
	if err != nil {
		// Redis 不可用：拒绝登录请求，防止被锁定账户绕过检查
		zap.L().Error("Account lockout check unavailable, rejecting login for security", zap.Error(err))
		return nil, "", "", apperrors.ErrInternalServer
	}
	if locked {
		minutes := int(remainingTime.Minutes())
		return nil, "", "", apperrors.ErrAccountLocked.WithDetails(map[string]interface{}{
			"remaining_minutes": minutes,
		})
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		_ = utils.CheckPassword("constant-time-dummy-value", getDummyHash())

		s.lockoutManager.RecordFailedAttempt(email, clientIP)
		return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"hint": "multiple_failures_will_lock_account",
		})
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		s.lockoutManager.RecordFailedAttempt(email, clientIP)
		return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"hint": "multiple_failures_will_lock_account",
		})
	}

	if user.Status != "active" {
		return nil, "", "", apperrors.ErrAccountNotActive
	}

	s.lockoutManager.ClearFailedAttempts(email, clientIP)

	// 缓存用户 active 状态，供 auth 中间件校验
	if err := s.blacklistMgr.SetUserStatus(user.ID, "active"); err != nil {
		zap.L().Warn("Failed to cache user status on login", zap.Uint("user_id", user.ID), zap.Error(err))
	}

	accessToken, _, err := utils.GenerateTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	refreshToken, _, err := utils.GenerateRefreshTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	if storeErr := s.refreshTokenMgr.Store(user.ID, refreshToken, refreshTokenTTL); storeErr != nil {
		zap.L().Error("Failed to store refresh token, aborting login", zap.Error(storeErr))
		return nil, "", "", apperrors.ErrInternalServer
	}

	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		zap.L().Warn("Failed to update last login", zap.Error(err))
	}

	userWithRoles, err := s.userRepo.FindByIDWithRoles(user.ID)
	if err != nil {
		s.auditLogger.Log(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP}, "login", "user", map[string]interface{}{
			"method": "password",
		})
		return user, accessToken, refreshToken, nil
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP}, "login", "user", map[string]interface{}{
		"method": "password",
	})

	return userWithRoles, accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshToken string) (*repository.User, string, string, bool, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return nil, "", "", false, apperrors.ErrInvalidRefreshToken
	}

	if claims.TokenType != "refresh" {
		return nil, "", "", false, apperrors.ErrInvalidRefreshToken
	}

	storedUserID, validateErr := s.refreshTokenMgr.Validate(refreshToken)
	if validateErr != nil {
		if errors.Is(validateErr, auth.ErrTokenNotFound) {
			if claims.ExpiresAt != nil && time.Now().Before(claims.ExpiresAt.Time) {
				zap.L().Warn("Refresh token reuse detected (JWT valid but Redis key missing), revoking all sessions",
					zap.Uint("user_id", claims.UserID),
					zap.String("jti", claims.JTI),
				)
				_ = s.refreshTokenMgr.RevokeAll(claims.UserID)
				_ = s.blacklistMgr.RevokeAllUserTokens(claims.UserID)
			}
			return nil, "", "", false, apperrors.ErrInvalidRefreshToken
		}
		zap.L().Warn("Refresh token validation failed due to store error",
			zap.Uint("user_id", claims.UserID),
			zap.Error(validateErr),
		)
		return nil, "", "", false, apperrors.ErrInvalidRefreshToken
	}
	if storedUserID != claims.UserID {
		return nil, "", "", false, apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, "", "", false, apperrors.ErrUserNotFound
	}

	// 从旧 refresh token 的 claims 中继承 RememberMe 状态
	rememberMe := claims.RememberMe

	newAccessToken, _, err := utils.GenerateTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", false, apperrors.ErrInternalServer
	}

	if err := s.refreshTokenMgr.Revoke(user.ID, refreshToken); err != nil {
		zap.L().Warn("Failed to revoke old refresh token", zap.Error(err))
	}

	if remaining := time.Until(claims.ExpiresAt.Time); remaining > 0 {
		if err := s.blacklistMgr.AddToBlacklist(claims.JTI, remaining); err != nil {
			zap.L().Warn("Failed to blacklist old refresh token JTI", zap.Error(err))
		}
	}

	newRefreshToken, _, err := utils.GenerateRefreshTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", false, apperrors.ErrInternalServer
	}
	if storeErr := s.refreshTokenMgr.Store(user.ID, newRefreshToken, refreshTokenTTL); storeErr != nil {
		zap.L().Warn("Failed to store new refresh token", zap.Error(storeErr))
	}

	return user, newAccessToken, newRefreshToken, rememberMe, nil
}

func (s *authService) Logout(userID uint, refreshToken, accessToken string) error {
	if refreshToken != "" {
		if err := s.refreshTokenMgr.Revoke(userID, refreshToken); err != nil {
			zap.L().Warn("Failed to revoke refresh token on logout",
				zap.Uint("user_id", userID), zap.Error(err))
		}
	}

	if accessToken != "" {
		if claims, err := utils.ParseToken(accessToken); err == nil {
			remaining := time.Until(claims.ExpiresAt.Time)
			if remaining > 0 {
				if err := s.blacklistMgr.AddToBlacklist(claims.JTI, remaining); err != nil {
					zap.L().Warn("Failed to blacklist access token on logout",
						zap.Uint("user_id", userID), zap.Error(err))
				}
			}
		}
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: userID}, "logout", "user", map[string]interface{}{
		"method": "manual",
	})

	return nil
}

func (s *authService) LogoutAll(userID uint) error {
	if err := s.refreshTokenMgr.RevokeAll(userID); err != nil {
		zap.L().Warn("Failed to revoke all refresh tokens",
			zap.Uint("user_id", userID), zap.Error(err))
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: userID}, "logout_all", "user", map[string]interface{}{
		"method": "all_devices",
	})

	return nil
}

func (s *authService) GetCurrentUser(userID uint) (*repository.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isUsernameConstraint 通过 PostgreSQL 约束名判断是否为 username 唯一约束冲突
func isUsernameConstraint(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName == ConstraintUsersUsernameKey
	}
	return false
}

func extractPlatform(userAgent string) string {
	if userAgent == "" {
		return "unknown"
	}
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "mobile"):
		return "mobile"
	case strings.Contains(ua, "tablet"):
		return "tablet"
	default:
		return "desktop"
	}
}
