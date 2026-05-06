package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

type OAuthApplication struct {
	gorm.Model
	Name         string       `gorm:"size:100;not null"`
	ClientID     string       `gorm:"size:100;uniqueIndex;not null"`
	ClientSecret string       `gorm:"size:255;not null"`
	RedirectURIs string       `gorm:"type:text;not null"`
	Scopes       string       `gorm:"type:text;default:read,write"`
	Tokens       []OAuthToken `gorm:"foreignKey:ApplicationID"`
}

type OAuthToken struct {
	gorm.Model
	ApplicationID uint   `gorm:"not null;index:idx_oauth_token_app"`
	UserID        uint   `gorm:"not null;index:idx_oauth_token_user"`
	AccessToken   string `gorm:"size:64;uniqueIndex;not null"`
	RefreshToken  string `gorm:"size:64;uniqueIndex"`
	ExpiresAt     time.Time
	Application   OAuthApplication `gorm:"foreignKey:ApplicationID"`
	User          User             `gorm:"foreignKey:UserID"`
}

// HashOAuthToken 计算 OAuth token 的 SHA-256 哈希，用于数据库存储和查询
func HashOAuthToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type OAuthApplicationRepository interface {
	Create(app *OAuthApplication) error
	FindByID(id uint) (*OAuthApplication, error)
	FindByClientID(clientID string) (*OAuthApplication, error)
	Update(app *OAuthApplication) error
	Delete(id uint) error
	List(offset, limit int) ([]OAuthApplication, int64, error)
}

type OAuthTokenRepository interface {
	Create(token *OAuthToken) error
	FindByAccessToken(hash string) (*OAuthToken, error)
	FindByRefreshToken(hash string) (*OAuthToken, error)
	Update(token *OAuthToken) error
	Delete(id uint) error
	RevokeByTokenHash(hash string) error
}

type oauthApplicationRepository struct {
	db *gorm.DB
}

func NewOAuthApplicationRepository(db *gorm.DB) OAuthApplicationRepository {
	return &oauthApplicationRepository{db: db}
}

func (r *oauthApplicationRepository) Create(app *OAuthApplication) error {
	return r.db.Create(app).Error
}

func (r *oauthApplicationRepository) FindByID(id uint) (*OAuthApplication, error) {
	var app OAuthApplication
	err := r.db.First(&app, id).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *oauthApplicationRepository) FindByClientID(clientID string) (*OAuthApplication, error) {
	var app OAuthApplication
	err := r.db.Where("client_id = ?", clientID).First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *oauthApplicationRepository) Update(app *OAuthApplication) error {
	return r.db.Model(app).Select("name", "redirect_uris").Updates(app).Error
}

func (r *oauthApplicationRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("application_id = ?", id).Delete(&OAuthToken{}).Error; err != nil {
			return err
		}
		return tx.Delete(&OAuthApplication{}, id).Error
	})
}

func (r *oauthApplicationRepository) List(offset, limit int) ([]OAuthApplication, int64, error) {
	var apps []OAuthApplication
	var total int64

	if err := r.db.Model(&OAuthApplication{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Offset(offset).Limit(limit).Find(&apps).Error
	return apps, total, err
}

type oauthTokenRepository struct {
	db *gorm.DB
}

func NewOAuthTokenRepository(db *gorm.DB) OAuthTokenRepository {
	return &oauthTokenRepository{db: db}
}

func (r *oauthTokenRepository) Create(token *OAuthToken) error {
	return r.db.Create(token).Error
}

func (r *oauthTokenRepository) FindByAccessToken(hash string) (*OAuthToken, error) {
	var oauthToken OAuthToken
	err := r.db.Preload("User").Preload("Application").Where("access_token = ?", hash).First(&oauthToken).Error
	if err != nil {
		return nil, err
	}
	return &oauthToken, nil
}

func (r *oauthTokenRepository) FindByRefreshToken(hash string) (*OAuthToken, error) {
	var oauthToken OAuthToken
	err := r.db.Preload("User").Preload("Application").Where("refresh_token = ?", hash).First(&oauthToken).Error
	if err != nil {
		return nil, err
	}
	return &oauthToken, nil
}

func (r *oauthTokenRepository) Update(token *OAuthToken) error {
	return r.db.Model(token).Select("access_token", "refresh_token", "expires_at").Updates(token).Error
}

func (r *oauthTokenRepository) Delete(id uint) error {
	return r.db.Delete(&OAuthToken{}, id).Error
}

// RevokeByTokenHash 通过 token 哈希吊销（匹配 access_token 或 refresh_token）
func (r *oauthTokenRepository) RevokeByTokenHash(hash string) error {
	return r.db.Where("access_token = ? OR refresh_token = ?", hash, hash).Delete(&OAuthToken{}).Error
}
