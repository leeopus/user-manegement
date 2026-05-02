"use client"

import { useEffect } from 'react'
import { useLocale } from 'next-intl'
import { useRouter, usePathname } from 'next/navigation'
import { routing } from '@/i18n/routing'
import type { Locale } from '@/i18n/config'

/**
 * 客户端组件，用于处理语言偏好和自动检测
 * 按照主流国际化最佳实践实现
 */
export function LocalePreference() {
  const locale = useLocale() as Locale
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    // 1. 检查是否有保存的语言偏好
    const savedLocale = localStorage.getItem('preferred-locale') as Locale

    // 2. 如果保存的语言与当前不同，切换到保存的语言
    if (savedLocale && savedLocale !== locale && routing.locales.includes(savedLocale)) {
      const pathnameWithoutLocale = pathname.replace(`/${locale}`, '').replace(/^\//, '')
      router.replace(`/${savedLocale}${pathnameWithoutLocale ? '/' + pathnameWithoutLocale : ''}`)
      return
    }

    // 3. 如果没有保存偏好，尝试检测浏览器语言
    if (!savedLocale) {
      const browserLang = navigator.language.split('-')[0] as Locale
      const supportedLocale = routing.locales.includes(browserLang) ? browserLang : routing.defaultLocale

      if (supportedLocale !== locale) {
        const pathnameWithoutLocale = pathname.replace(`/${locale}`, '').replace(/^\//, '')
        router.replace(`/${supportedLocale}${pathnameWithoutLocale ? '/' + pathnameWithoutLocale : ''}`)
      }
    }
  }, [locale, pathname, router])

  // 在客户端设置HTML lang属性
  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  return null
}
