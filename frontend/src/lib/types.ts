// =====================================================
// API 响应类型定义
// =====================================================

/**
 * 统一 API 响应格式
 */
export interface APIResponse<T = any> {
  success: boolean
  data?: T
  error?: APIError
}

/**
 * API 错误详情
 */
export interface APIError {
  code: string // 错误码，如 "AUTH_LOGIN_INVALID_CREDENTIALS_401"
  message: string // 翻译键，如 "AUTH_LOGIN_INVALID_CREDENTIALS"
  details?: Record<string, unknown> // 额外错误详情
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

export interface User {
  ID: number
  Email: string
  Username: string
  Avatar?: string
  Status?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface LoginResponseData {
  user: User
  // Token 现在通过 httpOnly cookie 返回，不再在响应体中
}

export interface RegisterResponseData {
  user: User
}
