/**
 * CSRF Token 管理
 * Token 为一次性使用，每次请求前获取新 token
 */

const CSRF_TIMEOUT_MS = 5000

/**
 * 带超时的 fetch 封装
 */
function fetchWithTimeout(url: string, options: RequestInit, timeoutMs: number): Promise<Response> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs)
  return fetch(url, { ...options, signal: controller.signal }).finally(() => clearTimeout(timeoutId))
}

/**
 * 获取新的 CSRF token
 */
export async function getCSRFToken(): Promise<string> {
  const baseURL = process.env.NEXT_PUBLIC_API_URL || ''
  const url = baseURL ? `${baseURL}/api/csrf-token` : '/api/csrf-token'

  const response = await fetchWithTimeout(url, {
    method: 'GET',
    credentials: 'include',
  }, CSRF_TIMEOUT_MS)

  if (!response.ok) {
    throw new Error('Failed to fetch CSRF token')
  }

  const result = await response.json()

  if (!result.success || !result.data?.csrf_token) {
    throw new Error('Invalid CSRF token response')
  }

  return result.data.csrf_token
}

/**
 * 为 fetch 请求添加 CSRF token 到请求头
 * 每次调用都获取新 token（一次性使用）
 */
export async function addCSRFToHeaders(
  headers: HeadersInit
): Promise<HeadersInit> {
  const token = await getCSRFToken()

  return {
    ...headers,
    'X-CSRF-Token': token,
  }
}
