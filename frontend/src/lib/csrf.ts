/**
 * CSRF Token 管理
 */

let csrfToken: string | null = null
let tokenExpiry: number | null = null

const TOKEN_CACHE_DURATION = 50 * 60 * 1000 // 50 分钟（token 1 小时过期）

/**
 * 获取 CSRF token
 */
export async function getCSRFToken(): Promise<string> {
  // 检查缓存
  if (csrfToken && tokenExpiry && Date.now() < tokenExpiry) {
    return csrfToken
  }

  // 获取新 token
  try {
    const response = await fetch('/api/csrf-token', {
      method: 'GET',
      credentials: 'include', // 包含 cookie
    })

    if (!response.ok) {
      throw new Error('Failed to fetch CSRF token')
    }

    const result = await response.json()

    if (!result.success || !result.data?.csrf_token) {
      throw new Error('Invalid CSRF token response')
    }

    csrfToken = result.data.csrf_token
    tokenExpiry = Date.now() + TOKEN_CACHE_DURATION

    return csrfToken
  } catch (error) {
    console.error('Failed to get CSRF token:', error)
    throw error
  }
}

/**
 * 清除缓存的 CSRF token
 */
export function clearCSRFToken(): void {
  csrfToken = null
  tokenExpiry = null
}

/**
 * 预加载 CSRF token（在用户操作前）
 */
export async function preloadCSRFToken(): Promise<void> {
  try {
    await getCSRFToken()
  } catch (error) {
    console.warn('Failed to preload CSRF token:', error)
  }
}

/**
 * 为 fetch 添加 CSRF token 到请求头
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
