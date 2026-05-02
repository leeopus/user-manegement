package repository

import (
	"time"

	"gorm.io/gorm"
)

// PasswordResetToken 密码重置令牌
type PasswordResetToken struct {
	gorm.Model
	Email     string
	Token     string    `gorm:"uniqueIndex;not null"` // 唯一索引
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	UserID    uint      // 关联用户ID
}
