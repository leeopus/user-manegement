package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	FindByIDWithTx(tx *gorm.DB, id uint) (*Role, error)
	FindByCode(code string) (*Role, error)
	Update(role *Role) error
	Delete(id uint) error
	List(offset, limit int) ([]Role, int64, error)
	AssignPermission(roleID, permissionID uint) error
	RemovePermission(roleID, permissionID uint) error
	GetUserIDsByRoleID(roleID uint) ([]uint, error)
	AssignRoleToUser(userID, roleID uint) error
	AssignRoleToUserWithTx(tx *gorm.DB, userID, roleID uint) error
	RemoveRoleFromUser(userID, roleID uint) error
	RemoveRoleFromUserWithTx(tx *gorm.DB, userID, roleID uint) error
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

func (r *roleRepository) FindByIDWithTx(tx *gorm.DB, id uint) (*Role, error) {
	var role Role
	err := tx.Preload("Permissions").First(&role, id).Error
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
	return r.db.Model(role).Select("name", "code", "description").Updates(role).Error
}

// Delete 删除角色并级联清理所有关联关系
func (r *roleRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 清理 user_roles 关联
		if err := tx.Where("role_id = ?", id).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		// 清理 role_permissions 关联
		if err := tx.Where("role_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		// 删除角色本身
		return tx.Delete(&Role{}, id).Error
	})
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
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
		CreatedAt:    time.Now(),
	}).Error
}

func (r *roleRepository) RemovePermission(roleID, permissionID uint) error {
	return r.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).Delete(&RolePermission{}).Error
}

func (r *roleRepository) GetUserIDsByRoleID(roleID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.Table("user_roles").Where("role_id = ?", roleID).Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *roleRepository) AssignRoleToUser(userID, roleID uint) error {
	return r.assignRoleToUser(r.db, userID, roleID)
}

func (r *roleRepository) AssignRoleToUserWithTx(tx *gorm.DB, userID, roleID uint) error {
	return r.assignRoleToUser(tx, userID, roleID)
}

func (r *roleRepository) assignRoleToUser(db *gorm.DB, userID, roleID uint) error {
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&UserRole{
		UserID:    userID,
		RoleID:    roleID,
		CreatedAt: time.Now(),
	}).Error
}

func (r *roleRepository) RemoveRoleFromUser(userID, roleID uint) error {
	return r.removeRoleFromUser(r.db, userID, roleID)
}

func (r *roleRepository) RemoveRoleFromUserWithTx(tx *gorm.DB, userID, roleID uint) error {
	return r.removeRoleFromUser(tx, userID, roleID)
}

func (r *roleRepository) removeRoleFromUser(db *gorm.DB, userID, roleID uint) error {
	return db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&UserRole{}).Error
}
