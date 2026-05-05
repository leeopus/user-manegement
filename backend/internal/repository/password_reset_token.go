package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// PasswordResetToken 密码重置令牌
type PasswordResetToken struct {
	gorm.Model
	Email     string    `gorm:"size:255;index:idx_prt_email"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"` // 存储 SHA-256 哈希，非明文
	ExpiresAt time.Time `gorm:"not null;index:idx_prt_expires"`
	Used      bool      `gorm:"default:false"`
	UserID    uint
}

// HashToken 对明文 token 进行 SHA-256 哈希
func HashResetToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
