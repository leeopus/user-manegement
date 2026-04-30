package service

import (
	"github.com/user-system/backend/internal/repository"
)

type RoleService interface {
	CreateRole(name, code, description string) (*repository.Role, error)
	GetRole(id uint) (*repository.Role, error)
	UpdateRole(id uint, name, code, description string) (*repository.Role, error)
	DeleteRole(id uint) error
	ListRoles(page, pageSize int) ([]repository.Role, int64, error)
}

type roleService struct {
	roleRepo     repository.RoleRepository
	permissionRepo repository.PermissionRepository
	auditLogRepo repository.AuditLogRepository
}

func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository, auditLogRepo repository.AuditLogRepository) RoleService {
	return &roleService{
		roleRepo:     roleRepo,
		permissionRepo: permissionRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (s *roleService) CreateRole(name, code, description string) (*repository.Role, error) {
	role := &repository.Role{
		Name:        name,
		Code:        code,
		Description: description,
	}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) GetRole(id uint) (*repository.Role, error) {
	return s.roleRepo.FindByID(id)
}

func (s *roleService) UpdateRole(id uint, name, code, description string) (*repository.Role, error) {
	role, err := s.roleRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	role.Name = name
	role.Code = code
	role.Description = description
	if err := s.roleRepo.Update(role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *roleService) DeleteRole(id uint) error {
	return s.roleRepo.Delete(id)
}

func (s *roleService) ListRoles(page, pageSize int) ([]repository.Role, int64, error) {
	offset := (page - 1) * pageSize
	return s.roleRepo.List(offset, pageSize)
}
