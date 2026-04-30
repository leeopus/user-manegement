package models

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	UserID     uint           `json:"user_id"`
	Action     string         `gorm:"size:50;not null" json:"action"`
	Resource   string         `gorm:"size:100;not null" json:"resource"`
	ResourceID uint           `json:"resource_id,omitempty"`
	Details    string         `gorm:"type:text" json:"details,omitempty"`
	IPAddress  string         `gorm:"size:50" json:"ip_address,omitempty"`
	UserAgent  string         `gorm:"size:500" json:"user_agent,omitempty"`
	User       User           `gorm:"foreignKey:UserID" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type UserRole struct {
	UserID uint `gorm:"primaryKey" json:"user_id"`
	RoleID uint `gorm:"primaryKey" json:"role_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`
	Role   Role `gorm:"foreignKey:RoleID" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

type RolePermission struct {
	RoleID       uint       `gorm:"primaryKey" json:"role_id"`
	PermissionID uint       `gorm:"primaryKey" json:"permission_id"`
	Role         Role       `gorm:"foreignKey:RoleID" json:"-"`
	Permission   Permission `gorm:"foreignKey:PermissionID" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
