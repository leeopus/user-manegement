package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:50;not null" json:"name"`
	Code        string         `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description string         `gorm:"size:255" json:"description,omitempty"`
	Users       []User         `gorm:"many2many:user_roles;" json:"-"`
	Permissions []Permission   `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}
