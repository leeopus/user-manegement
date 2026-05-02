package service

import (
	"fmt"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"github.com/user-system/backend/pkg/redis"
)

type AuthService interface {
	Register(username, email, password, clientIP, userAgent string) (*repository.User, error)
	Login(email, password string) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, error)
	Logout(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
}

type authService struct {
	userRepo          repository.UserRepository
	auditLogRepo      repository.AuditLogRepository
	lockoutManager    *auth.AccountLockoutManager
}

func NewAuthService(userRepo repository.UserRepository, auditLogRepo repository.AuditLogRepository) AuthService {
	return &authService{
		userRepo:     userRepo,
		auditLogRepo: auditLogRepo,
		lockoutManager: auth.NewAccountLockoutManager(redis.Client),
	}
}

func (s *authService) Register(username, email, password, clientIP, userAgent string) (*repository.User, error) {
	// 1. 验证用户名格式
	if err := utils.ValidateUsername(username); err != nil {
		return nil, err
	}

	// 2. 验证邮箱格式
	if err := utils.ValidateEmail(email); err != nil {
		return nil, err
	}

	// 3. 验证密码强度
	strength, err := utils.ValidatePassword(password, username)
	if err != nil {
		return nil, err
	}

	// 密码强度至少为中等
	if strength < utils.PasswordFair {
		return nil, apperrors.ErrPasswordTooWeak
	}

	// 4. 检查是否为一次性邮箱
	if utils.IsDisposableEmail(email) {
		return nil, apperrors.ErrDisposableEmail
	}

	// 5. 检查邮箱是否已存在
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, apperrors.ErrEmailAlreadyExists
	}

	// 6. 检查用户名是否已存在
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, apperrors.ErrUsernameAlreadyExists
	}

	// 7. 加密密码
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, apperrors.ErrInternalServer
	}

	// 8. 创建用户
	user := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active", // 可以改为 "pending" 需要邮箱验证
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	// 9. 记录审计日志（包含 IP 和 User-Agent）
	auditLog := &repository.AuditLog{
		UserID:    user.ID,
		Action:    "register",
		Resource:  "user",
		IPAddress: clientIP,
		UserAgent: userAgent,
		Details:   fmt.Sprintf("User registered from IP: %s", clientIP),
	}
	s.auditLogRepo.Create(auditLog)

	return user, nil
}

func (s *authService) Login(email, password string) (*repository.User, string, string, error) {
	// 检查账户是否被锁定
	locked, remainingTime, err := s.lockoutManager.IsAccountLocked(email)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}
	if locked {
		// 返回账户锁定错误，带详情
		minutes := int(remainingTime.Minutes())
		return nil, "", "", apperrors.ErrAccountLocked.WithDetails(map[string]interface{}{
			"remaining_minutes": minutes,
		})
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// 记录失败尝试
		s.lockoutManager.RecordFailedAttempt(email)

		// 获取剩余尝试次数
		remaining, _ := s.lockoutManager.GetRemainingAttempts(email)
		if remaining > 0 {
			return nil, "", "", apperrors.ErrInvalidCredentials.WithDetails(map[string]interface{}{
				"remaining_attempts": remaining,
			})
		}

		return nil, "", "", apperrors.ErrInvalidCredentials
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		// 记录失败尝试
		s.lockoutManager.RecordFailedAttempt(email)

		// 获取剩余尝试次数
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

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", apperrors.ErrInternalServer
	}

	// Update last login
	s.userRepo.UpdateLastLogin(user.ID)

	// Create audit log
	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "login",
		Resource: "user",
		Details:  "User logged in successfully",
	}
	s.auditLogRepo.Create(auditLog)

	return user, accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshToken string) (*repository.User, string, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
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

func (s *authService) Logout(userID uint) error {
	// Create audit log
	auditLog := &repository.AuditLog{
		UserID:   userID,
		Action:   "logout",
		Resource: "user",
		Details:  "User logged out",
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
