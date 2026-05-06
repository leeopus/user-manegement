package repository

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/user-system/backend/internal/config"
	"gorm.io/gorm"
)

// PasswordResetToken 密码重置令牌
type PasswordResetToken struct {
	gorm.Model
	Email     string    `gorm:"size:255;index:idx_prt_email"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"` // 存储 HMAC-SHA256 哈希，非明文
	ExpiresAt time.Time `gorm:"not null;index:idx_prt_expires"`
	Used      bool      `gorm:"default:false"`
	UserID    uint
}

// HashResetToken 对明文 token 进行 HMAC-SHA256 哈希（使用服务端密钥签名）
func HashResetToken(token string) string {
	secret := "default-reset-secret"
	if cfg := config.Get(); cfg != nil && cfg.JWT.Secret != "" {
		secret = cfg.JWT.Secret
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
