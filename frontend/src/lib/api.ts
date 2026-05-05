import {
  APIResponse,
  LoginRequest,
  LoginResponseData,
  RegisterRequest,
  RegisterResponseData,
  User,
} from './types'
import { APIException } from './errors'
import { addCSRFToHeaders } from './csrf'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

const CSRF_REQUIRED_METHODS = ['POST', 'PUT', 'DELETE', 'PATCH']
const REQUEST_TIMEOUT_MS = 30000

// Token 刷新锁，防止并发刷新
let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null
let lastRefreshAttempt = 0
const REFRESH_COOLDOWN_MS = 5000

// 跨标签页同步：当一个标签页刷新 token 后，通知其他标签页重新获取用户信息
const authChannel = typeof BroadcastChannel !== 'undefined'
  ? new BroadcastChannel('auth_sync')
  : null

if (authChannel) {
  authChannel.onmessage = (event: MessageEvent) => {
    if (event.data?.type === 'TOKEN_REFRESHED') {
      // 另一个标签页完成了 token 刷新，本标签页的 cookie 已更新
      // 重置刷新状态以便本标签页后续请求正常
      isRefreshing = false
      refreshPromise = null
    } else if (event.data?.type === 'LOGOUT') {
      // 另一个标签页登出，本标签页也需要清除状态
      isRefreshing = false
      refreshPromise = null
    }
  }
}

async function refreshAccessToken(): Promise<boolean> {
  const now = Date.now()
  if (now - lastRefreshAttempt < REFRESH_COOLDOWN_MS) {
    return false
  }

  if (isRefreshing && refreshPromise) {
    return refreshPromise
  }

  isRefreshing = true
  lastRefreshAttempt = now
  refreshPromise = (async () => {
    try {
      const baseURL = API_BASE
      const url = baseURL ? `${baseURL}/api/v1/auth/refresh` : '/api/v1/auth/refresh'

      const response = await fetch(url, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
      })

      const data = await response.json()
      if (data.success === true) {
        authChannel?.postMessage({ type: 'TOKEN_REFRESHED' })
      }
      return data.success === true
    } catch {
      return false
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

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

    // 创建 AbortController 用于请求超时
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)

    let headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options?.headers as Record<string, string>) || {}),
    }

    // 对于需要 CSRF 的方法，获取新 token 并添加
    if (options?.method && CSRF_REQUIRED_METHODS.includes(options.method)) {
      try {
        headers = (await addCSRFToHeaders(headers)) as Record<string, string>
      } catch (error) {
        // CSRF token 获取失败，明确抛出错误，不静默降级
        throw new Error('CSRF_TOKEN_UNAVAILABLE: Failed to obtain CSRF token. Please refresh the page and try again.')
      }
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
        credentials: 'include',
        signal: controller.signal,
      })

      // 防止非 JSON 响应（如 502/504 HTML 错误页）导致 JSON 解析失败
      const contentType = response.headers.get('content-type') || ''
      if (!contentType.includes('application/json')) {
        throw new Error(`NETWORK_ERROR: Server returned non-JSON response (HTTP ${response.status})`)
      }

      const data: APIResponse<T> = await response.json()

      // 401 且不是刷新请求本身 → 尝试刷新 token 并重试一次
      if (!data.success && data.error?.code === 'UNAUTHORIZED_401' && !path.includes('/auth/refresh')) {
        const refreshed = await refreshAccessToken()
        if (refreshed) {
          // 重试时重新获取 CSRF token（原 token 已被一次性消耗）
          let retryHeaders: Record<string, string> = {
            'Content-Type': 'application/json',
          }

          if (options?.method && CSRF_REQUIRED_METHODS.includes(options.method)) {
            try {
              retryHeaders = (await addCSRFToHeaders(retryHeaders)) as Record<string, string>
            } catch {
              // CSRF 获取失败时，不重试写操作，直接返回 401 错误让用户重新登录
              return data
            }
          }

          try {
            const retryResponse = await fetch(url, {
              ...options,
              headers: retryHeaders,
              credentials: 'include',
              signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
            })

            const retryContentType = retryResponse.headers.get('content-type') || ''
            if (!retryContentType.includes('application/json')) {
              return data
            }

            return await retryResponse.json()
          } catch {
            // 重试网络错误时返回原始 401 错误
            return data
          }
        }
      }

      return data
    } catch (error) {
      if (error instanceof Error && error.message.startsWith('CSRF_TOKEN_UNAVAILABLE')) {
        throw error
      }
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new Error('REQUEST_TIMEOUT: Request timed out. Please try again.')
      }
      throw new Error(`NETWORK_ERROR: ${error}`)
    } finally {
      clearTimeout(timeoutId)
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

  async getUserInfo(): Promise<User> {
    const response = await this.request<User>('/api/v1/auth/me', {})

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  async refreshToken(): Promise<{ user: User }> {
    const response = await this.request<{ user: User }>('/api/v1/auth/refresh', {
      method: 'POST',
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  async logout(): Promise<void> {
    const response = await this.request<void>('/api/v1/auth/logout', {
      method: 'POST',
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }

    authChannel?.postMessage({ type: 'LOGOUT' })
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
    const response = await this.request<void>('/api/v1/auth/password/reset', {
      method: 'POST',
      body: JSON.stringify({ token, new_password: newPassword }),
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async validateResetToken(token: string): Promise<{ valid: boolean }> {
    const response = await this.request<{ valid: boolean }>('/api/v1/auth/password/validate-token', {
      method: 'POST',
      body: JSON.stringify({ token }),
    })

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  async listUsers(page = 1, pageSize = 10): Promise<{ users: User[]; total: number }> {
    const response = await this.request<{ users: User[]; total: number }>(
      `/api/v1/users?page=${page}&page_size=${pageSize}`,
      {}
    )

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }
}

export const api = new APIClient()
