package dto

import (
	"encoding/json"
	"strings"

	"github.com/user-system/backend/internal/repository"
)

// ToUserResponse 将 User 模型转换为公开响应（不包含 LastLoginIP 等敏感信息）
func ToUserResponse(user *repository.User) UserResponse {
	return UserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Avatar:            user.Avatar,
		Status:            user.Status,
		EmailVerifiedAt:   user.EmailVerifiedAt,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
}

// ToAdminUserResponse 将 User 模型转换为管理员可见的响应（包含 LastLoginIP）
func ToAdminUserResponse(user *repository.User) AdminUserResponse {
	return AdminUserResponse{
		UserResponse: ToUserResponse(user),
		LastLoginIP:  user.LastLoginIP,
	}
}

// ToUserWithRolesResponse 将 User 模型转换为带角色的响应
func ToUserWithRolesResponse(user *repository.User) UserWithRolesResponse {
	resp := UserWithRolesResponse{
		UserResponse: ToUserResponse(user),
	}
	for _, role := range user.Roles {
		resp.Roles = append(resp.Roles, ToRoleResponse(&role))
	}
	return resp
}

// ToRoleResponse 将 Role 模型转换为公开响应
func ToRoleResponse(role *repository.Role) RoleResponse {
	resp := RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		CreatedAt:   role.CreatedAt,
	}
	for _, perm := range role.Permissions {
		resp.Permissions = append(resp.Permissions, ToPermissionResponse(&perm))
	}
	return resp
}

// ToPermissionResponse 将 Permission 模型转换为公开响应
func ToPermissionResponse(perm *repository.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          perm.ID,
		Name:        perm.Name,
		Code:        perm.Code,
		Resource:    perm.Resource,
		Action:      perm.Action,
		Description: perm.Description,
		CreatedAt:   perm.CreatedAt,
	}
}

// ToOAuthApplicationResponse 将 OAuthApplication 模型转换为公开响应
func ToOAuthApplicationResponse(app *repository.OAuthApplication) OAuthApplicationResponse {
	return OAuthApplicationResponse{
		ID:           app.ID,
		Name:         app.Name,
		ClientID:     app.ClientID,
		RedirectURIs: app.RedirectURIs,
		Scopes:       app.Scopes,
		CreatedAt:    app.CreatedAt,
	}
}

// ToUserResponseList 批量转换
func ToUserResponseList(users []repository.User) []UserResponse {
	result := make([]UserResponse, len(users))
	for i := range users {
		result[i] = ToUserResponse(&users[i])
	}
	return result
}

// ToMaskedUserResponseList 批量转换并遮蔽邮箱（用于用户列表等批量查询场景）
func ToMaskedUserResponseList(users []repository.User) []UserResponse {
	result := make([]UserResponse, len(users))
	for i := range users {
		resp := ToUserResponse(&users[i])
		resp.Email = MaskEmail(resp.Email)
		result[i] = resp
	}
	return result
}

// ToUserWithRolesResponseList 批量转换（带角色）
func ToUserWithRolesResponseList(users []repository.User) []UserWithRolesResponse {
	result := make([]UserWithRolesResponse, len(users))
	for i := range users {
		result[i] = ToUserWithRolesResponse(&users[i])
	}
	return result
}

// ToAuditLogResponse 将 AuditLog 模型转换为公开响应（脱敏 details 中的邮箱等敏感字段）
func ToAuditLogResponse(log *repository.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:         log.ID,
		UserID:     log.UserID,
		Username:   log.Username,
		Action:     log.Action,
		Resource:   log.Resource,
		ResourceID: log.ResourceID,
		Details:    maskSensitiveDetails(log.Details),
		IPAddress:  log.IPAddress,
		UserAgent:  log.UserAgent,
		RequestID:  log.RequestID,
		CreatedAt:  log.CreatedAt,
	}
}

// maskSensitiveDetails 对 details JSON 中的邮箱、IP 等敏感字段做脱敏
func maskSensitiveDetails(details string) string {
	if details == "" {
		return details
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(details), &m); err != nil {
		return details
	}
	for key, val := range m {
		s, ok := val.(string)
		if !ok || s == "" {
			continue
		}
		lowerKey := strings.ToLower(key)
		if containsAnyPattern(lowerKey, "email") && strings.Contains(s, "@") {
			m[key] = MaskEmail(s)
		} else if containsAnyPattern(lowerKey, "ip") && isIPv4(s) {
			m[key] = maskIP(s)
		}
	}
	masked, err := json.Marshal(m)
	if err != nil {
		return details
	}
	return string(masked)
}

func containsAnyPattern(key string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(key, p) {
			return true
		}
	}
	return false
}

func isIPv4(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && c != '.' {
			return false
		}
	}
	return strings.Contains(s, ".")
}

// maskIP 将 IPv4 地址脱敏，如 "192.168.1.100" → "192.168.*.*"
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		parts[2] = "*"
		parts[3] = "*"
		return strings.Join(parts, ".")
	}
	return ip
}

// MaskEmail 将邮箱脱敏为 a***@domain.com 格式
func MaskEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return email
	}
	return string(email[0]) + "***" + email[at:]
}
