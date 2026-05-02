/**
 * 类型安全的国际化翻译系统
 *
 * 使用 TypeScript 严格类型约束，确保翻译键的正确性
 */

// =====================================================
// 翻译文件结构定义
// =====================================================

export interface TranslationModules {
  common: typeof import('@/messages/zh/common.json')
  errors: typeof import('@/messages/zh/errors.json')
  auth: typeof import('@/messages/zh/auth.json')
  profile: typeof import('@/messages/zh/profile.json')
  dashboard: typeof import('@/messages/zh/dashboard.json')
  users: typeof import('@/messages/zh/users.json')
  validation: typeof import('@/messages/zh/validation.json')
  passwordStrength: typeof import('@/messages/zh/passwordStrength.json')
  clearData: typeof import('@/messages/zh/clearData.json')
}

// =====================================================
// 翻译命名空间类型
// =====================================================

export type TranslationNamespace = keyof TranslationModules

// 提取某个命名空间的所有翻译键
export type TranslationKeys<N extends TranslationNamespace> = keyof TranslationModules[N]

// =====================================================
// 类型安全的翻译 Hook 参数类型
// =====================================================

export interface TypedTranslations<N extends TranslationNamespace> {
  /**
   * 获取翻译文本（类型安全）
   * @param key 翻译键，会有完整的类型提示和自动补全
   * @param params 可选的插值参数
   */
  t<K extends TranslationKeys<N>>(
    key: K,
    params?: Record<string, string | number>
  ): string

  /**
   * 获取翻译原始对象（用于特殊情况）
   */
  raw: TranslationModules[N]
}

// =====================================================
// 翻译完整性验证类型
// =====================================================

export type TranslationFileStructure = {
  [N in TranslationNamespace]: Record<string, unknown>
}

export type LocaleType = 'zh' | 'en' | 'ja' | 'ko' | 'fr' | 'de' | 'es' | 'pt'

export interface LocaleConfig {
  default: LocaleType
  available: LocaleType[]
}
