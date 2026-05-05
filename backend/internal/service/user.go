package service

import (
	"strings"

	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"github.com/user-system/backend/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(username, email, password string, auditCtx dto.AuditContext) (*repository.User, error)
	GetUser(id uint) (*repository.User, error)
	UpdateUser(id uint, username, email string, auditCtx dto.AuditContext) (*repository.User, error)
	DeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error
	ListUsers(offset, pageSize int) ([]repository.User, int64, error)
	AssignRole(userID, roleID uint, auditCtx dto.AuditContext) error
	RemoveRole(userID, roleID uint, auditCtx dto.AuditContext) error
}

type userService struct {
	userRepo        repository.UserRepository
	roleRepo        repository.RoleRepository
	auditLogger     *AuditLogger
	rbacCache       *auth.RBACCacheManager
	refreshTokenMgr *auth.RefreshTokenManager
	blacklistMgr    *auth.TokenBlacklistManager
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auditLogger *AuditLogger, rbacCache *auth.RBACCacheManager, blacklistMgr *auth.TokenBlacklistManager, refreshTokenMgr *auth.RefreshTokenManager) UserService {
	return &userService{
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		auditLogger:     auditLogger,
		rbacCache:       rbacCache,
		refreshTokenMgr: refreshTokenMgr,
		blacklistMgr:    blacklistMgr,
	}
}

func (s *userService) CreateUser(username, email, password string, auditCtx dto.AuditContext) (*repository.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	if err := utils.ValidateUsername(username); err != nil {
		return nil, apperrors.ErrUsernameInvalidPattern.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if _, err := utils.ValidatePassword(password, username); err != nil {
		return nil, apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, apperrors.ErrInternalServer
	}

	var user *repository.User
	createErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		// 事务内查询+插入，利用唯一约束防止并发重复
		if _, err := s.userRepo.FindByUsernameWithTx(tx, username); err == nil {
			return apperrors.ErrUsernameAlreadyExists
		}
		if _, err := s.userRepo.FindByEmailWithTx(tx, email); err == nil {
			return apperrors.ErrEmailAlreadyExists
		}

		user = &repository.User{
			Username:     username,
			Email:        email,
			PasswordHash: passwordHash,
			Status:       StatusActive,
		}
		return s.userRepo.CreateWithTx(tx, user)
	})

	if createErr != nil {
		if appErr, ok := apperrors.IsAppError(createErr); ok {
			return nil, appErr
		}
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "create_user", "user", map[string]interface{}{
		"target_username": username,
		"target_email":   email,
	})

	return user, nil
}

func (s *userService) GetUser(id uint) (*repository.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) UpdateUser(id uint, username, email string, auditCtx dto.AuditContext) (*repository.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	username = strings.TrimSpace(username)

	if err := utils.ValidateUsername(username); err != nil {
		return nil, apperrors.ErrUsernameInvalidPattern.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	var user *repository.User
	updateErr := s.userRepo.Transaction(func(tx *gorm.DB) error {
		var err error
		user, err = s.userRepo.FindByID(id)
		if err != nil {
			return apperrors.ErrUserNotFound
		}

		if username != user.Username {
			if existing, _ := s.userRepo.FindByUsernameWithTx(tx, username); existing != nil {
				return apperrors.ErrUsernameAlreadyExists
			}
		}

		if email != user.Email {
			if existing, _ := s.userRepo.FindByEmailWithTx(tx, email); existing != nil {
				return apperrors.ErrEmailAlreadyExists
			}
		}

		user.Username = username
		user.Email = email
		return s.userRepo.UpdateWithTx(tx, user)
	})

	if updateErr != nil {
		if appErr, ok := apperrors.IsAppError(updateErr); ok {
			return nil, appErr
		}
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "update_user", "user", map[string]interface{}{
		"target_id":       id,
		"target_username": username,
		"target_email":    email,
	})

	return user, nil
}

func (s *userService) DeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error {
	if id == currentUserID {
		return apperrors.ErrCannotDeleteSelf
	}

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// 先将状态设为 deleted，防止删除过程中新会话建立
	user.Status = "deleted"
	if err := s.userRepo.Update(user); err != nil {
		return apperrors.ErrInternalServer
	}

	// 同步 Redis 用户状态缓存，立即阻断后续请求
	_ = s.blacklistMgr.SetUserStatus(id, "deleted")

	// 吊销被删除用户的所有 refresh token 和 access token
	_ = s.refreshTokenMgr.RevokeAll(id)
	_ = s.blacklistMgr.RevokeAllUserTokens(id)

	// 清除 RBAC 缓存
	_ = s.rbacCache.InvalidateUserRoles(id)

	// 执行软删除（BeforeDelete hook 会自动清除唯一约束字段）
	if err := s.userRepo.Delete(id); err != nil {
		return apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "delete_user", "user", map[string]interface{}{
		"target_id": id,
	})

	return nil
}

func (s *userService) ListUsers(offset, pageSize int) ([]repository.User, int64, error) {
	return s.userRepo.List(offset, pageSize)
}

func (s *userService) AssignRole(userID, roleID uint, auditCtx dto.AuditContext) error {
	if _, err := s.userRepo.FindByID(userID); err != nil {
		return apperrors.ErrUserNotFound
	}
	if _, err := s.roleRepo.FindByID(roleID); err != nil {
		return apperrors.ErrRoleNotFound
	}

	if err := s.roleRepo.AssignRoleToUser(userID, roleID); err != nil {
		return apperrors.ErrInternalServer
	}

	if err := s.rbacCache.InvalidateUserRoles(userID); err != nil {
		zap.L().Warn("Failed to invalidate RBAC cache after role assignment", zap.Error(err))
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "assign_role", "user", map[string]interface{}{
		"target_user_id": userID,
		"role_id":        roleID,
	})

	return nil
}

func (s *userService) RemoveRole(userID, roleID uint, auditCtx dto.AuditContext) error {
	if _, err := s.userRepo.FindByID(userID); err != nil {
		return apperrors.ErrUserNotFound
	}

	if err := s.roleRepo.RemoveRoleFromUser(userID, roleID); err != nil {
		return apperrors.ErrInternalServer
	}

	if err := s.rbacCache.InvalidateUserRoles(userID); err != nil {
		zap.L().Warn("Failed to invalidate RBAC cache after role removal", zap.Error(err))
	}

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "remove_role", "user", map[string]interface{}{
		"target_user_id": userID,
		"role_id":        roleID,
	})

	return nil
}
