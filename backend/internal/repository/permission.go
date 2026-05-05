package repository

import (
	"gorm.io/gorm"
)

type Permission struct {
	gorm.Model
	Name        string  `gorm:"size:50;not null"`
	Code        string  `gorm:"size:100;uniqueIndex;not null"`
	Resource    string  `gorm:"size:50;not null"`
	Action      string  `gorm:"size:20;not null"`
	Description string  `gorm:"size:255"`
	Roles       []Role  `gorm:"many2many:role_permissions;"`
}

type PermissionRepository interface {
	Create(permission *Permission) error
	FindByID(id uint) (*Permission, error)
	FindByCode(code string) (*Permission, error)
	Update(permission *Permission) error
	Delete(id uint) error
	List(offset, limit int) ([]Permission, int64, error)
	GetUserIDsByPermissionID(permissionID uint) ([]uint, error)
}

type permissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(permission *Permission) error {
	return r.db.Create(permission).Error
}

func (r *permissionRepository) FindByID(id uint) (*Permission, error) {
	var permission Permission
	err := r.db.First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) FindByCode(code string) (*Permission, error) {
	var permission Permission
	err := r.db.Where("code = ?", code).First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepository) Update(permission *Permission) error {
	return r.db.Model(permission).Select("name", "code", "resource", "action", "description").Updates(permission).Error
}

func (r *permissionRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Permission{}, id).Error
	})
}

func (r *permissionRepository) List(offset, limit int) ([]Permission, int64, error) {
	var permissions []Permission
	var total int64

	if err := r.db.Model(&Permission{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Offset(offset).Limit(limit).Find(&permissions).Error
	return permissions, total, err
}

// GetUserIDsByPermissionID 查询持有指定权限的所有用户 ID（通过 role_permissions → user_roles）
func (r *permissionRepository) GetUserIDsByPermissionID(permissionID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_roles").
		Select("DISTINCT user_roles.user_id").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Where("role_permissions.permission_id = ?", permissionID).
		Pluck("user_roles.user_id", &userIDs).Error
	return userIDs, err
}
