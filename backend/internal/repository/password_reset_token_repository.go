package repository

import (
	"time"

	"gorm.io/gorm"
)

// PasswordResetTokenRepository 密码重置令牌仓库
type PasswordResetTokenRepository interface {
	Create(token *PasswordResetToken) error
	FindByToken(token string) (*PasswordResetToken, error)
	MarkAsUsed(token string) error
	DeleteExpiredTokens() error
	FindByEmail(email string) ([]PasswordResetToken, error)
}

type passwordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{db: db}
}

func (r *passwordResetTokenRepository) Create(token *PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *passwordResetTokenRepository) FindByToken(token string) (*PasswordResetToken, error) {
	var resetToken PasswordResetToken
	err := r.db.Where("token = ?", token).First(&resetToken).Error
	if err != nil {
		return nil, err
	}
	return &resetToken, nil
}

func (r *passwordResetTokenRepository) MarkAsUsed(token string) error {
	return r.db.Model(&PasswordResetToken{}).Where("token = ?", token).Update("used", true).Error
}

func (r *passwordResetTokenRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&PasswordResetToken{}).Error
}

func (r *passwordResetTokenRepository) FindByEmail(email string) ([]PasswordResetToken, error) {
	var tokens []PasswordResetToken
	err := r.db.Where("email = ?", email).Find(&tokens).Error
	return tokens, err
}
