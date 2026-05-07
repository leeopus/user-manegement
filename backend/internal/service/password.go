package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// PasswordResetService 密码重置服务
type PasswordResetService interface {
	RequestPasswordReset(email string) error
	ResetPassword(token, newPassword string) error
	ValidateResetToken(token string) (bool, error)
}

type passwordResetService struct {
	userRepo            repository.UserRepository
	tokenRepo           repository.PasswordResetTokenRepository
	passwordHistoryRepo repository.PasswordHistoryRepository
	auditLogger         *AuditLogger
	emailService        email.EmailService
	tokenGenerator      func() (string, error)
	frontendURL         string
	refreshTokenMgr     *auth.RefreshTokenManager
	blacklistMgr        *auth.TokenBlacklistManager
	redisClient         *redis.Client
}

func NewPasswordResetService(
	userRepo repository.UserRepository,
	tokenRepo repository.PasswordResetTokenRepository,
	passwordHistoryRepo repository.PasswordHistoryRepository,
	auditLogger *AuditLogger,
	emailService email.EmailService,
	frontendURL string,
	refreshTokenMgr *auth.RefreshTokenManager,
	blacklistMgr *auth.TokenBlacklistManager,
	redisClient *redis.Client,
) PasswordResetService {
	return &passwordResetService{
		userRepo:            userRepo,
		tokenRepo:           tokenRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		auditLogger:         auditLogger,
		emailService:        emailService,
		tokenGenerator:      generateSecureToken,
		frontendURL:         frontendURL,
		refreshTokenMgr:     refreshTokenMgr,
		blacklistMgr:        blacklistMgr,
		redisClient:         redisClient,
	}
}

func generateSecureToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// RequestPasswordReset 请求密码重置（常量时间响应，防止时序侧信道枚举邮箱）
func (s *passwordResetService) RequestPasswordReset(emailAddr string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(emailAddr))

	// 邮箱维度冷却检查：同一邮箱每 60 秒只能请求一次
	if s.redisClient != nil {
		cooldownKey := fmt.Sprintf("pw_reset:cooldown:%s", normalizedEmail)
		ctx := context.Background()
		ttl, err := s.redisClient.TTL(ctx, cooldownKey).Result()
		if err == nil && ttl > 0 {
			return nil // 冷却中，静默返回成功（防止枚举）
		}
	}

	user, err := s.userRepo.FindByEmail(emailAddr)

	if err != nil {
		dummyToken, _ := generateSecureToken()
		_ = repository.HashResetToken(dummyToken)
		s.tokenRepo.TouchDummy()
		// 不存在的邮箱也设冷却，保持常量时间
		if s.redisClient != nil {
			s.redisClient.Set(context.Background(), fmt.Sprintf("pw_reset:cooldown:%s", normalizedEmail), "1", 60*time.Second)
		}
		return nil
	}

	token, err := s.tokenGenerator()
	if err != nil {
		return apperrors.ErrInternalServer
	}

	resetToken := &repository.PasswordResetToken{
		Email:     emailAddr,
		TokenHash: repository.HashResetToken(token),
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
		UserID:    user.ID,
	}

	if err := s.tokenRepo.Create(resetToken); err != nil {
		zap.L().Error("Failed to save reset token", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	if err := s.emailService.SendPasswordResetEmail(emailAddr, resetLink); err != nil {
		zap.L().Error("Failed to send reset email", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{UserID: user.ID}, "password_reset_requested", "user", map[string]interface{}{
		"email": emailAddr,
	})

	// 设置邮箱冷却，60 秒内不可重复请求
	if s.redisClient != nil {
		s.redisClient.Set(context.Background(), fmt.Sprintf("pw_reset:cooldown:%s", normalizedEmail), "1", 60*time.Second)
	}

	return nil
}

func (s *passwordResetService) ValidateResetToken(token string) (bool, error) {
	tokenHash := repository.HashResetToken(token)
	resetToken, err := s.tokenRepo.FindByTokenHash(tokenHash)
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

// ResetPassword 重置密码（事务保护防止竞态）
func (s *passwordResetService) ResetPassword(token, newPassword string) error {
	tokenHash := repository.HashResetToken(token)

	txErr := s.tokenRepo.Transaction(func(tx *gorm.DB) error {
		resetToken, err := s.tokenRepo.FindByTokenHashForUpdate(tx, tokenHash)
		if err != nil {
			return apperrors.ErrInvalidResetToken
		}

		if resetToken.Used {
			return apperrors.ErrResetTokenAlreadyUsed
		}

		if time.Now().After(resetToken.ExpiresAt) {
			return apperrors.ErrResetTokenExpired
		}

		if err := s.tokenRepo.MarkAsUsedByHash(tx, tokenHash); err != nil {
			return apperrors.ErrInternalServer
		}

		user, err := s.userRepo.FindByIDWithTx(tx, resetToken.UserID)
		if err != nil {
			return apperrors.ErrUserNotFound
		}

		if _, err := utils.ValidatePassword(newPassword, user.Username); err != nil {
			return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
				"reason": err.Error(),
			})
		}

		if utils.CheckPassword(newPassword, user.PasswordHash) {
			return apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
				"reason": "new password must be different from current password",
			})
		}

		histories, histErr := s.passwordHistoryRepo.FindByUserID(user.ID, 5)
		if histErr != nil {
			zap.L().Error("Failed to check password history, aborting reset for security",
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
			zap.L().Error("Failed to hash password", zap.Error(err))
			return apperrors.ErrInternalServer
		}

		user.PasswordHash = hashedPassword
		now := time.Now()
		user.PasswordChangedAt = &now
		if err := tx.Save(user).Error; err != nil {
			return apperrors.ErrInternalServer
		}

		if err := s.passwordHistoryRepo.CreateWithTx(tx, &repository.PasswordHistory{
			UserID:       user.ID,
			PasswordHash: hashedPassword,
		}); err != nil {
			zap.L().Error("Failed to save password history", zap.Error(err))
		}

		return nil
	})

	if txErr != nil {
		if appErr, ok := apperrors.IsAppError(txErr); ok {
			return appErr
		}
		return apperrors.ErrInternalServer
	}

	resetToken, _ := s.tokenRepo.FindByTokenHash(tokenHash)
	if resetToken != nil {
		user, _ := s.userRepo.FindByID(resetToken.UserID)
		if user != nil {
			_ = s.refreshTokenMgr.RevokeAll(user.ID)
			_ = s.blacklistMgr.RevokeAllUserTokens(user.ID)
			_ = s.emailService.SendPasswordChangedNotification(user.Email)
			_ = s.passwordHistoryRepo.CleanupOld(user.ID, 5)
			s.auditLogger.Log(&dto.AuditContext{UserID: user.ID}, "password_reset_completed", "user", map[string]interface{}{
				"email": user.Email,
			})
		}
	}

	return nil
}
