package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/redis"
	"github.com/user-system/backend/pkg/utils"
)

type AuthService interface {
	Register(email, password, clientIP, userAgent string) (*repository.User, error)
	Login(email, password string) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, error)
	Logout(userID uint, refreshToken string) error
	LogoutAll(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
}

type authService struct {
	userRepo        repository.UserRepository
	auditLogRepo    repository.AuditLogRepository
	lockoutManager  *auth.AccountLockoutManager
	refreshTokenMgr *auth.RefreshTokenManager
}

func NewAuthService(userRepo repository.UserRepository, auditLogRepo repository.AuditLogRepository) AuthService {
	return &authService{
		userRepo:        userRepo,
		auditLogRepo:    auditLogRepo,
		lockoutManager:  auth.NewAccountLockoutManager(redis.Client),
		refreshTokenMgr: auth.NewRefreshTokenManager(redis.Client),
	}
}

func (s *authService) Register(email, password, clientIP, userAgent string) (*repository.User, error) {
	// 1. 验证邮箱格式
	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 2. 验证密码强度
	if _, err := utils.ValidatePassword(password, ""); err != nil {
		return nil, apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 3. 检查是否为一次性邮箱
	if utils.IsDisposableEmail(email) {
		return nil, apperrors.ErrDisposableEmail
	}

	// 4. 检查邮箱是否已存在
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, apperrors.ErrEmailAlreadyExists
	}

	// 5. 自动生成username（从email前缀提取），处理冲突
	username := utils.GenerateUsernameFromEmail(email)
	for i := 0; i < 10; i++ {
		_, err := s.userRepo.FindByUsername(username)
		if err != nil {
			break // 用户名不存在，可以使用
		}
		// 用户名冲突，在基础名上追加随机后缀
		username = utils.GenerateUsernameFromEmail(email) + utils.RandomSuffix(4)
	}

	// 6. 加密密码
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		return nil, apperrors.ErrInternalServer
	}

	// 7. 创建用户
	user := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		log.Printf("ERROR: Failed to create user (email=%s, username=%s): %v", email, username, err)
		// 区分唯一约束冲突 vs 其他数据库错误
		if isUniqueViolation(err) {
			return nil, apperrors.ErrEmailAlreadyExists
		}
		return nil, apperrors.ErrInternalServer
	}

	// 8. 记录审计日志
	auditLog := &repository.AuditLog{
		UserID:    user.ID,
		Action:    "register",
		Resource:  "user",
		IPAddress: clientIP,
		UserAgent: userAgent,
		Details:   fmt.Sprintf("User registered from IP: %s", clientIP),
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		log.Printf("WARN: Failed to create audit log: %v", err)
	}

	return user, nil
}

func (s *authService) Login(email, password string) (*repository.User, string, string, error) {
	// 检查账户是否被锁定
	locked, remainingTime, err := s.lockoutManager.IsAccountLocked(email)
	if err != nil {
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
		// 用户不存在时也执行 bcrypt 比对，防止时间侧信道攻击
		_ = utils.CheckPassword("dummy-password-to-normalize-timing", "$2a$10$00000000000000000000000000000000000000000000000000000")

		s.lockoutManager.RecordFailedAttempt(email)
		remaining, _ := s.lockoutManager.GetRemainingAttempts(email)
		if remaining > 0 {
			return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
				"remaining_attempts": remaining,
			})
		}
		return nil, "", "", apperrors.ErrInvalidCredentials
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		s.lockoutManager.RecordFailedAttempt(email)
		remaining, _ := s.lockoutManager.GetRemainingAttempts(email)
		if remaining > 0 {
			return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
				"remaining_attempts": remaining,
			})
		}
		return nil, "", "", apperrors.ErrInvalidCredentials
	}

	if user.Status != "active" {
		return nil, "", "", apperrors.ErrAccountNotActive
	}

	// 登录成功，清除失败尝试
	s.lockoutManager.ClearFailedAttempts(email)

	accessToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	if storeErr := s.refreshTokenMgr.Store(user.ID, refreshToken, 30*24*time.Hour); storeErr != nil {
		log.Printf("WARN: Failed to store refresh token: %v", storeErr)
	}

	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		log.Printf("WARN: Failed to update last login: %v", err)
	}

	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "login",
		Resource: "user",
		Details:  "User logged in successfully",
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		log.Printf("WARN: Failed to create audit log: %v", err)
	}

	return user, accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshToken string) (*repository.User, string, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return nil, "", apperrors.ErrInvalidRefreshToken
	}

	storedUserID, err := s.refreshTokenMgr.Validate(refreshToken)
	if err != nil {
		return nil, "", apperrors.ErrInvalidRefreshToken
	}
	if storedUserID != 0 && storedUserID != claims.UserID {
		return nil, "", apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, "", apperrors.ErrUserNotFound
	}

	newToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", apperrors.ErrInternalServer
	}

	return user, newToken, nil
}

func (s *authService) Logout(userID uint, refreshToken string) error {
	if refreshToken != "" {
		_ = s.refreshTokenMgr.Revoke(userID, refreshToken)
	}

	auditLog := &repository.AuditLog{
		UserID:   userID,
		Action:   "logout",
		Resource: "user",
		Details:  "User logged out",
	}
	return s.auditLogRepo.Create(auditLog)
}

func (s *authService) LogoutAll(userID uint) error {
	_ = s.refreshTokenMgr.RevokeAll(userID)

	auditLog := &repository.AuditLog{
		UserID:   userID,
		Action:   "logout_all",
		Resource: "user",
		Details:  "User logged out from all devices",
	}
	return s.auditLogRepo.Create(auditLog)
}

func (s *authService) GetCurrentUser(userID uint) (*repository.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

// isUniqueViolation 检查是否为数据库唯一约束冲突错误
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "23505") // PostgreSQL unique violation error code
}
