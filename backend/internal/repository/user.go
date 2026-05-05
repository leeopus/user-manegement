package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username     string     `gorm:"size:50;uniqueIndex" json:"username"`
	Email        string     `gorm:"size:100;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Avatar       string     `gorm:"size:255" json:"avatar"`
	Status       string     `gorm:"size:20;default:active" json:"status"`
	LastLoginAt  *time.Time `gorm:"" json:"last_login_at"`
	Roles        []Role     `gorm:"many2many:user_roles;" json:"roles"`
}

// BeforeDelete 在软删除前清除唯一约束字段，避免阻止同名用户重新注册
// 对 email(username) 截断以避免超出列宽度限制
func (u *User) BeforeDelete(tx *gorm.DB) error {
	emailPrefix := fmt.Sprintf("deleted_%d_", u.ID)
	emailVal := emailPrefix + u.Email
	if len(emailVal) > 95 { // 100 - 5 bytes safety margin
		emailVal = emailPrefix + u.Email[:95-len(emailPrefix)]
	}
	u.Email = emailVal

	usernamePrefix := fmt.Sprintf("deleted_%d_", u.ID)
	usernameVal := usernamePrefix + u.Username
	if len(usernameVal) > 45 { // 50 - 5 bytes safety margin
		usernameVal = usernamePrefix + u.Username[:45-len(usernamePrefix)]
	}
	u.Username = usernameVal
	return nil
}

type UserRepository interface {
	Create(user *User) error
	FindByID(id uint) (*User, error)
	FindByIDWithRoles(id uint) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByEmailWithRoles(email string) (*User, error)
	FindByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id uint) error
	List(offset, limit int) ([]User, int64, error)
	UpdateLastLogin(id uint) error
	GetUserRoles(userID uint) ([]Role, error)
	Transaction(fn func(tx *gorm.DB) error) error
	CreateWithTx(tx *gorm.DB, user *User) error
	UpdateWithTx(tx *gorm.DB, user *User) error
	FindByEmailWithTx(tx *gorm.DB, email string) (*User, error)
	FindByUsernameWithTx(tx *gorm.DB, username string) (*User, error)
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

// FindByID 仅加载用户基本信息（不含角色）
func (r *userRepository) FindByID(id uint) (*User, error) {
	var user User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByIDWithRoles 加载用户信息及其角色和权限
func (r *userRepository) FindByIDWithRoles(id uint) (*User, error) {
	var user User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 仅加载用户基本信息（登录验证密码用）
func (r *userRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmailWithRoles 加载用户信息及其角色
func (r *userRepository) FindByEmailWithRoles(email string) (*User, error) {
	var user User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").Where("email = ?", email).First(&user).Error
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
	return r.db.Model(user).Select("username", "email", "password_hash", "status", "avatar").Updates(user).Error
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

	err := r.db.Preload("Roles").Offset(offset).Limit(limit).Order("id ASC").Find(&users).Error
	return users, total, err
}

func (r *userRepository) UpdateLastLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&User{}).Where("id = ?", id).Update("last_login_at", now).Error
}

func (r *userRepository) GetUserRoles(userID uint) ([]Role, error) {
	var user User
	err := r.db.Preload("Roles").Preload("Roles.Permissions").First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return user.Roles, nil
}

func (r *userRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *userRepository) CreateWithTx(tx *gorm.DB, user *User) error {
	return tx.Create(user).Error
}

func (r *userRepository) UpdateWithTx(tx *gorm.DB, user *User) error {
	return tx.Model(user).Select("username", "email", "password_hash", "status", "avatar").Updates(user).Error
}

func (r *userRepository) FindByEmailWithTx(tx *gorm.DB, email string) (*User, error) {
	var user User
	err := tx.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByUsernameWithTx(tx *gorm.DB, username string) (*User, error) {
	var user User
	err := tx.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
