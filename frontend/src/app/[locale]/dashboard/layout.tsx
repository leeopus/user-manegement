"use client"

import { useTranslations } from 'next-intl'
import { usePathname } from '@/i18n/routing'
import { Button } from "@/components/ui/button"
import { LanguageSwitcher } from "@/components/language-switcher"
import { Link } from "@/i18n/routing"
import { useAuth, hasPermission } from "@/lib/auth-provider"
import { OpusSystemLogo } from "@/components/opus-logo"

interface NavItem {
  href: string
  labelKey: string
  permission?: string
}

const navItems: NavItem[] = [
  { href: "/dashboard", labelKey: "home" },
  { href: "/dashboard/users", labelKey: "users", permission: "users:read" },
  { href: "/dashboard/roles", labelKey: "roles", permission: "roles:manage" },
  { href: "/dashboard/permissions", labelKey: "permissions", permission: "permissions:manage" },
  { href: "/dashboard/applications", labelKey: "ssoApps", permission: "oauth:manage" },
  { href: "/dashboard/audit-logs", labelKey: "auditLogs", permission: "audit:read" },
]

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const t = useTranslations('dashboard')
  const { user, logout, loading } = useAuth()
  const pathname = usePathname()

  if (loading) {
    return <div className="min-h-screen flex items-center justify-center">{t('loading', { fallback: 'Loading...' })}</div>
  }

  if (!user) {
    return null
  }

  const visibleNavItems = navItems.filter(
    item => !item.permission || hasPermission(user, item.permission)
  )

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <OpusSystemLogo system="account" />
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

      <nav className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4">
          <div className="flex space-x-8">
            {visibleNavItems.map(({ href, labelKey }) => {
              const isActive = pathname === href
              return (
                <Link
                  key={href}
                  href={href}
                  className={`py-4 px-2 text-sm font-medium border-b-2 ${
                    isActive
                      ? 'text-gray-900 border-gray-900'
                      : 'text-gray-600 border-transparent hover:text-gray-900'
                  }`}
                >
                  {t(labelKey)}
                </Link>
              )
            })}
          </div>
        </div>
      </nav>

      <main className="container mx-auto px-4 py-8">
        {children}
      </main>
    </div>
  )
}
