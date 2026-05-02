"use client"

import { useLocale } from 'next-intl'
import { useRouter, usePathname } from 'next/navigation'
import { locales, localeNames, localeFlags, type Locale } from '@/i18n/config'
import { Languages } from "lucide-react"
import { useState } from "react"

export function LanguageSwitcher() {
  const locale = useLocale() as Locale
  const router = useRouter()
  const pathname = usePathname()
  const [isOpen, setIsOpen] = useState(false)

  const handleLanguageChange = (newLocale: string) => {
    // Save to localStorage for persistence
    if (typeof window !== 'undefined') {
      localStorage.setItem('preferred-locale', newLocale)
    }

    // Get the current pathname without the locale prefix
    const pathnameWithoutLocale = pathname.replace(`/${locale}`, '').replace(/^\//, '')

    // Navigate to the new locale
    router.push(`/${newLocale}${pathnameWithoutLocale ? '/' + pathnameWithoutLocale : ''}`)

    // Close the dropdown
    setIsOpen(false)
  }

  // Get current locale info
  const currentLocaleName = localeNames[locale]
  const currentLocaleFlag = localeFlags[locale]

  return (
    <div className="relative z-50">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium border border-gray-300 rounded-md hover:bg-blue-50 hover:border-blue-300 transition-colors bg-white shadow-sm"
        aria-label="Switch language"
      >
        <Languages className="h-4 w-4 text-gray-600" />
        <span className="hidden sm:inline">{currentLocaleFlag} <span className="text-gray-700">{currentLocaleName}</span></span>
        <span className="sm:hidden">{currentLocaleFlag}</span>
      </button>

      {isOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Dropdown Menu */}
          <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 z-50 overflow-hidden">
            <div className="py-1">
              {locales.map((loc) => {
                const isSelected = locale === loc
                return (
                  <button
                    key={loc}
                    onClick={() => handleLanguageChange(loc)}
                    className={`w-full text-left px-4 py-2.5 text-sm flex items-center gap-3 transition-colors ${
                      isSelected
                        ? 'bg-blue-50 text-blue-700 font-medium'
                        : 'text-gray-700 hover:bg-gray-50'
                    }`}
                  >
                    <span className="text-lg">{localeFlags[loc]}</span>
                    <span>{localeNames[loc]}</span>
                    {isSelected && (
                      <span className="ml-auto text-blue-600">✓</span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
