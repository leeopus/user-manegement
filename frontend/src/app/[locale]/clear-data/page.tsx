"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export default function ClearDataPage() {
  const t = useTranslations('clearData')
  const router = useRouter()
  const [cleared, setCleared] = useState(false)

  useEffect(() => {
    // 清除所有 localStorage 数据
    localStorage.clear()

    // Token 现在存储在 httpOnly cookie 中，会自动随浏览器关闭清除
    // 如果需要立即清除，需要调用后端登出 API
    const logout = async () => {
      try {
        await fetch('/api/v1/auth/logout', {
          method: 'POST',
          credentials: 'include', // 包含 cookie
        })
      } catch (error) {
        console.error('Logout error:', error)
      }
    }

    logout()

    setCleared(true)

    // 延迟后跳转
    setTimeout(() => {
      router.push('/login')
    }, 1500)
  }, [router])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        <Card>
          <CardHeader>
            <CardTitle>{t('title')}</CardTitle>
            <CardDescription>
              {cleared ? t('cleared') : t('clearing')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {cleared && (
              <div className="text-sm text-gray-600 text-center">
                {t('redirectHint')}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
