package service

import (
	"github.com/user-system/backend/internal/repository"
)

type PermissionService interface {
	CreatePermission(name, code, resource, action, description string) (*repository.Permission, error)
	GetPermission(id uint) (*repository.Permission, error)
	UpdatePermission(id uint, name, code, resource, action, description string) (*repository.Permission, error)
	DeletePermission(id uint) error
	ListPermissions(page, pageSize int) ([]repository.Permission, int64, error)
}

type permissionService struct {
	permissionRepo repository.PermissionRepository
	auditLogRepo   repository.AuditLogRepository
}

func NewPermissionService(permissionRepo repository.PermissionRepository, auditLogRepo repository.AuditLogRepository) PermissionService {
	return &permissionService{
		permissionRepo: permissionRepo,
		auditLogRepo:   auditLogRepo,
	}
}

func (s *permissionService) CreatePermission(name, code, resource, action, description string) (*repository.Permission, error) {
	permission := &repository.Permission{
		Name:        name,
		Code:        code,
		Resource:    resource,
		Action:      action,
		Description: description,
	}
	if err := s.permissionRepo.Create(permission); err != nil {
		return nil, err
	}
	return permission, nil
}

func (s *permissionService) GetPermission(id uint) (*repository.Permission, error) {
	return s.permissionRepo.FindByID(id)
}

func (s *permissionService) UpdatePermission(id uint, name, code, resource, action, description string) (*repository.Permission, error) {
	permission, err := s.permissionRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	permission.Name = name
	permission.Code = code
	permission.Resource = resource
	permission.Action = action
	permission.Description = description
	if err := s.permissionRepo.Update(permission); err != nil {
		return nil, err
	}
	return permission, nil
}

func (s *permissionService) DeletePermission(id uint) error {
	return s.permissionRepo.Delete(id)
}

func (s *permissionService) ListPermissions(page, pageSize int) ([]repository.Permission, int64, error) {
	offset := (page - 1) * pageSize
	return s.permissionRepo.List(offset, pageSize)
}
