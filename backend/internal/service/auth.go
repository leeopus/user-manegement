package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/config"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

var _ = auth.ErrTokenNotFound // 确保 auth 包被正确引用

const (
	maxRegisterAttempts = 10
)

func getRefreshTokenTTL() time.Duration {
	return config.Get().GetRefreshTokenTTL()
}

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
	Login(email, password, clientIP, userAgent string, rememberMe bool) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, string, bool, error)
	Logout(userID uint, refreshToken, accessToken string) error
	LogoutAll(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
	ChangePassword(userID uint, currentPassword, newPassword string, auditCtx dto.AuditContext) error
}

type authService struct {
	userRepo            repository.UserRepository
	passwordHistoryRepo repository.PasswordHistoryRepository
	auditLogger         *AuditLogger
	lockoutManager      *auth.AccountLockoutManager
	refreshTokenMgr     *auth.RefreshTokenManager
	blacklistMgr        *auth.TokenBlacklistManager
}

func NewAuthService(userRepo repository.UserRepository, passwordHistoryRepo repository.PasswordHistoryRepository, auditLogger *AuditLogger, redisClient *redis.Client, blacklistMgr *auth.TokenBlacklistManager, refreshTokenMgr *auth.RefreshTokenManager) AuthService {
	return &authService{
		userRepo:            userRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		auditLogger:         auditLogger,
		lockoutManager:      auth.NewAccountLockoutManager(redisClient),
		refreshTokenMgr:     refreshTokenMgr,
		blacklistMgr:        blacklistMgr,
	}
}

func (s *authService) Register(email, password string, auditCtx dto.AuditContext) (*repository.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	// 输入验证直接返回错误，依赖 RegisterRateLimit 防止邮箱枚举
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

	// 无论邮箱是否存在，都执行一次 bcrypt 保持恒定时间，防止时序侧信道枚举
	if emailExists {
		_, _ = utils.HashPassword("constant-time-dummy-value")
		// 不返回"邮箱已存在"错误，防止邮箱枚举攻击
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

		now := time.Now()
		for attempt := 0; attempt < maxRegisterAttempts; attempt++ {
			user = &repository.User{
				Username:          currentUsername,
				Email:             email,
				PasswordHash:      passwordHash,
				Status:            "active",
				PasswordChangedAt: &now,
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

func (s *authService) Login(email, password, clientIP, userAgent string, rememberMe bool) (*repository.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if err := utils.ValidateEmail(email); err != nil {
		return nil, "", "", apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

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

	if storeErr := s.refreshTokenMgr.Store(user.ID, refreshToken, getRefreshTokenTTL()); storeErr != nil {
		zap.L().Error("Failed to store refresh token, aborting login", zap.Error(storeErr))
		return nil, "", "", apperrors.ErrInternalServer
	}

	if err := s.userRepo.UpdateLastLogin(user.ID, clientIP); err != nil {
		zap.L().Warn("Failed to update last login", zap.Error(err))
	}

	userWithRoles, err := s.userRepo.FindByIDWithRoles(user.ID)
	if err != nil {
		s.auditLogger.Log(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP, UserAgent: userAgent}, "login", "user", map[string]interface{}{
			"method": "password",
		})
		return user, accessToken, refreshToken, nil
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP, UserAgent: userAgent}, "login", "user", map[string]interface{}{
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

	// 先检查 token 是否在 Redis 中存在（用于重放检测）
	_, validateErr := s.refreshTokenMgr.Validate(refreshToken)
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

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, "", "", false, apperrors.ErrUserNotFound
	}

	if user.Status != "active" {
		// 用户已被禁用/删除，撤销其所有 token
		_ = s.refreshTokenMgr.RevokeAll(user.ID)
		_ = s.blacklistMgr.RevokeAllUserTokens(user.ID)
		return nil, "", "", false, apperrors.ErrAccountNotActive
	}

	// 从旧 refresh token 的 claims 中继承 RememberMe 状态
	rememberMe := claims.RememberMe

	newAccessToken, _, err := utils.GenerateTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", false, apperrors.ErrInternalServer
	}

	newRefreshToken, _, err := utils.GenerateRefreshTokenWithRememberMe(user.ID, user.Username, user.Email, rememberMe)
	if err != nil {
		return nil, "", "", false, apperrors.ErrInternalServer
	}

	// 原子旋转：validate-old → revoke-old → store-new，单次 Redis 操作
	if rotateErr := s.refreshTokenMgr.Rotate(claims.UserID, refreshToken, newRefreshToken, getRefreshTokenTTL()); rotateErr != nil {
		zap.L().Error("Failed to rotate refresh token atomically",
			zap.Uint("user_id", claims.UserID),
			zap.Error(rotateErr),
		)
		return nil, "", "", false, apperrors.ErrInvalidRefreshToken
	}

	// Best-effort: 将旧 refresh token JTI 加入黑名单
	if remaining := time.Until(claims.ExpiresAt.Time); remaining > 0 {
		if err := s.blacklistMgr.AddToBlacklist(claims.JTI, remaining); err != nil {
			zap.L().Warn("Failed to blacklist old refresh token JTI", zap.Error(err))
		}
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

func (s *authService) ChangePassword(userID uint, currentPassword, newPassword string, auditCtx dto.AuditContext) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	if !utils.CheckPassword(currentPassword, user.PasswordHash) {
		return apperrors.ErrCurrentPasswordIncorrect
	}

	if _, err := utils.ValidatePassword(newPassword, user.Username); err != nil {
		return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if utils.CheckPassword(newPassword, user.PasswordHash) {
		return apperrors.ErrPasswordSameAsOld
	}

	histories, histErr := s.passwordHistoryRepo.FindByUserID(user.ID, 5)
	if histErr != nil {
		zap.L().Error("Failed to check password history, aborting change for security",
			zap.Uint("user_id", user.ID), zap.Error(histErr))
		return apperrors.ErrInternalServer
	}
	if len(histories) > 0 {
		g, _ := errgroup.WithContext(context.Background())
		reused := make([]bool, len(histories))
		for i, h := range histories {
			i, h := i, h
			g.Go(func() error {
				if utils.CheckPassword(newPassword, h.PasswordHash) {
					reused[i] = true
				}
				return nil
			})
		}
		_ = g.Wait()
		for _, r := range reused {
			if r {
				return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
					"reason": "new password was used recently, please choose a different one",
				})
			}
		}
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return apperrors.ErrInternalServer
	}

	// 密码更新、历史记录必须在同一事务中，防止部分写入导致密码历史绕过
	txErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		user.PasswordHash = hashedPassword
		now := time.Now()
		user.PasswordChangedAt = &now
		if err := s.userRepo.UpdateWithTx(tx, user); err != nil {
			return err
		}

		if err := s.passwordHistoryRepo.CreateWithTx(tx, &repository.PasswordHistory{
			UserID:       user.ID,
			PasswordHash: hashedPassword,
		}); err != nil {
			zap.L().Warn("Failed to save password history on change", zap.Error(err))
		}
		return nil
	})
	if txErr != nil {
		return apperrors.ErrInternalServer
	}
	_ = s.passwordHistoryRepo.CleanupOld(user.ID, 5)

	// 密码修改后吊销所有已有 token，强制重新登录
	var revokeErr error
	if err := s.refreshTokenMgr.RevokeAll(user.ID); err != nil {
		zap.L().Error("CRITICAL: failed to revoke refresh tokens after password change",
			zap.Uint("user_id", user.ID), zap.Error(err))
		revokeErr = err
	}
	if err := s.blacklistMgr.RevokeAllUserTokens(user.ID); err != nil {
		zap.L().Error("CRITICAL: failed to blacklist user tokens after password change",
			zap.Uint("user_id", user.ID), zap.Error(err))
		revokeErr = err
	}

	s.auditLogger.Log(&auditCtx, "change_password", "user", nil)

	if revokeErr != nil {
		return apperrors.ErrPasswordChangeRevokeFailed
	}

	return nil
}
