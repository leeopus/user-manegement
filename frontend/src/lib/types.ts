// =====================================================
// API 响应类型定义
// =====================================================

export interface APIResponse<T = any> {
  success: boolean
  data?: T
  error?: APIError
}

export interface APIError {
  code: string
  message: string
  request_id?: string
  details?: Record<string, unknown>
}

// =====================================================
// 请求类型
// =====================================================

export interface LoginRequest {
  email: string
  password: string
  remember_me?: boolean
}

export interface RegisterRequest {
  email: string
  password: string
}

export interface RefreshTokenRequest {
  refresh_token: string
}

// =====================================================
// 响应数据类型
// =====================================================

export interface Permission {
  id: number
  name: string
  code: string
  resource: string
  action: string
  description?: string
  created_at?: string
}

export interface Role {
  id: number
  name: string
  code: string
  description?: string
  permissions?: Permission[]
  created_at?: string
}

export interface User {
  id: number
  email: string
  username: string
  avatar?: string
  status?: string
  roles?: Role[]
  created_at?: string
  updated_at?: string
}

export interface LoginResponseData {
  user: User
}

export interface RegisterResponseData {
  message: string
}

// =====================================================
// 请求类型（管理页面）
// =====================================================

export interface CreateUserRequest {
  username: string
  email: string
  password: string
}

export interface UpdateUserRequest {
  username: string
  email: string
}

export interface CreateRoleRequest {
  name: string
  code: string
  description?: string
}

export interface UpdateRoleRequest {
  name: string
  code: string
  description?: string
}

export interface CreatePermissionRequest {
  name: string
  code: string
  resource: string
  action: string
  description?: string
}

export interface UpdatePermissionRequest {
  name: string
  code: string
  resource: string
  action: string
  description?: string
}

export interface CreateOAuthAppRequest {
  name: string
  redirect_uris: string
  scopes?: string
}

export interface UpdateOAuthAppRequest {
  name: string
  redirect_uris: string
}

// =====================================================
// 数据模型（扩展）
// =====================================================

export interface OAuthApplication {
  id: number
  name: string
  client_id: string
  client_secret?: string
  redirect_uris: string
  scopes?: string
  created_at: string
}

export interface AuditLog {
  id: number
  user_id: number
  username: string
  action: string
  resource: string
  resource_id: number
  details: string
  ip_address: string
  user_agent: string
  request_id: string
  created_at: string
}
