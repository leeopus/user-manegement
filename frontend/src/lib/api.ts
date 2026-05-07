'use client'

import {
  APIResponse,
  LoginRequest,
  LoginResponseData,
  RegisterRequest,
  RegisterResponseData,
  User,
  Role,
  Permission,
  OAuthApplication,
  OAuthApplicationCreateResult,
  AuditLog,
  CreateUserRequest,
  UpdateUserRequest,
  CreateRoleRequest,
  UpdateRoleRequest,
  CreatePermissionRequest,
  UpdatePermissionRequest,
  CreateOAuthAppRequest,
  UpdateOAuthAppRequest,
} from './types'
import { APIException } from './errors'
import { addCSRFToHeaders } from './csrf'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || ''

const CSRF_REQUIRED_METHODS = ['POST', 'PUT', 'DELETE', 'PATCH']
const REQUEST_TIMEOUT_MS = 30000

// Token 刷新锁和重试队列，仅在客户端初始化，避免 SSR 环境下跨请求状态污染
const isClient = typeof window !== 'undefined'

// 客户端专用可变状态，通过闭包隔离，SSR 不会初始化
let clientState: {
  isRefreshing: boolean
  refreshPromise: Promise<boolean> | null
  lastRefreshAttempt: number
  retryQueue: Array<() => void>
  isRetrying: boolean
} | null = null

function getState() {
  if (!isClient) return null
  if (!clientState) {
    clientState = {
      isRefreshing: false,
      refreshPromise: null,
      lastRefreshAttempt: 0,
      retryQueue: [],
      isRetrying: false,
    }
  }
  return clientState
}

const REFRESH_COOLDOWN_MS = 5000

function processRetryQueue() {
  const st = getState()
  if (!st || st.isRetrying || st.retryQueue.length === 0) return
  st.isRetrying = true
  const next = st.retryQueue.shift()
  if (next) {
    next()
  }
}

// 跨标签页同步：当一个标签页刷新 token 后，通知其他标签页重新获取用户信息
let authChannel: BroadcastChannel | null = null

function getAuthChannel(): BroadcastChannel | null {
  if (authChannel) return authChannel
  if (typeof window !== 'undefined' && typeof BroadcastChannel !== 'undefined') {
    authChannel = new BroadcastChannel('auth_sync')
    authChannel.onmessage = (event: MessageEvent) => {
      const st = getState()
      if (!st) return
      if (event.data?.type === 'TOKEN_REFRESHED') {
        st.isRefreshing = false
        st.refreshPromise = null
      } else if (event.data?.type === 'LOGOUT') {
        st.isRefreshing = false
        st.refreshPromise = null
      }
    }
  }
  return authChannel
}

async function refreshAccessToken(): Promise<boolean> {
  const st = getState()
  if (!st) return false

  const now = Date.now()
  if (now - st.lastRefreshAttempt < REFRESH_COOLDOWN_MS) {
    return false
  }

  if (st.isRefreshing && st.refreshPromise) {
    return st.refreshPromise
  }

  st.isRefreshing = true
  st.lastRefreshAttempt = now
  st.refreshPromise = (async () => {
    try {
      const baseURL = API_BASE
      const url = baseURL ? `${baseURL}/api/v1/auth/refresh` : '/api/v1/auth/refresh'

      let headers: Record<string, string> = { 'Content-Type': 'application/json' }
      try {
        headers = (await addCSRFToHeaders(headers)) as Record<string, string>
      } catch {
        // CSRF token 获取失败，仍然尝试刷新（httpOnly cookie 本身提供同站保护）
      }

      const response = await fetch(url, {
        method: 'POST',
        credentials: 'include',
        headers,
      })

      const data = await response.json()
      if (data.success === true) {
        getAuthChannel()?.postMessage({ type: 'TOKEN_REFRESHED' })
      }
      return data.success === true
    } catch {
      return false
    } finally {
      if (clientState) {
        clientState.isRefreshing = false
        clientState.refreshPromise = null
      }
    }
  })()

  return st.refreshPromise
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
        throw new Error('NETWORK_ERROR: Server returned an unexpected response. Please try again later.')
      }

      const data: APIResponse<T> = await response.json()

      // 401 且不是刷新请求本身 → 尝试刷新 token 并重试一次
      if (!data.success && data.error?.code === 'UNAUTHORIZED_401' && !path.includes('/auth/refresh')) {
        const refreshed = await refreshAccessToken()
        if (refreshed) {
          // Queue the retry to serialize concurrent retries (avoids CSRF race)
          return new Promise<APIResponse<T>>((resolve) => {
            const st = getState()
            if (!st) { resolve(data); return }
            st.retryQueue.push(async () => {
              try {
                let retryHeaders: Record<string, string> = {
                  'Content-Type': 'application/json',
                }

                if (options?.method && CSRF_REQUIRED_METHODS.includes(options.method)) {
                  try {
                    retryHeaders = (await addCSRFToHeaders(retryHeaders)) as Record<string, string>
                  } catch {
                    resolve(data)
                    return
                  }
                }

                const retryController = new AbortController()
                const retryTimeoutId = setTimeout(() => retryController.abort(), REQUEST_TIMEOUT_MS)
                try {
                  const retryResponse = await fetch(url, {
                    ...options,
                    headers: retryHeaders,
                    credentials: 'include',
                    signal: retryController.signal,
                  })

                  const retryContentType = retryResponse.headers.get('content-type') || ''
                  if (!retryContentType.includes('application/json')) {
                    resolve(data)
                    return
                  }

                  resolve(await retryResponse.json())
                } catch {
                  // 重试网络错误时返回原始 401 错误
                  resolve(data)
                } finally {
                  clearTimeout(retryTimeoutId)
                }
              } finally {
                const st2 = getState()
                if (st2) {
                  st2.isRetrying = false
                }
                processRetryQueue()
              }
            })
            processRetryQueue()
          })
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

    getAuthChannel()?.postMessage({ type: 'LOGOUT' })
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

  async listUsers(page = 1, pageSize = 10, filters?: { status?: string; search?: string }): Promise<{ users: User[]; total: number }> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    if (filters?.status) params.set('status', filters.status)
    if (filters?.search) params.set('search', filters.search)

    const response = await this.request<{ users: User[]; total: number }>(
      `/api/v1/users?${params.toString()}`,
      {}
    )

    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }

    return response.data
  }

  async updateUserStatus(id: number, status: string): Promise<void> {
    const response = await this.request<void>(`/api/v1/users/${id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status }),
    })

    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async getUser(id: number): Promise<User> {
    const response = await this.request<User>(`/api/v1/users/${id}`, {})
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async createUser(data: CreateUserRequest): Promise<User> {
    const response = await this.request<User>('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async updateUser(id: number, data: UpdateUserRequest): Promise<User> {
    const response = await this.request<User>(`/api/v1/users/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async deleteUser(id: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/users/${id}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async assignRole(userId: number, roleId: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/users/${userId}/roles`, {
      method: 'POST',
      body: JSON.stringify({ role_id: roleId }),
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async removeRole(userId: number, roleId: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/users/${userId}/roles/${roleId}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  // Role management
  async listRoles(page = 1, pageSize = 10): Promise<{ roles: Role[]; total: number }> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    const response = await this.request<{ roles: Role[]; total: number }>(
      `/api/v1/roles?${params.toString()}`,
      {}
    )
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async getRole(id: number): Promise<Role> {
    const response = await this.request<Role>(`/api/v1/roles/${id}`, {})
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async createRole(data: CreateRoleRequest): Promise<Role> {
    const response = await this.request<Role>('/api/v1/roles', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async updateRole(id: number, data: UpdateRoleRequest): Promise<Role> {
    const response = await this.request<Role>(`/api/v1/roles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async deleteRole(id: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/roles/${id}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async assignPermission(roleId: number, permissionId: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/roles/${roleId}/permissions`, {
      method: 'POST',
      body: JSON.stringify({ permission_id: permissionId }),
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  async removePermission(roleId: number, permissionId: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/roles/${roleId}/permissions/${permissionId}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  // Permission management
  async listPermissions(page = 1, pageSize = 10): Promise<{ permissions: Permission[]; total: number }> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    const response = await this.request<{ permissions: Permission[]; total: number }>(
      `/api/v1/permissions?${params.toString()}`,
      {}
    )
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async createPermission(data: CreatePermissionRequest): Promise<Permission> {
    const response = await this.request<Permission>('/api/v1/permissions', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async updatePermission(id: number, data: UpdatePermissionRequest): Promise<Permission> {
    const response = await this.request<Permission>(`/api/v1/permissions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async deletePermission(id: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/permissions/${id}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  // OAuth applications
  async listApplications(page = 1, pageSize = 10): Promise<{ applications: OAuthApplication[]; total: number }> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    const response = await this.request<{ applications: OAuthApplication[]; total: number }>(
      `/api/v1/oauth/applications?${params.toString()}`,
      {}
    )
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async createApplication(data: CreateOAuthAppRequest): Promise<OAuthApplicationCreateResult> {
    const response = await this.request<OAuthApplicationCreateResult>('/api/v1/oauth/applications', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async updateApplication(id: number, data: UpdateOAuthAppRequest): Promise<OAuthApplication> {
    const response = await this.request<OAuthApplication>(`/api/v1/oauth/applications/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }

  async deleteApplication(id: number): Promise<void> {
    const response = await this.request<void>(`/api/v1/oauth/applications/${id}`, {
      method: 'DELETE',
    })
    if (!response.success) {
      throw APIException.fromAPIError(response.error!)
    }
  }

  // Audit logs
  async listAuditLogs(page = 1, pageSize = 10, filters?: { action?: string; resource?: string; search?: string }): Promise<{ logs: AuditLog[]; total: number }> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    if (filters?.action) params.set('action', filters.action)
    if (filters?.resource) params.set('resource', filters.resource)
    if (filters?.search) params.set('search', filters.search)

    const response = await this.request<{ logs: AuditLog[]; total: number }>(
      `/api/v1/audit-logs?${params.toString()}`,
      {}
    )
    if (!response.success || !response.data) {
      throw APIException.fromAPIError(response.error!)
    }
    return response.data
  }
}

export const api = new APIClient()
