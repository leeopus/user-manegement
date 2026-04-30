package service

import (
	"errors"
	"time"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/utils"
)

type OAuthService interface {
	CreateApplication(name, clientSecret, redirectURIs string) (*repository.OAuthApplication, error)
	Authorize(clientID, redirectURI string) (*repository.OAuthApplication, error)
	Token(clientID, clientSecret, code string) (string, string, error)
	Userinfo(accessToken string) (*repository.User, error)
}

type oauthService struct {
	appRepo   repository.OAuthApplicationRepository
	tokenRepo repository.OAuthTokenRepository
	userRepo  repository.UserRepository
	auditLogRepo repository.AuditLogRepository
}

func NewOAuthService(appRepo repository.OAuthApplicationRepository, tokenRepo repository.OAuthTokenRepository, userRepo repository.UserRepository, auditLogRepo repository.AuditLogRepository) OAuthService {
	return &oauthService{
		appRepo:   appRepo,
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (s *oauthService) CreateApplication(name, clientSecret, redirectURIs string) (*repository.OAuthApplication, error) {
	app := &repository.OAuthApplication{
		Name:         name,
		ClientID:     utils.GenerateToken(1, "app", "client")[0:32],
		ClientSecret: clientSecret,
		RedirectURIs: redirectURIs,
	}
	if err := s.appRepo.Create(app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *oauthService) Authorize(clientID, redirectURI string) (*repository.OAuthApplication, error) {
	app, err := s.appRepo.FindByClientID(clientID)
	if err != nil {
		return nil, errors.New("invalid client_id")
	}
	return app, nil
}

func (s *oauthService) Token(clientID, clientSecret, code string) (string, string, error) {
	app, err := s.appRepo.FindByClientID(clientID)
	if err != nil {
		return "", "", errors.New("invalid client_id")
	}

	if app.ClientSecret != clientSecret {
		return "", "", errors.New("invalid client_secret")
	}

	// Generate access token
	accessToken := utils.GenerateToken(1, "access", "token")
	refreshToken := utils.GenerateRefreshToken(1, "refresh", "token")

	// For simplicity, using user ID 1
	oauthToken := &repository.OAuthToken{
		ApplicationID: app.ID,
		UserID:        1,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		ExpiresAt:     time.Now().Add(3600 * time.Second),
	}

	if err := s.tokenRepo.Create(oauthToken); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *oauthService) Userinfo(accessToken string) (*repository.User, error) {
	token, err := s.tokenRepo.FindByAccessToken(accessToken)
	if err != nil {
		return nil, errors.New("invalid access token")
	}

	return s.userRepo.FindByID(token.UserID)
}
