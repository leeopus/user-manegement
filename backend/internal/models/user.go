package models

import (
	"time"

	"gorm.io/gorm"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusLocked   UserStatus = "locked"
)

type User struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Username     string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email        string     `gorm:"size:100;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Avatar       string     `gorm:"size:255" json:"avatar,omitempty"`
	Status       UserStatus `gorm:"size:20;default:active" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	Roles        []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

func (User) TableName() string {
	return "users"
}
