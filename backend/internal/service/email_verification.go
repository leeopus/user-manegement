package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/user-system/backend/internal/email"
	"github.com/user-system/backend/internal/repository"
	apperrors "github.com/user-system/backend/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EmailVerificationService interface {
	SendVerification(userID uint, userEmail string) error
	VerifyEmail(token string) (*repository.User, error)
	ResendVerification(userID uint) error
}

type emailVerificationService struct {
	emailVerifyTokenRepo repository.EmailVerificationTokenRepository
	userRepo             repository.UserRepository
	emailService         email.EmailService
	redisClient          *redis.Client
	frontendURL          string
	eventPublisher       *UserEventPublisher
}

func NewEmailVerificationService(
	emailVerifyTokenRepo repository.EmailVerificationTokenRepository,
	userRepo repository.UserRepository,
	emailService email.EmailService,
	redisClient *redis.Client,
	frontendURL string,
	eventPublisher *UserEventPublisher,
) EmailVerificationService {
	return &emailVerificationService{
		emailVerifyTokenRepo: emailVerifyTokenRepo,
		userRepo:             userRepo,
		emailService:         emailService,
		redisClient:          redisClient,
		frontendURL:          frontendURL,
		eventPublisher:       eventPublisher,
	}
}

func (s *emailVerificationService) SendVerification(userID uint, userEmail string) error {
	if s.redisClient != nil {
		cooldownKey := fmt.Sprintf("email_verify:cooldown:%d", userID)
		exists, _ := s.redisClient.Exists(context.Background(), cooldownKey).Result()
		if exists > 0 {
			return apperrors.ErrEmailVerifyCooldown
		}
	}

	rawToken := generateVerificationToken()
	tokenHash := repository.HashResetToken(rawToken)

	token := &repository.EmailVerificationToken{
		Email:     userEmail,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserID:    userID,
	}

	if err := s.emailVerifyTokenRepo.Create(token); err != nil {
		zap.L().Error("Failed to create email verification token", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	if s.redisClient != nil {
		cooldownKey := fmt.Sprintf("email_verify:cooldown:%d", userID)
		s.redisClient.Set(context.Background(), cooldownKey, "1", 60*time.Second)
	}

	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", s.frontendURL, rawToken)
	if err := s.emailService.SendEmailVerificationEmail(userEmail, verifyLink); err != nil {
		zap.L().Error("Failed to send verification email", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	return nil
}

func (s *emailVerificationService) VerifyEmail(rawToken string) (*repository.User, error) {
	tokenHash := repository.HashResetToken(rawToken)

	verifyToken, err := s.emailVerifyTokenRepo.FindActiveByTokenHash(tokenHash)
	if err != nil {
		return nil, apperrors.ErrEmailVerificationInvalid
	}

	var user *repository.User
	verifyErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		if err := s.emailVerifyTokenRepo.MarkAsUsed(tx, verifyToken.ID); err != nil {
			return err
		}

		var err error
		user, err = s.userRepo.FindByIDWithTx(tx, verifyToken.UserID)
		if err != nil {
			return apperrors.ErrUserNotFound
		}

		if user.PendingEmail != "" {
			user.Email = user.PendingEmail
			user.PendingEmail = ""
		}

		now := time.Now()
		user.EmailVerifiedAt = &now

		return s.userRepo.UpdateProfileWithTx(tx, user)
	})

	if verifyErr != nil {
		if appErr, ok := apperrors.IsAppError(verifyErr); ok {
			return nil, appErr
		}
		return nil, apperrors.ErrInternalServer
	}

	s.eventPublisher.Publish(context.Background(), EventEmailVerified, user.ID, UserEventData{
		Email:           user.Email,
		EmailVerifiedAt: user.EmailVerifiedAt,
	})

	return user, nil
}

func (s *emailVerificationService) ResendVerification(userID uint) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	if user.EmailVerifiedAt != nil {
		return apperrors.ErrEmailAlreadyVerified
	}

	return s.SendVerification(userID, user.Email)
}

func generateVerificationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
