package dto

import "time"

// UserResponse 用户公开信息（API 响应）
type UserResponse struct {
	ID                uint       `json:"id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	Avatar            string     `json:"avatar,omitempty"`
	Status            string     `json:"status"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at,omitempty"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// AdminUserResponse 管理员可见的用户信息（包含 LastLoginIP 等敏感字段）
type AdminUserResponse struct {
	UserResponse
	LastLoginIP string `json:"last_login_ip,omitempty"`
}

// UserWithRolesResponse 带角色信息的用户响应（管理员使用）
type UserWithRolesResponse struct {
	UserResponse
	Roles []RoleResponse `json:"roles,omitempty"`
}

// RoleResponse 角色公开信息
type RoleResponse struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Code        string              `json:"code"`
	Description string              `json:"description,omitempty"`
	Permissions []PermissionResponse `json:"permissions,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

// PermissionResponse 权限公开信息
type PermissionResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// OAuthApplicationResponse OAuth 应用公开信息
type OAuthApplicationResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"` // 仅在创建时返回
	RedirectURIs string    `json:"redirect_uris"`
	Scopes       string    `json:"scopes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
