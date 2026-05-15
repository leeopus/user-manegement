package service

import (
	"context"
	"encoding/json"
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
	"gorm.io/gorm"
)

var _ = auth.ErrTokenNotFound // 确保 auth 包被正确引用

const (
	maxRegisterAttempts = 10
)

func getRefreshTokenTTL() time.Duration {
	return config.Get().GetRefreshTokenTTL()
}

func getRefreshTokenTTLForRememberMe(rememberMe bool) time.Duration {
	return config.Get().GetRefreshTokenTTLForRememberMe(rememberMe)
}

var (
	dummyHash     string
	dummyHashOnce sync.Once
)

// precomputedDummyHash 是一个合法的 bcrypt hash（cost=12），用于 crypto/rand 失败时的 fallback。
// 对应明文 "constant-time-dummy-value"，保证 bcrypt.CompareHashAndPassword 正常执行，
// 维持常量时间比较以防止时序侧信道枚举。
const precomputedDummyHash = "$2a$12$FGm4lLxTOiUDYUsXlf3hM.7y/8P5.i5q3FsOFjcUTD5tKEUctCmzG"

func getDummyHash() string {
	dummyHashOnce.Do(func() {
		h, err := utils.HashPassword("constant-time-dummy-value")
		if err != nil {
			zap.L().Error("crypto/rand failed, using precomputed dummy hash")
			dummyHash = precomputedDummyHash
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
	redis               *redis.Client
}

func NewAuthService(userRepo repository.UserRepository, passwordHistoryRepo repository.PasswordHistoryRepository, auditLogger *AuditLogger, redisClient *redis.Client, blacklistMgr *auth.TokenBlacklistManager, refreshTokenMgr *auth.RefreshTokenManager) AuthService {
	return &authService{
		userRepo:            userRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		auditLogger:         auditLogger,
		lockoutManager:      auth.NewAccountLockoutManager(redisClient),
		refreshTokenMgr:     refreshTokenMgr,
		blacklistMgr:        blacklistMgr,
		redis:               redisClient,
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

	username := utils.GenerateUsernameFromEmail(email)

	// 用生成的用户名重新校验密码（检查是否包含用户名）
	if _, err := utils.ValidatePassword(password, username); err != nil {
		return nil, apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 两条路径统一执行 bcrypt，消除时序差异
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		zap.L().Error("Failed to hash password", zap.Error(err))
		return nil, apperrors.ErrInternalServer
	}

	// 事务内 INSERT：邮箱不存在则创建用户，邮箱已存在则返回明确错误
	//
	// 使用 SAVEPOINT 保护每次 INSERT 尝试：PostgreSQL 在语句失败后将整个事务标记为
	// aborted，后续操作（包括 COMMIT）都会被拒绝。通过 SAVEPOINT + ROLLBACK TO
	// 可以恢复事务到正常状态，使后续操作或最终 COMMIT 正常执行。
	const spName = "register_attempt"
	var user *repository.User
	createErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		currentUsername := username

		now := time.Now()
		for attempt := 0; attempt < maxRegisterAttempts; attempt++ {
			candidate := &repository.User{
				Username:          currentUsername,
				Email:             email,
				PasswordHash:      passwordHash,
				Status:            "active",
				PasswordChangedAt: &now,
				Nickname:          generateNickname(tx, s.userRepo),
			}

			tx.SavePoint(spName)
			if err := tx.Create(candidate).Error; err != nil {
				tx.RollbackTo(spName)
				if isUniqueViolation(err) {
					if isUsernameConstraint(err) {
						suffix, sErr := utils.RandomSuffix(6)
						if sErr != nil {
							return apperrors.ErrInternalServer
						}
						currentUsername = username + suffix
						continue
					}
					// email 唯一约束冲突 → 邮箱已存在，返回明确错误
					return apperrors.ErrEmailAlreadyExists
				}
				return apperrors.ErrInternalServer
			}
			user = candidate
			return nil
		}

		return apperrors.ErrInternalServer
	})

	if createErr != nil {
		if appErr, ok := apperrors.IsAppError(createErr); ok {
			return nil, appErr
		}
		zap.L().Error("Failed to create user", zap.Error(createErr))
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

		if err := s.lockoutManager.RecordFailedAttempt(email, clientIP); err != nil {
			zap.L().Error("CRITICAL: failed to record failed login attempt, lockout may be bypassed",
				zap.String("email", email), zap.String("client_ip", clientIP), zap.Error(err))
		}
		return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"hint": "multiple_failures_will_lock_account",
		})
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		if err := s.lockoutManager.RecordFailedAttempt(email, clientIP); err != nil {
			zap.L().Error("CRITICAL: failed to record failed login attempt, lockout may be bypassed",
				zap.String("email", email), zap.String("client_ip", clientIP), zap.Error(err))
		}
		return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"hint": "multiple_failures_will_lock_account",
		})
	}

	if user.Status != "active" {
		return nil, "", "", apperrors.ErrAccountNotActive
	}

	s.lockoutManager.ClearFailedAttempts(email, clientIP)

	// 清除可能遗留的用户级吊销标记（如管理员先禁用再激活用户后未清除）
	if err := s.blacklistMgr.ClearUserRevoked(user.ID); err != nil {
		zap.L().Warn("Failed to clear user revocation on login", zap.Uint("user_id", user.ID), zap.Error(err))
	}

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

	if storeErr := s.refreshTokenMgr.Store(user.ID, refreshToken, getRefreshTokenTTLForRememberMe(rememberMe)); storeErr != nil {
		zap.L().Error("Failed to store refresh token, aborting login", zap.Error(storeErr))
		return nil, "", "", apperrors.ErrInternalServer
	}

	if err := s.userRepo.UpdateLastLogin(user.ID, clientIP); err != nil {
		zap.L().Warn("Failed to update last login", zap.Error(err))
	}

	userWithRoles, err := s.userRepo.FindByIDWithRoles(user.ID)
	if err != nil {
		_ = s.auditLogger.LogSync(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP, UserAgent: userAgent}, "login", "user", map[string]interface{}{
			"method": "password",
		})
		return user, accessToken, refreshToken, nil
	}

	_ = s.auditLogger.LogSync(&dto.AuditContext{UserID: user.ID, IPAddress: clientIP, UserAgent: userAgent}, "login", "user", map[string]interface{}{
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
	if rotateErr := s.refreshTokenMgr.Rotate(claims.UserID, refreshToken, newRefreshToken, getRefreshTokenTTLForRememberMe(rememberMe)); rotateErr != nil {
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

	// Notify other systems via Redis Pub/Sub
	if s.redis != nil {
		msg, _ := json.Marshal(map[string]interface{}{"user_id": userID})
		if err := s.redis.Publish(context.Background(), "sso:user:logout", msg).Err(); err != nil {
			zap.L().Warn("Failed to publish logout event", zap.Uint("user_id", userID), zap.Error(err))
		}
	}

	return nil
}

func (s *authService) LogoutAll(userID uint) error {
	if err := s.refreshTokenMgr.RevokeAll(userID); err != nil {
		zap.L().Warn("Failed to revoke all refresh tokens",
			zap.Uint("user_id", userID), zap.Error(err))
	}

	if err := s.blacklistMgr.RevokeAllUserTokens(userID); err != nil {
		zap.L().Warn("Failed to blacklist all user tokens",
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

// generateNickname generates a globally unique nickname within the transaction
func generateNickname(tx *gorm.DB, userRepo repository.UserRepository) string {
	for i := 0; i < 5; i++ {
		nickname, err := utils.GenerateUniqueNickname()
		if err != nil {
			continue
		}
		existing, _ := userRepo.FindByNicknameWithTx(tx, nickname)
		if existing == nil {
			return nickname
		}
	}
	// Fallback with timestamp suffix
	suffix, _ := utils.RandomSuffix(6)
	return "User_" + suffix
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
	for _, h := range histories {
		if utils.CheckPassword(newPassword, h.PasswordHash) {
			return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
				"reason": "new password was used recently, please choose a different one",
			})
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

	s.auditLogger.LogSync(&auditCtx, "change_password", "user", nil)

	if revokeErr != nil {
		return apperrors.ErrPasswordChangeRevokeFailed
	}

	return nil
}
