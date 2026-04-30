package repository

import (
	"gorm.io/gorm"
)

type Role struct {
	gorm.Model
	Name        string       `gorm:"size:50;not null"`
	Code        string       `gorm:"size:50;uniqueIndex;not null"`
	Description string       `gorm:"size:255"`
	Users       []User       `gorm:"many2many:user_roles;"`
	Permissions []Permission `gorm:"many2many:role_permissions;"`
}

type RoleRepository interface {
	Create(role *Role) error
	FindByID(id uint) (*Role, error)
	FindByCode(code string) (*Role, error)
	Update(role *Role) error
	Delete(id uint) error
	List(offset, limit int) ([]Role, int64, error)
	AssignPermission(roleID, permissionID uint) error
	RemovePermission(roleID, permissionID uint) error
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(role *Role) error {
	return r.db.Create(role).Error
}

func (r *roleRepository) FindByID(id uint) (*Role, error) {
	var role Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindByCode(code string) (*Role, error) {
	var role Role
	err := r.db.Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) Update(role *Role) error {
	return r.db.Save(role).Error
}

func (r *roleRepository) Delete(id uint) error {
	return r.db.Delete(&Role{}, id).Error
}

func (r *roleRepository) List(offset, limit int) ([]Role, int64, error) {
	var roles []Role
	var total int64

	if err := r.db.Model(&Role{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Offset(offset).Limit(limit).Preload("Permissions").Find(&roles).Error
	return roles, total, err
}

func (r *roleRepository) AssignPermission(roleID, permissionID uint) error {
	return r.db.Exec("INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES (?, ?, NOW())", roleID, permissionID).Error
}

func (r *roleRepository) RemovePermission(roleID, permissionID uint) error {
	return r.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).Delete(&RolePermission{}).Error
}
