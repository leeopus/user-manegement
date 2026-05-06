package repository

import (
	"time"

	"gorm.io/gorm"
)

const maxPasswordHistory = 5

// PasswordHistory 密码历史记录，防止密码重复使用
type PasswordHistory struct {
	gorm.Model
	UserID       uint `gorm:"not null;index:idx_ph_user_created"`
	PasswordHash string
	User         User `gorm:"foreignKey:UserID"`
	CreatedAt    time.Time
}

type PasswordHistoryRepository interface {
	Create(ph *PasswordHistory) error
	CreateWithTx(tx *gorm.DB, ph *PasswordHistory) error
	FindByUserID(userID uint, limit int) ([]PasswordHistory, error)
	CleanupOld(userID uint, keep int) error
}

type passwordHistoryRepository struct {
	db *gorm.DB
}

func NewPasswordHistoryRepository(db *gorm.DB) PasswordHistoryRepository {
	return &passwordHistoryRepository{db: db}
}

func (r *passwordHistoryRepository) Create(ph *PasswordHistory) error {
	return r.db.Create(ph).Error
}

func (r *passwordHistoryRepository) CreateWithTx(tx *gorm.DB, ph *PasswordHistory) error {
	return tx.Create(ph).Error
}

func (r *passwordHistoryRepository) FindByUserID(userID uint, limit int) ([]PasswordHistory, error) {
	var histories []PasswordHistory
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories).Error
	return histories, err
}

func (r *passwordHistoryRepository) CleanupOld(userID uint, keep int) error {
	return r.db.Where("user_id = ? AND id NOT IN (?)",
		userID,
		r.db.Model(&PasswordHistory{}).
			Select("id").
			Where("user_id = ?", userID).
			Order("created_at DESC").
			Limit(keep),
	).Delete(&PasswordHistory{}).Error
}
