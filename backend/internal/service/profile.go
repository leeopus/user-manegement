package service

import (
	"context"
	"strings"
	"time"

	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const nicknameCooldown = 24 * time.Hour

type ProfileService interface {
	GetProfile(userID uint) (*repository.User, error)
	UpdateProfile(userID uint, nickname, bio, avatar string, auditCtx dto.AuditContext) (*repository.User, error)
	DeleteAccount(userID uint, password string, auditCtx dto.AuditContext) error
}

type profileService struct {
	userRepo        repository.UserRepository
	auditLogger     *AuditLogger
	refreshTokenMgr interface {
		RevokeAll(userID uint) error
	}
	blacklistMgr interface {
		RevokeAllUserTokens(userID uint) error
	}
	eventPublisher *UserEventPublisher
}

func NewProfileService(
	userRepo repository.UserRepository,
	auditLogger *AuditLogger,
	refreshTokenMgr interface {
		RevokeAll(userID uint) error
	},
	blacklistMgr interface {
		RevokeAllUserTokens(userID uint) error
	},
	eventPublisher *UserEventPublisher,
) ProfileService {
	return &profileService{
		userRepo:        userRepo,
		auditLogger:     auditLogger,
		refreshTokenMgr: refreshTokenMgr,
		blacklistMgr:    blacklistMgr,
		eventPublisher:  eventPublisher,
	}
}

func (s *profileService) GetProfile(userID uint) (*repository.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

func (s *profileService) UpdateProfile(userID uint, nickname, bio, avatar string, auditCtx dto.AuditContext) (*repository.User, error) {
	nickname = strings.TrimSpace(nickname)
	bio = strings.TrimSpace(bio)

	var user *repository.User
	updateErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		var err error
		user, err = s.userRepo.FindByIDWithTx(tx, userID)
		if err != nil {
			return apperrors.ErrUserNotFound
		}

		// Nickname change: validate + cooldown + uniqueness
		if nickname != "" && nickname != user.Nickname {
			if err := utils.ValidateNickname(nickname); err != nil {
				return apperrors.ErrNicknameInvalid.WithDetails(map[string]interface{}{
					"reason": err.Error(),
				})
			}
			if user.NicknameUpdatedAt != nil && time.Since(*user.NicknameUpdatedAt) < nicknameCooldown {
				remaining := nicknameCooldown - time.Since(*user.NicknameUpdatedAt)
				return apperrors.ErrNicknameCooldown.WithDetails(map[string]interface{}{
					"remaining_hours": int(remaining.Hours()) + 1,
				})
			}
			if existing, _ := s.userRepo.FindByNicknameWithTx(tx, nickname); existing != nil {
				return apperrors.ErrNicknameAlreadyExists
			}
			user.Nickname = nickname
			now := time.Now()
			user.NicknameUpdatedAt = &now
		}

		user.Bio = bio
		if avatar != "" {
			user.Avatar = avatar
		}

		return s.userRepo.UpdateProfileWithTx(tx, user)
	})

	if updateErr != nil {
		if appErr, ok := apperrors.IsAppError(updateErr); ok {
			return nil, appErr
		}
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "update_profile", "users", map[string]interface{}{
		"user_id": userID,
	})

	s.eventPublisher.Publish(context.Background(), EventProfileUpdated, userID, UserEventData{
		Nickname: user.Nickname,
		Bio:      user.Bio,
		Avatar:   user.Avatar,
	})

	return user, nil
}

func (s *profileService) DeleteAccount(userID uint, password string, auditCtx dto.AuditContext) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return apperrors.ErrCurrentPasswordIncorrect
	}

	now := time.Now()
	user.DeletionRequestedAt = &now
	user.Status = "deleted"

	if err := s.userRepo.Update(user); err != nil {
		zap.L().Error("Failed to update user status for deletion", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	if err := s.userRepo.Delete(userID); err != nil {
		zap.L().Error("Failed to soft delete user", zap.Error(err))
		return apperrors.ErrInternalServer
	}

	if err := s.refreshTokenMgr.RevokeAll(userID); err != nil {
		zap.L().Error("Failed to revoke refresh tokens on account deletion", zap.Error(err))
	}
	if err := s.blacklistMgr.RevokeAllUserTokens(userID); err != nil {
		zap.L().Error("Failed to blacklist user tokens on account deletion", zap.Error(err))
	}

	s.auditLogger.LogSync(&auditCtx, "self_delete_account", "users", map[string]interface{}{
		"user_id": userID,
	})

	return nil
}
