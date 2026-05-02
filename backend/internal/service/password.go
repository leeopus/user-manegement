package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/repository"
	apperrors "github.com/user-system/backend/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// PasswordResetService 密码重置服务
type PasswordResetService interface {
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
	ValidateResetToken(token string) (bool, error)
}

type passwordResetService struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.PasswordResetTokenRepository
	auditLogRepo   repository.AuditLogRepository
	emailService   email.EmailService
	tokenGenerator func() (string, error)
	frontendURL    string
}

func NewPasswordResetService(
	userRepo repository.UserRepository,
	tokenRepo repository.PasswordResetTokenRepository,
	auditLogRepo repository.AuditLogRepository,
	emailService email.EmailService,
	frontendURL string,
) PasswordResetService {
	return &passwordResetService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		auditLogRepo: auditLogRepo,
		emailService: emailService,
		tokenGenerator: generateSecureToken,
		frontendURL:  frontendURL,
	}
}

// generateSecureToken 生成安全的随机令牌
func generateSecureToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

type RequestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// RequestPasswordReset 请求密码重置
func (s *passwordResetService) RequestPasswordReset(email string) error {
	// 检查用户是否存在
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// 不透露用户是否存在，返回成功
		return nil
	}

	// 生成重置令牌
	token, err := s.tokenGenerator()
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// 创建重置令牌记录（1小时有效期）
	resetToken := &repository.PasswordResetToken{
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      false,
		UserID:    user.ID,
	}

	if err := s.tokenRepo.Create(resetToken); err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	// 生成重置链接（使用配置的前端URL，不包含语言路径）
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	// 发送重置邮件
	if err := s.emailService.SendPasswordResetEmail(email, resetLink); err != nil {
		return fmt.Errorf("failed to send reset email: %w", err)
	}

	// 记录审计日志
	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "password_reset_requested",
		Resource: "user",
		Details:  "Password reset requested via email",
	}
	s.auditLogRepo.Create(auditLog)

	return nil
}

// ValidateResetToken 验证重置令牌
func (s *passwordResetService) ValidateResetToken(token string) (bool, error) {
	resetToken, err := s.tokenRepo.FindByToken(token)
	if err != nil {
		return false, nil // 不透露具体错误
	}

	// 检查是否已使用
	if resetToken.Used {
		return false, nil
	}

	// 检查是否过期
	if time.Now().After(resetToken.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// ResetPassword 重置密码
func (s *passwordResetService) ResetPassword(token, newPassword string) error {
	// 查找并验证令牌
	resetToken, err := s.tokenRepo.FindByToken(token)
	if err != nil {
		return apperrors.ErrInvalidResetToken
	}

	// 检查是否已使用
	if resetToken.Used {
		return apperrors.ErrResetTokenAlreadyUsed
	}

	// 检查是否过期
	if time.Now().After(resetToken.ExpiresAt) {
		return apperrors.ErrResetTokenExpired
	}

	// 查找用户
	user, err := s.userRepo.FindByEmail(resetToken.Email)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// 验证密码强度
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 更新用户密码
	user.PasswordHash = string(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 标记令牌已使用
	if err := s.tokenRepo.MarkAsUsed(token); err != nil {
		// 记录错误但不中断流程
		fmt.Printf("Warning: failed to mark token as used: %v\n", err)
	}

	// 发送密码更改通知
	if err := s.emailService.SendPasswordChangedNotification(user.Email); err != nil {
		// 记录错误但不中断流程
		fmt.Printf("Warning: failed to send password change notification: %v\n", err)
	}

	// 记录审计日志
	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "password_reset_completed",
		Resource: "user",
		Details:  "Password reset via email link",
	}
	s.auditLogRepo.Create(auditLog)

	return nil
}

// validatePasswordStrength 验证密码强度
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return apperrors.ErrPasswordTooWeak
	}

	// 可以添加更多密码强度验证规则
	// 例如：包含大小写字母、数字、特殊字符等

	return nil
}
