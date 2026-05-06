"use client"

import { useEffect } from 'react'
import { useLocale } from 'next-intl'
import { useRouter, usePathname } from '@/i18n/routing'
import { routing } from '@/i18n/routing'
import type { Locale } from '@/i18n/config'

export function LocalePreference() {
  const locale = useLocale() as Locale
  const router = useRouter()
  const pathname = usePathname()

  useEffect(() => {
    const savedLocale = localStorage.getItem('preferred-locale') as Locale

    if (savedLocale && savedLocale !== locale && routing.locales.includes(savedLocale)) {
      router.replace(pathname, { locale: savedLocale })
      return
    }

    if (!savedLocale) {
      const browserLang = navigator.language.split('-')[0] as Locale
      const supportedLocale = routing.locales.includes(browserLang) ? browserLang : routing.defaultLocale

      if (supportedLocale !== locale) {
        router.replace(pathname, { locale: supportedLocale })
      }
    }
  }, [locale, pathname, router])

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  return null
}
