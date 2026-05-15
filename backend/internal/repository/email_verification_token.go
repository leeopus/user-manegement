package repository

import (
	"time"

	"gorm.io/gorm"
)

type EmailVerificationToken struct {
	gorm.Model
	Email     string    `gorm:"size:100;not null;index"`
	TokenHash string    `gorm:"uniqueIndex;size:64;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	Used      bool      `gorm:"default:false"`
	UserID    uint      `gorm:"not null;index"`
}

type EmailVerificationTokenRepository interface {
	Create(token *EmailVerificationToken) error
	FindActiveByTokenHash(tokenHash string) (*EmailVerificationToken, error)
	MarkAsUsed(tx *gorm.DB, id uint) error
	DeleteExpired() (int64, error)
	Transaction(fn func(tx *gorm.DB) error) error
}

type emailVerificationTokenRepository struct {
	db *gorm.DB
}

func NewEmailVerificationTokenRepository(db *gorm.DB) EmailVerificationTokenRepository {
	return &emailVerificationTokenRepository{db: db}
}

func (r *emailVerificationTokenRepository) Create(token *EmailVerificationToken) error {
	return r.db.Create(token).Error
}

func (r *emailVerificationTokenRepository) FindActiveByTokenHash(tokenHash string) (*EmailVerificationToken, error) {
	var token EmailVerificationToken
	err := r.db.Where("token_hash = ? AND used = ? AND expires_at > ?", tokenHash, false, time.Now()).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *emailVerificationTokenRepository) MarkAsUsed(tx *gorm.DB, id uint) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.Model(&EmailVerificationToken{}).Where("id = ?", id).Update("used", true).Error
}

func (r *emailVerificationTokenRepository) DeleteExpired() (int64, error) {
	result := r.db.Where("expires_at < ? OR used = ?", time.Now(), true).Delete(&EmailVerificationToken{})
	return result.RowsAffected, result.Error
}

func (r *emailVerificationTokenRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
