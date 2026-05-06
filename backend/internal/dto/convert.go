package dto

import "github.com/user-system/backend/internal/repository"

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

// ToUserWithRolesResponseList 批量转换（带角色）
func ToUserWithRolesResponseList(users []repository.User) []UserWithRolesResponse {
	result := make([]UserWithRolesResponse, len(users))
	for i := range users {
		result[i] = ToUserWithRolesResponse(&users[i])
	}
	return result
}
