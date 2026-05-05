"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useAuth } from "@/lib/auth-provider"
import type { User } from "@/lib/types"

export default function ProfilePage() {
  const t = useTranslations('profile')
  const router = useRouter()
  const { user, logout, loading: authLoading } = useAuth()

  const handleLogout = async () => {
    await logout()
  }

  if (authLoading) {
    return <div className="min-h-screen flex items-center justify-center">{t('status', { fallback: 'Loading...' })}</div>
  }

  if (!user) {
    return null
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">{t('title')}</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              {user.username || user.email}
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
                  <p className="text-lg">{user.username}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('email')}</label>
                  <p className="text-lg">{user.email}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('status')}</label>
                  <div>
                    <Badge variant={user.status === "active" ? "default" : "secondary"}>
                      {user.status === "active" ? t('active') : t('inactive')}
                    </Badge>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">{t('registrationTime')}</label>
                  <p className="text-lg">{user.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}</p>
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
                    {t('adminPanel')}
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
