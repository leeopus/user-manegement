"use client"

import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { LanguageSwitcher } from "@/components/language-switcher"
import { Link } from "@/i18n/routing"
import { useAuth } from "@/lib/auth-provider"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const t = useTranslations('dashboard')
  const { user, logout, loading } = useAuth()

  if (loading || !user) {
    return <div className="min-h-screen flex items-center justify-center">{t('loading', { fallback: 'Loading...' })}</div>
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">{t('title')}</h1>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <span className="text-sm text-gray-600">
              {t('welcome')}{user.username || user.email}
            </span>
            <Button variant="outline" size="sm" onClick={logout}>
              {t('logout')}
            </Button>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4">
          <div className="flex space-x-8">
            <Link
              href="/dashboard"
              className="py-4 px-2 text-sm font-medium text-gray-900 border-b-2 border-gray-900"
            >
              {t('home')}
            </Link>
            <Link
              href="/dashboard/users"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t('users')}
            </Link>
            <Link
              href="/dashboard/roles"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t('roles')}
            </Link>
            <Link
              href="/dashboard/permissions"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t('permissions')}
            </Link>
            <Link
              href="/dashboard/applications"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              {t('ssoApps')}
            </Link>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        {children}
      </main>
    </div>
  )
}
