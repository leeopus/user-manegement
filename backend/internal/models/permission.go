package models

import (
	"time"

	"gorm.io/gorm"
)

type Permission struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:50;not null" json:"name"`
	Code        string         `gorm:"size:100;uniqueIndex;not null" json:"code"`
	Resource    string         `gorm:"size:50;not null" json:"resource"`
	Action      string         `gorm:"size:20;not null" json:"action"`
	Description string         `gorm:"size:255" json:"description,omitempty"`
	Roles       []Role         `gorm:"many2many:role_permissions;" json:"-"`
}

func (Permission) TableName() string {
	return "permissions"
}
