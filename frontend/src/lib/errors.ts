import { APIError } from './types'

type TranslateFn = (key: string) => string

/**
 * API 错误类
 */
export class APIException extends Error {
  constructor(
    public code: string,
    public messageKey: string,
    public details?: Record<string, unknown>
  ) {
    super(messageKey)
    this.name = 'APIException'
  }

  /**
   * 获取本地化错误消息
   */
  getLocalizedMessage(t: TranslateFn): string {
    const message = t(this.messageKey)

    // 如果翻译键不存在，返回通用错误消息
    if (message === this.messageKey) {
      return t('UNKNOWN_ERROR')
    }

    // 如果有详情，可以用于格式化消息（可选）
    if (this.details) {
      // 这里可以根据 details 中的信息来增强错误消息
      // 例如：密钥长度要求等
    }

    return message
  }

  /**
   * 创建从 APIError 对象
   */
  static fromAPIError(apiError: APIError): APIException {
    return new APIException(
      apiError.code,
      apiError.message,
      apiError.details
    )
  }

  /**
   * 判断是否为某个错误码
   */
  isErrorCode(code: string): boolean {
    return this.code === code
  }

  /**
   * 判断是否为某个类别的错误
   */
  isCategory(category: string): boolean {
    return this.code.startsWith(`${category}_`)
  }

  /**
   * 获取错误类别（如 AUTH、VALIDATION、USER 等）
   */
  getCategory(): string {
    const parts = this.code.split('_')
    return parts[0] || 'UNKNOWN'
  }

  /**
   * 获取 HTTP 状态码
   */
  getHTTPStatus(): number {
    const parts = this.code.split('_')
    const statusCodeStr = parts[parts.length - 1]
    const statusCode = parseInt(statusCodeStr, 10)

    return isNaN(statusCode) ? 500 : statusCode
  }
}

/**
 * 判断错误是否为认证错误
 */
export function isAuthError(error: unknown): boolean {
  return error instanceof APIException && error.getCategory() === 'AUTH'
}

/**
 * 判断错误是否为验证错误
 */
export function isValidationError(error: unknown): boolean {
  return error instanceof APIException && error.getCategory() === 'VALIDATION'
}

/**
 * 判断是否为 401 错误
 */
export function isUnauthorized(error: unknown): boolean {
  return error instanceof APIException && error.getHTTPStatus() === 401
}

/**
 * 判断是否为网络错误
 */
export function isNetworkError(error: unknown): boolean {
  if (error instanceof Error) {
    return error.message.includes('Failed to fetch') ||
           error.message.includes('NetworkError') ||
           error.message.startsWith('NETWORK_ERROR:')
  }
  return false
}
