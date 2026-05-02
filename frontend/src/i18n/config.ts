export const defaultLocale = 'zh'
export const locales = ['zh', 'en'] as const

export const localeNames: Record<typeof locales[number], string> = {
  zh: '简体中文',
  en: 'English'
}

export const localeFlags: Record<typeof locales[number], string> = {
  zh: '🇨🇳',
  en: '🇺🇸'
}

export type Locale = typeof locales[number]
