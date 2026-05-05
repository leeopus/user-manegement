package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/repository"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
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
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		auditLogRepo:   auditLogRepo,
		emailService:   emailService,
		tokenGenerator: generateSecureToken,
		frontendURL:    frontendURL,
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

// RequestPasswordReset 请求密码重置
func (s *passwordResetService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// 不透露用户是否存在，返回成功
		return nil
	}

	token, err := s.tokenGenerator()
	if err != nil {
		return apperrors.ErrInternalServer
	}

	resetToken := &repository.PasswordResetToken{
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Used:      false,
		UserID:    user.ID,
	}

	if err := s.tokenRepo.Create(resetToken); err != nil {
		log.Printf("ERROR: Failed to save reset token: %v", err)
		return apperrors.ErrInternalServer
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	if err := s.emailService.SendPasswordResetEmail(email, resetLink); err != nil {
		log.Printf("ERROR: Failed to send reset email: %v", err)
		return apperrors.ErrInternalServer
	}

	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "password_reset_requested",
		Resource: "user",
		Details:  "Password reset requested via email",
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		log.Printf("WARN: Failed to create audit log: %v", err)
	}

	return nil
}

// ValidateResetToken 验证重置令牌
func (s *passwordResetService) ValidateResetToken(token string) (bool, error) {
	resetToken, err := s.tokenRepo.FindByToken(token)
	if err != nil {
		return false, nil
	}

	if resetToken.Used {
		return false, nil
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

// ResetPassword 重置密码
func (s *passwordResetService) ResetPassword(token, newPassword string) error {
	resetToken, err := s.tokenRepo.FindByToken(token)
	if err != nil {
		return apperrors.ErrInvalidResetToken
	}

	if resetToken.Used {
		return apperrors.ErrResetTokenAlreadyUsed
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return apperrors.ErrResetTokenExpired
	}

	user, err := s.userRepo.FindByEmail(resetToken.Email)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// 使用与注册相同的密码验证规则
	if _, err := utils.ValidatePassword(newPassword, user.Username); err != nil {
		return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 使用与注册相同的密码加密方法
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		return apperrors.ErrInternalServer
	}

	user.PasswordHash = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		log.Printf("ERROR: Failed to update password: %v", err)
		return apperrors.ErrInternalServer
	}

	if err := s.tokenRepo.MarkAsUsed(token); err != nil {
		log.Printf("WARN: Failed to mark token as used: %v", err)
	}

	if err := s.emailService.SendPasswordChangedNotification(user.Email); err != nil {
		log.Printf("WARN: Failed to send password change notification: %v", err)
	}

	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "password_reset_completed",
		Resource: "user",
		Details:  "Password reset via email link",
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		log.Printf("WARN: Failed to create audit log: %v", err)
	}

	return nil
}
