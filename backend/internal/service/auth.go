package service

import (
	"errors"
	"time"

	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/utils"
)

type AuthService interface {
	Register(username, email, password string) (*repository.User, error)
	Login(email, password string) (*repository.User, string, string, error)
	RefreshToken(refreshToken string) (*repository.User, string, error)
	Logout(userID uint) error
	GetCurrentUser(userID uint) (*repository.User, error)
}

type authService struct {
	userRepo     repository.UserRepository
	auditLogRepo repository.AuditLogRepository
}

func NewAuthService(userRepo repository.UserRepository, auditLogRepo repository.AuditLogRepository) AuthService {
	return &authService{
		userRepo:     userRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (s *authService) Register(username, email, password string) (*repository.User, error) {
	// Check if user exists
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, errors.New("email already exists")
	}

	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, errors.New("username already exists")
	}

	// Hash password
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Create audit log
	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "register",
		Resource: "user",
		Details:  "User registered successfully",
	}
	s.auditLogRepo.Create(auditLog)

	return user, nil
}

func (s *authService) Login(email, password string) (*repository.User, string, string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, "", "", errors.New("invalid credentials")
	}

	if user.Status != "active" {
		return nil, "", "", errors.New("account is not active")
	}

	// Generate tokens
	accessToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, "", "", err
	}

	// Update last login
	s.userRepo.UpdateLastLogin(user.ID)

	// Create audit log
	auditLog := &repository.AuditLog{
		UserID:   user.ID,
		Action:   "login",
		Resource: "user",
		Details:  "User logged in successfully",
	}
	s.auditLogRepo.Create(auditLog)

	return user, accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshToken string) (*repository.User, string, error) {
	claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	newToken, err := utils.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		return nil, err
	}

	return user, newToken, nil
}

func (s *authService) Logout(userID uint) error {
	// Create audit log
	auditLog := &repository.AuditLog{
		UserID:   userID,
		Action:   "logout",
		Resource: "user",
		Details:  "User logged out",
	}
	return s.auditLogRepo.Create(auditLog)
}

func (s *authService) GetCurrentUser(userID uint) (*repository.User, error) {
	return s.userRepo.FindByID(userID)
}
