package models

import (
	"time"

	"gorm.io/gorm"
)

type OAuthApplication struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	ClientID     string         `gorm:"size:100;uniqueIndex;not null" json:"client_id"`
	ClientSecret string         `gorm:"size:255;not null" json:"-"`
	RedirectURIs string         `gorm:"type:text;not null" json:"redirect_uris"`
	Scopes       string         `gorm:"type:text;default:read,write" json:"scopes"`
	Tokens       []OAuthToken   `gorm:"foreignKey:ApplicationID" json:"-"`
}

func (OAuthApplication) TableName() string {
	return "oauth_applications"
}

type OAuthToken struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	ApplicationID uint           `gorm:"not null" json:"application_id"`
	UserID        uint           `gorm:"not null" json:"user_id"`
	AccessToken   string         `gorm:"size:500;uniqueIndex;not null" json:"-"`
	RefreshToken  string         `gorm:"size:500;uniqueIndex" json:"-"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Application   OAuthApplication `gorm:"foreignKey:ApplicationID" json:"-"`
	User          User           `gorm:"foreignKey:UserID" json:"-"`
}

func (OAuthToken) TableName() string {
	return "oauth_tokens"
}
