package service

import (
	"strings"
	"time"

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
	GetUser(id uint, currentUserID uint) (*repository.User, error)
	UpdateUser(id uint, username, email string, currentUserID uint, auditCtx dto.AuditContext) (*repository.User, error)
	UpdateUserStatus(id, currentUserID uint, status string, auditCtx dto.AuditContext) error
	DeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error
	HardDeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error
	ListUsers(offset, pageSize int, filters repository.UserFilters) ([]repository.User, int64, error)
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
		if _, err := s.userRepo.FindByUsernameWithTx(tx, username); err == nil {
			return apperrors.ErrUsernameAlreadyExists
		}
		if _, err := s.userRepo.FindByEmailWithTx(tx, email); err == nil {
			return apperrors.ErrEmailAlreadyExists
		}

		now := time.Now()
		user = &repository.User{
			Username:          username,
			Email:             email,
			PasswordHash:      passwordHash,
			Status:            StatusActive,
			PasswordChangedAt: &now,
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

func (s *userService) GetUser(id uint, currentUserID uint) (*repository.User, error) {
	if !s.isOwnerOrAdmin(id, currentUserID) {
		return nil, apperrors.ErrUserNotFound
	}
	return s.userRepo.FindByID(id)
}

func (s *userService) UpdateUser(id uint, username, email string, currentUserID uint, auditCtx dto.AuditContext) (*repository.User, error) {
	if !s.isOwnerOrAdmin(id, currentUserID) {
		return nil, apperrors.ErrUserNotFound
	}
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
		user, err = s.userRepo.FindByIDWithTx(tx, id)
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

func (s *userService) UpdateUserStatus(id, currentUserID uint, status string, auditCtx dto.AuditContext) error {
	if id == currentUserID {
		return apperrors.ErrCannotDeleteSelf
	}

	if status != StatusActive && status != StatusDisabled {
		return apperrors.ErrInvalidStatus.WithDetails(map[string]interface{}{
			"reason": "status must be 'active' or 'disabled'",
		})
	}

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	oldStatus := user.Status
	if oldStatus == status {
		return nil
	}

	if err := s.userRepo.UpdateStatus(id, status); err != nil {
		return apperrors.ErrInternalServer
	}

	if status == StatusDisabled {
		if err := s.refreshTokenMgr.RevokeAll(id); err != nil {
			zap.L().Warn("Failed to revoke refresh tokens on disable", zap.Uint("user_id", id), zap.Error(err))
		}
		if err := s.blacklistMgr.RevokeAllUserTokens(id); err != nil {
			zap.L().Warn("Failed to blacklist user tokens on disable", zap.Uint("user_id", id), zap.Error(err))
		}
	}

	_ = s.rbacCache.InvalidateUserRoles(id)

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "update_user_status", "user", map[string]interface{}{
		"target_id":   id,
		"old_status":  oldStatus,
		"new_status":  status,
	})

	return nil
}

func (s *userService) DeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error {
	if id == currentUserID {
		return apperrors.ErrCannotDeleteSelf
	}

	if _, err := s.userRepo.FindByID(id); err != nil {
		return apperrors.ErrUserNotFound
	}

	// 在事务中执行状态更新 + 软删除，保证原子性
	if err := s.userRepo.Transaction(func(tx *gorm.DB) error {
		user, err := s.userRepo.FindByIDWithTx(tx, id)
		if err != nil {
			return apperrors.ErrUserNotFound
		}
		user.Status = "deleted"
		if err := s.userRepo.UpdateWithTx(tx, user); err != nil {
			return apperrors.ErrInternalServer
		}
		return s.userRepo.DeleteWithTx(tx, id)
	}); err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok {
			return appErr
		}
		return apperrors.ErrInternalServer
	}

	// 事务成功后执行副作用操作（Redis、审计）
	// 先设置黑名单状态，即使后续操作部分失败也能阻止已删除用户的访问
	if err := s.blacklistMgr.SetUserStatus(id, "deleted"); err != nil {
		zap.L().Error("CRITICAL: failed to blacklist deleted user, access may remain valid", zap.Uint("user_id", id), zap.Error(err))
	}

	if err := s.refreshTokenMgr.RevokeAll(id); err != nil {
		zap.L().Warn("Failed to revoke refresh tokens during soft delete", zap.Uint("user_id", id), zap.Error(err))
	}
	if err := s.blacklistMgr.RevokeAllUserTokens(id); err != nil {
		zap.L().Warn("Failed to blacklist user tokens during soft delete", zap.Uint("user_id", id), zap.Error(err))
	}
	_ = s.rbacCache.InvalidateUserRoles(id)

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "delete_user", "user", map[string]interface{}{
		"target_id": id,
	})

	return nil
}

func (s *userService) HardDeleteUser(id uint, currentUserID uint, auditCtx dto.AuditContext) error {
	if id == currentUserID {
		return apperrors.ErrCannotDeleteSelf
	}

	// 使用 Unscoped 查询，确保能找到已软删除的用户
	user, err := s.userRepo.FindByIDUnscoped(id)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// 先执行数据库删除，确保主操作成功
	if err := s.userRepo.HardDelete(id); err != nil {
		return apperrors.ErrInternalServer
	}

	// 数据库操作成功后执行副作用操作
	if err := s.blacklistMgr.SetUserStatus(id, "deleted"); err != nil {
		zap.L().Error("CRITICAL: failed to blacklist hard-deleted user, access may remain valid", zap.Uint("user_id", id), zap.Error(err))
	}
	if err := s.refreshTokenMgr.RevokeAll(id); err != nil {
		zap.L().Warn("Failed to revoke refresh tokens during hard delete", zap.Uint("user_id", id), zap.Error(err))
	}
	if err := s.blacklistMgr.RevokeAllUserTokens(id); err != nil {
		zap.L().Warn("Failed to blacklist user tokens during hard delete", zap.Uint("user_id", id), zap.Error(err))
	}
	_ = s.rbacCache.InvalidateUserRoles(id)

	s.auditLogger.Log(&dto.AuditContext{
		UserID:    auditCtx.UserID,
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
	}, "hard_delete_user", "user", map[string]interface{}{
		"target_id":       id,
		"target_username": user.Username,
		"target_email":    user.Email,
	})

	return nil
}

func (s *userService) ListUsers(offset, pageSize int, filters repository.UserFilters) ([]repository.User, int64, error) {
	return s.userRepo.List(offset, pageSize, filters)
}

func (s *userService) AssignRole(userID, roleID uint, auditCtx dto.AuditContext) error {
	if err := s.userRepo.Transaction(func(tx *gorm.DB) error {
		if _, err := s.userRepo.FindByIDWithTx(tx, userID); err != nil {
			return apperrors.ErrUserNotFound
		}
		if _, err := s.roleRepo.FindByIDWithTx(tx, roleID); err != nil {
			return apperrors.ErrRoleNotFound
		}
		return s.roleRepo.AssignRoleToUser(userID, roleID)
	}); err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok {
			return appErr
		}
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
	if err := s.userRepo.Transaction(func(tx *gorm.DB) error {
		if _, err := s.userRepo.FindByIDWithTx(tx, userID); err != nil {
			return apperrors.ErrUserNotFound
		}
		return s.roleRepo.RemoveRoleFromUser(userID, roleID)
	}); err != nil {
		if appErr, ok := apperrors.IsAppError(err); ok {
			return appErr
		}
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

// isOwnerOrAdmin checks if currentUserID is the resource owner or has admin role / user management permission.
func (s *userService) isOwnerOrAdmin(resourceUserID, currentUserID uint) bool {
	if resourceUserID == currentUserID {
		return true
	}

	// 先查 Redis 缓存
	roles, err := s.rbacCache.GetUserRoles(currentUserID)
	if err == nil && roles != nil {
		return s.hasAdminOrUserManagePermission(roles)
	}

	// 缓存 miss：回源 DB 查询
	dbRoles, dbErr := s.userRepo.GetUserRoles(currentUserID)
	if dbErr != nil {
		return false
	}

	for _, r := range dbRoles {
		if r.Code == RoleAdmin {
			return true
		}
		for _, p := range r.Permissions {
			if p.Code == "user:manage" || p.Code == "user:write" {
				return true
			}
		}
	}
	return false
}

func (s *userService) hasAdminOrUserManagePermission(roles []auth.RoleData) bool {
	for _, role := range roles {
		if role.Code == RoleAdmin {
			return true
		}
		for _, p := range role.Permissions {
			if p.Code == "user:manage" || p.Code == "user:write" {
				return true
			}
		}
	}
	return false
}
