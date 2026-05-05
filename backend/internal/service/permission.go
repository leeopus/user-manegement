package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/user-system/backend/internal/dto"
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/auth"
	apperrors "github.com/user-system/backend/pkg/errors"
	"go.uber.org/zap"
)

var permissionCodePattern = regexp.MustCompile(`^[a-z0-9_]{2,100}$`)

type PermissionService interface {
	CreatePermission(name, code, resource, action, description string, auditCtx dto.AuditContext) (*repository.Permission, error)
	GetPermission(id uint) (*repository.Permission, error)
	UpdatePermission(id uint, name, code, resource, action, description string, auditCtx dto.AuditContext) (*repository.Permission, error)
	DeletePermission(id uint, auditCtx dto.AuditContext) error
	ListPermissions(offset, pageSize int) ([]repository.Permission, int64, error)
}

type permissionService struct {
	permissionRepo repository.PermissionRepository
	auditLogger    *AuditLogger
	rbacCache      *auth.RBACCacheManager
}

func NewPermissionService(permissionRepo repository.PermissionRepository, auditLogger *AuditLogger, rbacCache *auth.RBACCacheManager) PermissionService {
	return &permissionService{
		permissionRepo: permissionRepo,
		auditLogger:    auditLogger,
		rbacCache:      rbacCache,
	}
}

func validatePermissionInput(name, code, resource, action string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return fmt.Errorf("permission name must be 1-50 characters")
	}
	if !permissionCodePattern.MatchString(strings.ToLower(strings.TrimSpace(code))) {
		return fmt.Errorf("permission code must be lowercase alphanumeric with underscores, 2-100 characters")
	}
	if strings.TrimSpace(resource) == "" || len(resource) > 50 {
		return fmt.Errorf("resource must be 1-50 characters")
	}
	if strings.TrimSpace(action) == "" || len(action) > 20 {
		return fmt.Errorf("action must be 1-20 characters")
	}
	return nil
}

func (s *permissionService) CreatePermission(name, code, resource, action, description string, auditCtx dto.AuditContext) (*repository.Permission, error) {
	if err := validatePermissionInput(name, code, resource, action); err != nil {
		return nil, apperrors.ErrValidationFailed.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	code = strings.ToLower(strings.TrimSpace(code))

	if _, err := s.permissionRepo.FindByCode(code); err == nil {
		return nil, apperrors.ErrValidationFailed.WithDetails(map[string]interface{}{
			"reason": fmt.Sprintf("permission code %q already exists", code),
		})
	}

	permission := &repository.Permission{
		Name:        strings.TrimSpace(name),
		Code:        code,
		Resource:    strings.TrimSpace(resource),
		Action:      strings.TrimSpace(action),
		Description: strings.TrimSpace(description),
	}
	if err := s.permissionRepo.Create(permission); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	s.auditLogger.Log(&auditCtx, "create_permission", "permission", map[string]interface{}{
		"permission_id":   permission.ID,
		"permission_name": name,
		"permission_code": code,
	})

	return permission, nil
}

func (s *permissionService) GetPermission(id uint) (*repository.Permission, error) {
	return s.permissionRepo.FindByID(id)
}

func (s *permissionService) UpdatePermission(id uint, name, code, resource, action, description string, auditCtx dto.AuditContext) (*repository.Permission, error) {
	if err := validatePermissionInput(name, code, resource, action); err != nil {
		return nil, apperrors.ErrValidationFailed.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	permission, err := s.permissionRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.ErrPermissionNotFound
	}

	code = strings.ToLower(strings.TrimSpace(code))

	permission.Name = strings.TrimSpace(name)
	permission.Code = code
	permission.Resource = strings.TrimSpace(resource)
	permission.Action = strings.TrimSpace(action)
	permission.Description = strings.TrimSpace(description)

	if err := s.permissionRepo.Update(permission); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	// 权限变更后清除持有该权限的所有用户的 RBAC 缓存
	s.invalidateCacheForPermission(id)

	s.auditLogger.Log(&auditCtx, "update_permission", "permission", map[string]interface{}{
		"permission_id":   id,
		"permission_name": name,
		"permission_code": code,
	})

	return permission, nil
}

func (s *permissionService) DeletePermission(id uint, auditCtx dto.AuditContext) error {
	// 先获取受影响的用户列表，再删除
	userIDs, err := s.permissionRepo.GetUserIDsByPermissionID(id)
	if err != nil {
		zap.L().Warn("Failed to get user IDs for RBAC cache invalidation",
			zap.Uint("permissionID", id), zap.Error(err))
	}

	if err := s.permissionRepo.Delete(id); err != nil {
		return apperrors.ErrInternalServer
	}

	// 清除缓存
	for _, uid := range userIDs {
		if err := s.rbacCache.InvalidateUserRoles(uid); err != nil {
			zap.L().Warn("Failed to invalidate RBAC cache",
				zap.Uint("userID", uid), zap.Error(err))
		}
	}

	s.auditLogger.Log(&auditCtx, "delete_permission", "permission", map[string]interface{}{
		"permission_id": id,
	})

	return nil
}

func (s *permissionService) ListPermissions(offset, pageSize int) ([]repository.Permission, int64, error) {
	return s.permissionRepo.List(offset, pageSize)
}

func (s *permissionService) invalidateCacheForPermission(permissionID uint) {
	userIDs, err := s.permissionRepo.GetUserIDsByPermissionID(permissionID)
	if err != nil {
		zap.L().Warn("Failed to get user IDs for RBAC cache invalidation",
			zap.Uint("permissionID", permissionID), zap.Error(err))
		return
	}
	for _, uid := range userIDs {
		if err := s.rbacCache.InvalidateUserRoles(uid); err != nil {
			zap.L().Warn("Failed to invalidate RBAC cache",
				zap.Uint("userID", uid), zap.Error(err))
		}
	}
}

