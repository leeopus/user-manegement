"use client"

import { useEffect, useState, Suspense } from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from 'next-intl'
import { api } from "@/lib/api"
import { useRouter } from "@/i18n/routing"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

function VerifyEmailContent() {
  const t = useTranslations('verifyEmail')
  const searchParams = useSearchParams()
  const router = useRouter()
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    const token = searchParams.get('token')
    if (!token) {
      setStatus('error')
      setErrorMessage(t('missingToken'))
      return
    }

    api.verifyEmail(token)
      .then(() => setStatus('success'))
      .catch((err) => {
        setStatus('error')
        setErrorMessage(err?.message || t('verificationFailed'))
      })
  }, [searchParams, t])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t('title')}</CardTitle>
        </CardHeader>
        <CardContent>
          {status === 'loading' && (
            <p className="text-gray-600">{t('verifying')}</p>
          )}
          {status === 'success' && (
            <div className="space-y-4">
              <p className="text-green-600">{t('success')}</p>
              <Button onClick={() => router.push('/profile')}>
                {t('goToProfile')}
              </Button>
            </div>
          )}
          {status === 'error' && (
            <div className="space-y-4">
              <p className="text-red-600">{errorMessage}</p>
              <Button variant="outline" onClick={() => router.push('/profile')}>
                {t('goToProfile')}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default function VerifyEmailPage() {
  return (
    <Suspense fallback={<div className="min-h-screen flex items-center justify-center">Loading...</div>}>
      <VerifyEmailContent />
    </Suspense>
  )
}
