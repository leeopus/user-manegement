/**
 * 类型安全的翻译 Hook
 *
 * 提供完整的 TypeScript 类型检查和 IDE 自动补全
 */

import { useTranslations as useNextIntlTranslations } from 'next-intl'
import type {
  TranslationNamespace,
  TypedTranslations,
} from './types'

/**
 * 类型安全的翻译 Hook
 *
 * @example
 * ```tsx
 * // 使用 auth 命名空间
 * const { t } = useTypedTranslations('auth')
 *
 * // ✅ 完整的类型提示和自动补全
 * t('login')
 * t('register')
 * t('loginFailed')
 *
 * // ❌ TypeScript 编译错误：键不存在
 * // t('typo')
 * // t('common.submit')  // 错误的命名空间
 *
 * // 带参数的翻译
 * t('welcome', { name: 'John' })
 * ```
 */
export function useTypedTranslations<N extends TranslationNamespace>(
  namespace: N
): TypedTranslations<N> {
  const t = useNextIntlTranslations(namespace)

  return {
    // 严格模式：只允许使用类型定义中的键
    t: ((key: unknown, params?: Record<string, string | number>) => {
      // TypeScript 会在编译时检查 key 的类型
      return t(key as string, params)
    }) as TypedTranslations<N>['t'],

    // 获取原始翻译对象
    raw: (t.raw as unknown) as TypedTranslations<N>['raw'],
  }
}

/**
 * 便捷的 Hook 获取多个命名空间
 *
 * @example
 * ```tsx
 * const { t: tAuth } = useTypedTranslations('auth')
 * const { t: tCommon } = useTypedTranslations('common')
 * const { t: tErrors } = useTypedTranslations('errors')
 * ```
 */
export function useTranslationNamespace<N extends TranslationNamespace>(
  namespace: N
) {
  return useTypedTranslations(namespace)
}

/**
 * 获取错误翻译的便捷 Hook
 *
 * @example
 * ```tsx
 * const { getError } = useErrorTranslations()
 * const errorMsg = getError('AUTH_LOGIN_INVALID_CREDENTIALS')
 * ```
 */
export function useErrorTranslations() {
  const { t } = useTypedTranslations('errors')

  return {
    getError: (errorCode: string) => {
      const message = t(errorCode as any)
      // 如果返回的是键本身，说明翻译不存在
      if (message === errorCode) {
        return t('UNKNOWN_ERROR')
      }
      return message
    },
  }
}
