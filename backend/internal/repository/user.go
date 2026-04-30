package repository

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string  `gorm:"size:50;uniqueIndex;not null"`
	Email        string  `gorm:"size:100;uniqueIndex;not null"`
	PasswordHash string  `gorm:"size:255;not null"`
	Avatar       string  `gorm:"size:255"`
	Status       string  `gorm:"size:20;default:active"`
	LastLoginAt  *gorm.DeletedAt
	Roles        []Role  `gorm:"many2many:user_roles;"`
}

type UserRepository interface {
	Create(user *User) error
	FindByID(id uint) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id uint) error
	List(offset, limit int) ([]User, int64, error)
	UpdateLastLogin(id uint) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id uint) (*User, error) {
	var user User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByUsername(username string) (*User, error) {
	var user User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(user *User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&User{}, id).Error
}

func (r *userRepository) List(offset, limit int) ([]User, int64, error) {
	var users []User
	var total int64

	if err := r.db.Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Offset(offset).Limit(limit).Preload("Roles").Find(&users).Error
	return users, total, err
}

func (r *userRepository) UpdateLastLogin(id uint) error {
	return r.db.Model(&User{}).Where("id = ?", id).Update("last_login_at", gorm.Expr("NOW()")).Error
}
