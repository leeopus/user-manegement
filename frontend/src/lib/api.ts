import { TFunction } from 'next-intl'
import {
  APIResponse,
  LoginRequest,
  LoginResponseData,
  RegisterRequest,
  RegisterResponseData,
  User,
} from './types'
import { APIException, extractDataOrThrow } from './errors'
import { getCSRFToken, addCSRFToHeaders } from './csrf'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

// 需要 CSRF token 的方法
const CSRF_REQUIRED_METHODS = ['POST', 'PUT', 'DELETE', 'PATCH']

class APIClient {
  private baseURL: string

  constructor(baseURL: string = API_BASE) {
    this.baseURL = baseURL
  }

  private getUrl(path: string): string {
    return this.baseURL ? `${this.baseURL}${path}` : path
  }

  private async request<T>(
    path: string,
    options?: RequestInit
  ): Promise<APIResponse<T>> {
    let url = this.getUrl(path)
    let headers = {
      'Content-Type': 'application/json',
      ...(options?.headers || {}),
    }

    // 对于需要 CSRF 的方法，添加 token
    if (options?.method && CSRF_REQUIRED_METHODS.includes(options.method)) {
      try {
        headers = await addCSRFToHeaders(headers)
      } catch (error) {
        console.error('Failed to add CSRF token:', error)
        // 继续请求，让后端返回错误
      }
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'include', // 包含 cookie
      })

      // 解析响应
      const data: APIResponse<T> = await response.json()

      // 如果是 CSRF 错误，清除缓存并重试一次
      if (!data.success && data.error?.code === 'CSRF_TOKEN_INVALID_403') {
        const { clearCSRFToken } = await import('./csrf')
        clearCSRFToken()

        // 重试一次
        headers = await addCSRFToHeaders({
          'Content-Type': 'application/json',
          ...(options?.headers || {}),
        })

        const retryResponse = await fetch(url, {
          ...options,
          headers,
          credentials: 'include',
        })

        return await retryResponse.json()
      }

      return data
    } catch (error) {
      // 网络错误处理
      throw new Error(`NETWORK_ERROR: ${error}`)
    }
  }

  /**
   * 登录
   */
  async login(credentials: LoginRequest): Promise<LoginResponseData> {
    const response = await this.request<LoginResponseData>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 注册
   */
  async register(data: RegisterRequest): Promise<RegisterResponseData> {
    const response = await this.request<RegisterResponseData>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 获取当前用户信息
   */
  async getUserInfo(token: string): Promise<User> {
    const response = await this.request<User>('/api/v1/auth/me', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 刷新令牌
   */
  async refreshToken(refreshToken: string): Promise<{ user: User; access_token: string }> {
    const response = await this.request<{ user: User; access_token: string }>(
      '/api/v1/auth/refresh',
      {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken }),
      }
    )

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 登出
   */
  async logout(token: string): Promise<void> {
    const response = await this.request<void>('/api/v1/auth/logout', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  /**
   * 请求密码重置
   */
  async requestPasswordReset(email: string): Promise<void> {
    const response = await this.request<void>('/api/v1/auth/password/reset-request', {
      method: 'POST',
      body: JSON.stringify({ email }),
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  /**
   * 重置密码 (不需要CSRF token，因为用户可能没有有效会话)
   */
  async resetPassword(token: string, newPassword: string): Promise<void> {
    // 直接使用fetch，绕过CSRF保护
    let url = this.getUrl('/api/v1/auth/password/reset')

    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ token, new_password: newPassword }),
    })

    const data: APIResponse<void> = await response.json()

    if (!data.success) {
      throw APIException.fromAPIError(data.error!)
    }
  }

  /**
   * 验证重置令牌 (不需要CSRF token)
   */
  async validateResetToken(token: string): Promise<{ valid: boolean }> {
    // 直接使用fetch，绕过CSRF保护
    let url = this.getUrl('/api/v1/auth/password/validate-token')

    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ token }),
    })

    const data: APIResponse<{ valid: boolean }> = await response.json()

    if (!data.success || !data.data) {
      throw APIException.fromAPIError(data.error!)
    }

    return data.data
  }
}

export const api = new APIClient()
