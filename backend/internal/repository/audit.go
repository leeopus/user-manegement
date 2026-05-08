package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	gorm.Model
	UserID     uint   `gorm:"not null;index:idx_audit_user_id"`
	Action     string `gorm:"size:50;not null;index:idx_audit_action"`
	Resource   string `gorm:"size:100;not null;index:idx_audit_resource"`
	ResourceID uint
	Details    string `gorm:"type:text"`
	IPAddress  string `gorm:"size:50"`
	UserAgent  string `gorm:"size:500"`
	RequestID  string `gorm:"size:64;index:idx_audit_request_id"`
	User       User   `gorm:"foreignKey:UserID"`
	Username   string `gorm:"-"` // 非持久化字段，由 JOIN 填充，避免 N+1 查询
}

type AuditLogFilters struct {
	UserID   uint
	Action   string
	Resource string
	Search   string
}

type AuditLogRepository interface {
	Create(log *AuditLog) error
	FindByUserID(userID uint, offset, limit int) ([]AuditLog, int64, error)
	List(offset, limit int) ([]AuditLog, int64, error)
	ListFiltered(offset, limit int, filters AuditLogFilters) ([]AuditLog, int64, error)
	CleanupOlderThan(retentionDays int) (int64, error)
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
	return r.ListFiltered(offset, limit, AuditLogFilters{})
}

func (r *auditLogRepository) ListFiltered(offset, limit int, filters AuditLogFilters) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := r.db.Model(&AuditLog{})
	if filters.UserID > 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if filters.Search != "" {
		search := filters.Search
		search = strings.ReplaceAll(search, "%", "\\%")
		search = strings.ReplaceAll(search, "_", "\\_")
		search = "%" + search + "%"
		query = query.Where("action ILIKE ? OR resource ILIKE ? OR details ILIKE ?", search, search, search)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Select("audit_logs.*, COALESCE(users.username, '') as username").
		Joins("LEFT JOIN users ON users.id = audit_logs.user_id").
		Order("audit_logs.created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

type UserRole struct {
	UserID    uint      `gorm:"primaryKey"`
	RoleID    uint      `gorm:"primaryKey"`
	User      User      `gorm:"foreignKey:UserID"`
	Role      Role      `gorm:"foreignKey:RoleID"`
	CreatedAt time.Time
}

type RolePermission struct {
	RoleID       uint       `gorm:"primaryKey"`
	PermissionID uint       `gorm:"primaryKey"`
	Role         Role       `gorm:"foreignKey:RoleID"`
	Permission   Permission `gorm:"foreignKey:PermissionID"`
	CreatedAt    time.Time
}

func (r *auditLogRepository) CleanupOlderThan(retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := r.db.Unscoped().Where("created_at < ?", cutoff).Delete(&AuditLog{})
	return result.RowsAffected, result.Error
}
