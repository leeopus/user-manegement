package service

import (
	"errors"
	"fmt"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/utils"
)

type AuthService interface {
	Register(username, email, password, clientIP, userAgent string) (*repository.User, error)
	Login(email, password string) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, error)
	Logout(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
}

type authService struct {
	userRepo     repository.UserRepository
	auditLogRepo repository.AuditLogRepository
}

func NewAuthService(userRepo repository.UserRepository, auditLogRepo repository.AuditLogRepository) AuthService {
	return &authService{
		userRepo:     userRepo,
		auditLogRepo: auditLogRepo,
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
		return nil, errors.New("密码强度不足，请使用更复杂的密码")
	}

	// 4. 检查是否为一次性邮箱
	if utils.IsDisposableEmail(email) {
		return nil, errors.New("不支持一次性邮箱注册")
	}

	// 5. 检查邮箱是否已存在
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, errors.New("该邮箱已被注册")
	}

	// 6. 检查用户名是否已存在
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, errors.New("该用户名已被使用")
	}

	// 7. 加密密码
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	// 8. 创建用户
	user := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active", // 可以改为 "pending" 需要邮箱验证
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("创建用户失败")
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
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, "", "", errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, "", "", errors.New("account is not active")
	}

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", err
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
		return nil, "", errors.New("invalid refresh token")
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, "", errors.New("user not found")
	}

	newToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", err
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
	return s.userRepo.FindByID(userID)
}
