package service

import (
	"github.com/user-system/backend/internal/repository"
	"github.com/user-system/backend/pkg/utils"
)

type UserService interface {
	CreateUser(username, email, password string) (*repository.User, error)
	GetUser(id uint) (*repository.User, error)
	UpdateUser(id uint, username, email string) (*repository.User, error)
	DeleteUser(id uint) error
	ListUsers(page, pageSize int) ([]repository.User, int64, error)
	AssignRole(userID, roleID uint) error
	RemoveRole(userID, roleID uint) error
}

type userService struct {
	userRepo     repository.UserRepository
	roleRepo     repository.RoleRepository
	auditLogRepo repository.AuditLogRepository
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auditLogRepo repository.AuditLogRepository) UserService {
	return &userService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		auditLogRepo: auditLogRepo,
	}
}

func (s *userService) CreateUser(username, email, password string) (*repository.User, error) {
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

	return user, nil
}

func (s *userService) GetUser(id uint) (*repository.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) UpdateUser(id uint, username, email string) (*repository.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	user.Username = username
	user.Email = email

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) DeleteUser(id uint) error {
	return s.userRepo.Delete(id)
}

func (s *userService) ListUsers(page, pageSize int) ([]repository.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.userRepo.List(offset, pageSize)
}

func (s *userService) AssignRole(userID, roleID uint) error {
	// Implementation would add user_role association
	return nil
}

func (s *userService) RemoveRole(userID, roleID uint) error {
	// Implementation would remove user_role association
	return nil
}
