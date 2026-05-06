package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"go.uber.org/zap"
)

var roleCodePattern = regexp.MustCompile(`^[a-z0-9_]{2,50}$`)

type RoleService interface {
	CreateRole(name, code, description string, auditCtx dto.AuditContext) (*repository.Role, error)
	GetRole(id uint) (*repository.Role, error)
	UpdateRole(id uint, name, code, description string, auditCtx dto.AuditContext) (*repository.Role, error)
	DeleteRole(id uint, auditCtx dto.AuditContext) error
	ListRoles(offset, pageSize int) ([]repository.Role, int64, error)
	AssignRolePermission(roleID, permissionID uint, auditCtx dto.AuditContext) error
	RemoveRolePermission(roleID, permissionID uint, auditCtx dto.AuditContext) error
}

type roleService struct {
	roleRepo       repository.RoleRepository
	permissionRepo repository.PermissionRepository
	auditLogger    *AuditLogger
	rbacCache      *auth.RBACCacheManager
}

func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository, auditLogger *AuditLogger, rbacCache *auth.RBACCacheManager) RoleService {
	return &roleService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
		auditLogger:    auditLogger,
		rbacCache:      rbacCache,
	}
}

func validateRoleInput(name, code string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return fmt.Errorf("role name must be 1-50 characters")
	}
	if !roleCodePattern.MatchString(strings.ToLower(strings.TrimSpace(code))) {
		return fmt.Errorf("role code must be lowercase alphanumeric with underscores, 2-50 characters")
	}
	return nil
}

func (s *roleService) CreateRole(name, code, description string, auditCtx dto.AuditContext) (*repository.Role, error) {
	if err := validateRoleInput(name, code); err != nil {
		return nil, apperrors.ErrRoleValidation.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	code = strings.ToLower(strings.TrimSpace(code))

	if _, err := s.roleRepo.FindByCode(code); err == nil {
		return nil, apperrors.ErrRoleCodeExists.WithDetails(map[string]interface{}{
			"reason": fmt.Sprintf("role code %q already exists", code),
		})
	}

	role := &repository.Role{
		Name:        strings.TrimSpace(name),
		Code:        code,
		Description: strings.TrimSpace(description),
	}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "create_role", "role", map[string]interface{}{
		"role_id":   role.ID,
		"role_name": name,
		"role_code": code,
	})

	return role, nil
}

func (s *roleService) GetRole(id uint) (*repository.Role, error) {
	return s.roleRepo.FindByID(id)
}

func (s *roleService) UpdateRole(id uint, name, code, description string, auditCtx dto.AuditContext) (*repository.Role, error) {
	if err := validateRoleInput(name, code); err != nil {
		return nil, apperrors.ErrRoleValidation.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.ErrRoleNotFound
	}

	code = strings.ToLower(strings.TrimSpace(code))

	// Check uniqueness if code is changing
	if code != role.Code {
		if existing, _ := s.roleRepo.FindByCode(code); existing != nil {
			return nil, apperrors.ErrRoleCodeExists.WithDetails(map[string]interface{}{
				"reason": fmt.Sprintf("role code %q already exists", code),
			})
		}
	}

	role.Name = strings.TrimSpace(name)
	role.Code = code
	role.Description = strings.TrimSpace(description)

	if err := s.roleRepo.Update(role); err != nil {
		// Handle race condition: concurrent request created same code
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperrors.ErrRoleCodeExists.WithDetails(map[string]interface{}{
				"reason": fmt.Sprintf("role code %q already exists", code),
			})
		}
		return nil, apperrors.ErrInternalServer
	}

	// DB 更新成功后再清缓存
	s.invalidateCacheForRole(id)

	s.auditLogger.Log(&auditCtx, "update_role", "role", map[string]interface{}{
		"role_id":   id,
		"role_name": name,
		"role_code": code,
	})

	return role, nil
}

func (s *roleService) DeleteRole(id uint, auditCtx dto.AuditContext) error {
	userIDs, err := s.roleRepo.GetUserIDsByRoleID(id)
	if err != nil {
		zap.L().Warn("Failed to get user IDs for RBAC cache invalidation", zap.Uint("roleID", id), zap.Error(err))
	}

	if err := s.roleRepo.Delete(id); err != nil {
		return err
	}

	s.invalidateCacheForUserIDs(userIDs)

	s.auditLogger.Log(&auditCtx, "delete_role", "role", map[string]interface{}{
		"role_id": id,
	})

	return nil
}

func (s *roleService) ListRoles(offset, pageSize int) ([]repository.Role, int64, error) {
	return s.roleRepo.List(offset, pageSize)
}

func (s *roleService) AssignRolePermission(roleID, permissionID uint, auditCtx dto.AuditContext) error {
	if _, err := s.roleRepo.FindByID(roleID); err != nil {
		return apperrors.ErrRoleNotFound
	}
	if _, err := s.permissionRepo.FindByID(permissionID); err != nil {
		return apperrors.ErrPermissionNotFound
	}

	if err := s.roleRepo.AssignPermission(roleID, permissionID); err != nil {
		return apperrors.ErrInternalServer
	}

	s.invalidateCacheForRole(roleID)

	s.auditLogger.Log(&auditCtx, "assign_permission", "role", map[string]interface{}{
		"role_id":       roleID,
		"permission_id": permissionID,
	})

	return nil
}

func (s *roleService) RemoveRolePermission(roleID, permissionID uint, auditCtx dto.AuditContext) error {
	if _, err := s.roleRepo.FindByID(roleID); err != nil {
		return apperrors.ErrRoleNotFound
	}

	if err := s.roleRepo.RemovePermission(roleID, permissionID); err != nil {
		return apperrors.ErrInternalServer
	}

	s.invalidateCacheForRole(roleID)

	s.auditLogger.Log(&auditCtx, "remove_permission", "role", map[string]interface{}{
		"role_id":       roleID,
		"permission_id": permissionID,
	})

	return nil
}

func (s *roleService) invalidateCacheForRole(roleID uint) {
	userIDs, err := s.roleRepo.GetUserIDsByRoleID(roleID)
	if err != nil {
		zap.L().Warn("Failed to get user IDs for RBAC cache invalidation", zap.Uint("roleID", roleID), zap.Error(err))
		return
	}
	s.invalidateCacheForUserIDs(userIDs)
}

func (s *roleService) invalidateCacheForUserIDs(userIDs []uint) {
	for _, uid := range userIDs {
		if err := s.rbacCache.InvalidateUserRoles(uid); err != nil {
			zap.L().Warn("Failed to invalidate RBAC cache", zap.Uint("userID", uid), zap.Error(err))
		}
	}
}

