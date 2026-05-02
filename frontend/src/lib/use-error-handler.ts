/**
 * 错误处理 Hook
 *
 * 提供统一的错误处理和本地化功能
 */

import { useTranslations } from 'next-intl'
import { APIException } from './errors'

/**
 * 处理 API 错误并返回本地化消息
 */
function baseHandleAPIError(error: unknown, t: any): string {
  // 如果是 APIException，使用其方法获取本地化消息
  if (error instanceof APIException) {
    return error.getLocalizedMessage(t)
  }

  // 如果是普通 Error 对象
  if (error instanceof Error) {
    // 尝试从错误消息中提取错误码
    const message = error.message

    // 网络错误
    if (message.includes('Failed to fetch') || message.includes('NetworkError')) {
      return t('NETWORK_ERROR')
    }

    // 其他错误，尝试直接翻译
    const translated = t(message)
    if (translated !== message) {
      return translated
    }
  }

  // 默认返回未知错误
  return t('UNKNOWN_ERROR')
}

/**
 * 错误处理 Hook
 *
 * @example
 * ```tsx
 * function LoginPage() {
 *   const { getError } = useErrorHandler()
 *
 *   try {
 *     await api.login({ email, password })
 *   } catch (err) {
 *     setError(getError(err))
 *   }
 * }
 * ```
 */
export function useErrorHandler() {
  const t = useTranslations('errors')

  /**
   * 获取本地化错误消息
   */
  function getError(error: unknown): string {
    return baseHandleAPIError(error, t)
  }

  return {
    getError,
  }
}

/**
 * 兼容的 handleAPIError 函数
 *
 * @deprecated 推荐使用 useErrorHandler hook
 */
export function handleAPIError(error: unknown, t: any): string {
  return baseHandleAPIError(error, t)
}
