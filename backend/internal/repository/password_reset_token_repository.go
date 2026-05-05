package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PasswordResetTokenRepository 密码重置令牌仓库
type PasswordResetTokenRepository interface {
	Create(token *PasswordResetToken) error
	FindByTokenHash(tokenHash string) (*PasswordResetToken, error)
	FindByTokenHashForUpdate(tx *gorm.DB, tokenHash string) (*PasswordResetToken, error)
	MarkAsUsedByHash(tx *gorm.DB, tokenHash string) error
	DeleteExpiredTokens() error
	FindByEmail(email string) ([]PasswordResetToken, error)
	Transaction(fn func(tx *gorm.DB) error) error
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

func (r *passwordResetTokenRepository) FindByTokenHash(tokenHash string) (*PasswordResetToken, error) {
	var resetToken PasswordResetToken
	err := r.db.Where("token_hash = ?", tokenHash).First(&resetToken).Error
	if err != nil {
		return nil, err
	}
	return &resetToken, nil
}

func (r *passwordResetTokenRepository) FindByTokenHashForUpdate(tx *gorm.DB, tokenHash string) (*PasswordResetToken, error) {
	var resetToken PasswordResetToken
	err := tx.Where("token_hash = ?", tokenHash).Clauses(clause.Locking{Strength: "UPDATE"}).First(&resetToken).Error
	if err != nil {
		return nil, err
	}
	return &resetToken, nil
}

func (r *passwordResetTokenRepository) MarkAsUsedByHash(tx *gorm.DB, tokenHash string) error {
	return tx.Model(&PasswordResetToken{}).Where("token_hash = ?", tokenHash).Update("used", true).Error
}

func (r *passwordResetTokenRepository) DeleteExpiredTokens() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&PasswordResetToken{}).Error
}

func (r *passwordResetTokenRepository) FindByEmail(email string) ([]PasswordResetToken, error) {
	var tokens []PasswordResetToken
	err := r.db.Where("email = ?", email).Find(&tokens).Error
	return tokens, err
}

func (r *passwordResetTokenRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
