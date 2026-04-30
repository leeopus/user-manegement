package repository

import (
	"gorm.io/gorm"
)

type AuditLog struct {
	gorm.Model
	UserID     uint
	Action     string `gorm:"size:50;not null"`
	Resource   string `gorm:"size:100;not null"`
	ResourceID uint
	Details    string `gorm:"type:text"`
	IPAddress  string `gorm:"size:50"`
	UserAgent  string `gorm:"size:500"`
	User       User `gorm:"foreignKey:UserID"`
}

type AuditLogRepository interface {
	Create(log *AuditLog) error
	FindByUserID(userID uint, offset, limit int) ([]AuditLog, int64, error)
	List(offset, limit int) ([]AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) FindByUserID(userID uint, offset, limit int) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	if err := r.db.Model(&AuditLog{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *auditLogRepository) List(offset, limit int) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	if err := r.db.Model(&AuditLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

type UserRole struct {
	UserID    uint `gorm:"primaryKey"`
	RoleID    uint `gorm:"primaryKey"`
	User      User `gorm:"foreignKey:UserID"`
	Role      Role `gorm:"foreignKey:RoleID"`
	CreatedAt gorm.DeletedAt
}

type RolePermission struct {
	RoleID       uint       `gorm:"primaryKey"`
	PermissionID uint       `gorm:"primaryKey"`
	Role         Role       `gorm:"foreignKey:RoleID"`
	Permission   Permission `gorm:"foreignKey:PermissionID"`
	CreatedAt    gorm.DeletedAt
}
