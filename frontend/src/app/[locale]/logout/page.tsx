"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-provider"
import { useTranslations } from "next-intl"

export default function LogoutPage() {
  const { logout } = useAuth()
  const t = useTranslations('auth')
  const [error, setError] = useState(false)

  useEffect(() => {
    logout().catch(() => setError(true))
  }, [logout])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100">
      <div className="text-center">
        {error ? (
          <p className="text-gray-500">{t('logoutError')}</p>
        ) : (
          <>
            <div className="inline-block h-6 w-6 animate-spin rounded-full border-2 border-gray-300 border-t-blue-600 mb-3" />
            <p className="text-gray-500">{t('loggingOut')}</p>
          </>
        )}
      </div>
    </div>
  )
}
