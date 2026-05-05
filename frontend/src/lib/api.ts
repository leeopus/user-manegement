import {
  APIResponse,
  LoginRequest,
  LoginResponseData,
  RegisterRequest,
  RegisterResponseData,
  User,
} from './types'
import { APIException } from './errors'
import { getCSRFToken, addCSRFToHeaders } from './csrf'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

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
    let headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options?.headers as Record<string, string>) || {}),
    }

    // 对于需要 CSRF 的方法，添加 token
    if (options?.method && CSRF_REQUIRED_METHODS.includes(options.method)) {
      try {
        headers = (await addCSRFToHeaders(headers)) as Record<string, string>
      } catch (error) {
        console.error('Failed to add CSRF token:', error)
      }
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'include',
      })

      const data: APIResponse<T> = await response.json()

      // CSRF token 过期，清除缓存并重试一次
      if (!data.success && data.error?.code === 'CSRF_TOKEN_INVALID_403') {
        const { clearCSRFToken } = await import('./csrf')
        clearCSRFToken()

        headers = (await addCSRFToHeaders({
          'Content-Type': 'application/json',
          ...((options?.headers as Record<string, string>) || {}),
        })) as Record<string, string>

        const retryResponse = await fetch(url, {
          ...options,
          headers,
          credentials: 'include',
        })

        return await retryResponse.json()
      }

      return data
    } catch (error) {
      throw new Error(`NETWORK_ERROR: ${error}`)
    }
  }

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
   * 获取当前用户信息（token 通过 httpOnly cookie 自动携带）
   */
  async getUserInfo(_token?: string): Promise<User> {
    const response = await this.request<User>('/api/v1/auth/me', {})

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 刷新令牌（refresh token 通过 httpOnly cookie 自动携带）
   */
  async refreshToken(_refreshToken?: string): Promise<{ user: User }> {
    const response = await this.request<{ user: User }>('/api/v1/auth/refresh', {
      method: 'POST',
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  /**
   * 登出（token 通过 httpOnly cookie 自动携带）
   */
  async logout(_token?: string): Promise<void> {
    const response = await this.request<void>('/api/v1/auth/logout', {
      method: 'POST',
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async requestPasswordReset(email: string): Promise<void> {
    const response = await this.request<void>('/api/v1/auth/password/reset-request', {
      method: 'POST',
      body: JSON.stringify({ email }),
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async resetPassword(token: string, newPassword: string): Promise<void> {
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

  async validateResetToken(token: string): Promise<{ valid: boolean }> {
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
