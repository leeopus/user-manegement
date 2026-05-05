package service

import (
	"fmt"

	"github.com/user-system/backend/internal/repository"
	apperrors "github.com/user-system/backend/pkg/errors"
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
	// 验证用户名
	if err := utils.ValidateUsername(username); err != nil {
		return nil, apperrors.ErrUsernameInvalidPattern.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 验证邮箱
	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 验证密码强度
	if _, err := utils.ValidatePassword(password, username); err != nil {
		return nil, apperrors.ErrPasswordTooWeak.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 检查用户名是否已存在
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, apperrors.ErrUsernameAlreadyExists
	}

	// 检查邮箱是否已存在
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, apperrors.ErrEmailAlreadyExists
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, apperrors.ErrInternalServer
	}

	user := &repository.User{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	// 审计日志
	auditLog := &repository.AuditLog{
		Action:   "create_user",
		Resource: "user",
		Details:  fmt.Sprintf("Admin created user: %s (%s)", username, email),
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		_ = err
	}

	return user, nil
}

func (s *userService) GetUser(id uint) (*repository.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *userService) UpdateUser(id uint, username, email string) (*repository.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	// 验证新用户名
	if err := utils.ValidateUsername(username); err != nil {
		return nil, apperrors.ErrUsernameInvalidPattern.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	// 验证新邮箱
	if err := utils.ValidateEmail(email); err != nil {
		return nil, apperrors.ErrEmailInvalid.WithDetails(map[string]interface{}{
			"reason": err.Error(),
		})
	}

	user.Username = username
	user.Email = email

	if err := s.userRepo.Update(user); err != nil {
		return nil, apperrors.ErrInternalServer
	}

	// 审计日志
	auditLog := &repository.AuditLog{
		UserID:   id,
		Action:   "update_user",
		Resource: "user",
		Details:  fmt.Sprintf("User updated to: %s (%s)", username, email),
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		_ = err
	}

	return user, nil
}

func (s *userService) DeleteUser(id uint) error {
	if err := s.userRepo.Delete(id); err != nil {
		return apperrors.ErrInternalServer
	}

	// 审计日志
	auditLog := &repository.AuditLog{
		Action:   "delete_user",
		Resource: "user",
		Details:  fmt.Sprintf("Admin deleted user ID: %d", id),
	}
	if err := s.auditLogRepo.Create(auditLog); err != nil {
		_ = err
	}

	return nil
}

func (s *userService) ListUsers(page, pageSize int) ([]repository.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.userRepo.List(offset, pageSize)
}

func (s *userService) AssignRole(userID, roleID uint) error {
	// TODO: implement with transaction
	return nil
}

func (s *userService) RemoveRole(userID, roleID uint) error {
	// TODO: implement with transaction
	return nil
}
