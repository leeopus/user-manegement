"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useTranslations, useLocale } from 'next-intl'
import { Button } from "@/components/ui/button"
import { LanguageSwitcher } from "@/components/language-switcher"
import { Link } from "@/i18n/routing"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const t = useTranslations('dashboard')
  const locale = useLocale()
  const router = useRouter()
  const [user, setUser] = useState<any>(null)

  useEffect(() => {
    const checkAuth = async () => {
      try {
        // 使用API验证身份，而不是依赖localStorage
        const response = await fetch("http://localhost:8080/api/v1/auth/me", {
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        const data = await response.json()
        if (data.success && data.data) {
          setUser(data.data)
          // 更新localStorage中的用户信息（仅用于显示，不含敏感数据）
          localStorage.setItem("user", JSON.stringify(data.data))
        } else {
          router.push("/login")
        }
      } catch (err) {
        console.error("Auth check failed:", err)
        router.push("/login")
      }
    }

    checkAuth()
  }, [router])

  const handleLogout = async () => {
    try {
      // 调用登出API清除httpOnly cookie
      await fetch("http://localhost:8080/api/v1/auth/logout", {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
      })
    } catch (err) {
      console.error("Logout failed:", err)
    } finally {
      // 清除本地用户信息
      localStorage.removeItem("user")
      router.push("/login")
    }
  }

  if (!user) {
    return <div>{t('loading', { fallback: 'Loading...' })}</div>
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
              {t('welcome')}{user.Username || user.Email}
            </span>
            <Button variant="outline" size="sm" onClick={handleLogout}>
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
