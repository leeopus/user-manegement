"use client"

import { useState } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth-provider"

export function AccountDeactivation() {
  const t = useTranslations('profile')
  const { logout } = useAuth()
  const [showDialog, setShowDialog] = useState(false)
  const [password, setPassword] = useState("")
  const [confirmed, setConfirmed] = useState(false)
  const [deactivating, setDeactivating] = useState(false)
  const [error, setError] = useState("")

  const handleDeactivate = async () => {
    if (!password || !confirmed) return

    setDeactivating(true)
    setError("")

    try {
      await api.deactivateAccount(password)
      await logout()
    } catch (err: any) {
      setError(err?.message || t('deactivateFailed'))
      setDeactivating(false)
    }
  }

  return (
    <Card className="border-red-200">
      <CardHeader>
        <CardTitle className="text-red-600">{t('dangerZone')}</CardTitle>
      </CardHeader>
      <CardContent>
        {!showDialog ? (
          <div>
            <p className="text-sm text-gray-600 mb-4">{t('deactivateWarning')}</p>
            <Button variant="destructive" onClick={() => setShowDialog(true)}>
              {t('deactivateAccount')}
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-gray-700">{t('deactivateConfirmText')}</p>
            <div>
              <Label>{t('enterPassword')}</Label>
              <Input
                type="password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder={t('currentPassword')}
                disabled={deactivating}
              />
            </div>
            <label className="flex items-start gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={confirmed}
                onChange={e => setConfirmed(e.target.checked)}
                disabled={deactivating}
                className="mt-1"
              />
              <span className="text-sm text-gray-700">{t('understandConsequences')}</span>
            </label>
            {error && <p className="text-sm text-red-500">{error}</p>}
            <div className="flex gap-2">
              <Button
                variant="destructive"
                onClick={handleDeactivate}
                disabled={deactivating || !password || !confirmed}
              >
                {deactivating ? '...' : t('confirmDeactivation')}
              </Button>
              <Button variant="outline" onClick={() => { setShowDialog(false); setPassword(""); setConfirmed(false); setError(""); }} disabled={deactivating}>
                {t('cancel')}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
