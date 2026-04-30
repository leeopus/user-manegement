package repository

import (
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
	ApplicationID uint   `gorm:"not null"`
	UserID        uint   `gorm:"not null"`
	AccessToken   string `gorm:"size:500;uniqueIndex;not null"`
	RefreshToken  string `gorm:"size:500;uniqueIndex"`
	ExpiresAt     gorm.DeletedAt
	Application   OAuthApplication `gorm:"foreignKey:ApplicationID"`
	User          User             `gorm:"foreignKey:UserID"`
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
	FindByAccessToken(token string) (*OAuthToken, error)
	FindByRefreshToken(token string) (*OAuthToken, error)
	Update(token *OAuthToken) error
	Delete(id uint) error
	RevokeToken(tokenString string) error
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
	return r.db.Save(app).Error
}

func (r *oauthApplicationRepository) Delete(id uint) error {
	return r.db.Delete(&OAuthApplication{}, id).Error
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

func (r *oauthTokenRepository) FindByAccessToken(token string) (*OAuthToken, error) {
	var oauthToken OAuthToken
	err := r.db.Preload("User").Preload("Application").Where("access_token = ?", token).First(&oauthToken).Error
	if err != nil {
		return nil, err
	}
	return &oauthToken, nil
}

func (r *oauthTokenRepository) FindByRefreshToken(token string) (*OAuthToken, error) {
	var oauthToken OAuthToken
	err := r.db.Preload("User").Preload("Application").Where("refresh_token = ?", token).First(&oauthToken).Error
	if err != nil {
		return nil, err
	}
	return &oauthToken, nil
}

func (r *oauthTokenRepository) Update(token *OAuthToken) error {
	return r.db.Save(token).Error
}

func (r *oauthTokenRepository) Delete(id uint) error {
	return r.db.Delete(&OAuthToken{}, id).Error
}

func (r *oauthTokenRepository) RevokeToken(tokenString string) error {
	return r.db.Where("access_token = ? OR refresh_token = ?", tokenString, tokenString).Delete(&OAuthToken{}).Error
}
