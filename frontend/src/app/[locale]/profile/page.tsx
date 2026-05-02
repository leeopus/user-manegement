"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

interface User {
  ID: number
  Username: string
  Email: string
  Avatar?: string
  Status?: string
  CreatedAt: string
}

export default function ProfilePage() {
  const t = useTranslations('profile')
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)

  useEffect(() => {
    const fetchProfile = async () => {
      try {
        // 通过API获取用户信息，而不是依赖localStorage
        const response = await fetch("http://localhost:8080/api/v1/auth/me", {
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        const data = await response.json()
        if (data.success && data.data) {
          setUser(data.data)
          // 仅用于显示，不包含敏感token信息
          localStorage.setItem("user", JSON.stringify(data.data))
        } else {
          router.push("/login")
        }
      } catch (err) {
        console.error("Failed to fetch profile:", err)
        router.push("/login")
      }
    }

    fetchProfile()
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
    return <div className="min-h-screen flex items-center justify-center">{t('status', { fallback: 'Loading...' })}</div>
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">{t('title')}</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              {user.Username || user.Email}
            </span>
            <Button variant="outline" size="sm" onClick={handleLogout}>
              {t('logout')}
            </Button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        <div className="max-w-2xl mx-auto">
          <Card>
            <CardHeader>
              <CardTitle>{t('personalInfo')}</CardTitle>
              <CardDescription>
                {t('description')}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('username')}</label>
                  <p className="text-lg">{user.Username}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('email')}</label>
                  <p className="text-lg">{user.Email}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('status')}</label>
                  <div>
                    <Badge variant={user.Status === "active" ? "default" : "secondary"}>
                      {user.Status === "active" ? t('active') : t('inactive')}
                    </Badge>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('registrationTime')}</label>
                  <p className="text-lg">{new Date(user.CreatedAt).toLocaleDateString()}</p>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-medium mb-3">{t('accessibleApps')}</h3>
                <div className="bg-gray-50 rounded-lg p-6 text-center text-gray-500">
                  <p>{t('noApps')}</p>
                  <p className="text-sm mt-2">{t('noAppsHint')}</p>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-medium mb-3">{t('quickLinks')}</h3>
                <div className="space-y-2">
                  <Button variant="outline" className="w-full justify-start" onClick={() => router.push("/dashboard")}>
                    🛠️ {t('adminPanel')}
                  </Button>
                  <p className="text-xs text-gray-500">{t('adminOnly')}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  )
}
