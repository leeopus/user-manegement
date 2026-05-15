"use client"

import { useState, useRef, useCallback } from "react"
import { useRouter } from "@/i18n/routing"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { useAuth } from "@/lib/auth-provider"
import { AvatarUpload } from "@/components/avatar-upload"
import { AccountDeactivation } from "@/components/account-deactivation"
import { api } from "@/lib/api"
import type { UpdateProfileRequest } from "@/lib/types"

type EditingField = 'nickname' | 'bio' | null

export default function ProfilePage() {
  const t = useTranslations('profile')
  const router = useRouter()
  const { user, logout, loading: authLoading, refreshUser } = useAuth()

  const [editing, setEditing] = useState<EditingField>(null)
  const [editValue, setEditValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [resending, setResending] = useState(false)
  const [resent, setResent] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const startEdit = useCallback((field: EditingField) => {
    if (!user || saving) return
    setEditing(field)
    setError('')
    if (field === 'nickname') setEditValue(user.nickname || '')
    else if (field === 'bio') setEditValue(user.bio || '')
  }, [user, saving])

  const cancelEdit = useCallback(() => {
    setEditing(null)
    setEditValue('')
    setError('')
  }, [])

  const saveEdit = useCallback(async () => {
    if (!user || !editing || saving) return

    setSaving(true)
    setError('')

    try {
      const data: UpdateProfileRequest = {}
      if (editing === 'nickname') data.nickname = editValue
      else if (editing === 'bio') data.bio = editValue

      await api.updateProfile(data)
      await refreshUser()
      setEditing(null)
      setEditValue('')
    } catch (err: any) {
      const code = err?.code || ''
      if (code.includes('NICKNAME_ALREADY_EXISTS')) {
        setError(t('nicknameTaken'))
      } else if (code.includes('NICKNAME_INVALID')) {
        setError(t('nicknameInvalid'))
      } else if (code.includes('NICKNAME_COOLDOWN')) {
        const hours = err?.details?.remaining_hours || 24
        setError(t('nicknameCooldown', { hours }))
      } else {
        setError(err?.message || t('saveFailed'))
      }
    } finally {
      setSaving(false)
    }
  }, [user, editing, saving, editValue, t, refreshUser])

  if (authLoading) {
    return <div className="min-h-screen flex items-center justify-center">Loading...</div>
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
              {user.nickname || user.username || user.email}
            </span>
            <Button variant="outline" size="sm" onClick={logout}>
              {t('logout')}
            </Button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        <div className="max-w-2xl mx-auto space-y-6">
          {/* Email verification banner */}
          {user && !user.email_verified_at && (
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
              <div className="flex items-start gap-3">
                <span className="text-amber-600 text-lg">&#9888;</span>
                <div className="flex-1">
                  <p className="font-medium text-amber-800">{t('emailNotVerified')}</p>
                  <p className="text-sm text-amber-700 mt-1">{t('emailNotVerifiedWarning')}</p>
                  <div className="mt-2">
                    {resent ? (
                      <span className="text-sm text-green-600">{t('verificationSent', { email: user.email })}</span>
                    ) : (
                      <Button
                        size="sm"
                        variant="outline"
                        className="border-amber-300 text-amber-800 hover:bg-amber-100"
                        disabled={resending}
                        onClick={async () => {
                          setResending(true)
                          try {
                            await api.resendEmailVerification()
                            setResent(true)
                          } catch { }
                          setResending(false)
                        }}
                      >
                        {resending ? '...' : t('resendVerification')}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Avatar */}
          <Card>
            <CardContent className="pt-6 flex items-center gap-6">
              <AvatarUpload />
              <div>
                <h2 className="text-xl font-semibold">{user.nickname || user.username}</h2>
                <p className="text-sm text-gray-500">{user.email}</p>
                <div className="mt-1">
                  <Badge variant={user.status === "active" ? "default" : "secondary"}>
                    {user.status === "active" ? t('active') : t('inactive')}
                  </Badge>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Editable fields */}
          <Card>
            <CardHeader>
              <CardTitle>{t('personalInfo')}</CardTitle>
              <CardDescription>{t('editHint')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Nickname - editable */}
              <div className="space-y-1">
                <Label className="text-sm font-medium text-gray-600">{t('nickname')}</Label>
                {editing === 'nickname' ? (
                  <div className="space-y-2">
                    <Input
                      value={editValue}
                      onChange={e => setEditValue(e.target.value)}
                      maxLength={50}
                      autoFocus
                    />
                    {error && <p className="text-sm text-red-500">{error}</p>}
                    <div className="flex gap-2">
                      <Button size="sm" onClick={saveEdit} disabled={saving}>
                        {saving ? t('saving') : t('save')}
                      </Button>
                      <Button size="sm" variant="outline" onClick={cancelEdit} disabled={saving}>
                        {t('cancel')}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div
                    className="flex items-center gap-2 cursor-pointer group"
                    onClick={() => startEdit('nickname')}
                  >
                    <span className="text-lg">{user.nickname || t('noNickname')}</span>
                    <span className="text-xs text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity">
                      {t('editHint')}
                    </span>
                  </div>
                )}
              </div>

              {/* Bio - editable */}
              <div className="space-y-1">
                <Label className="text-sm font-medium text-gray-600">{t('bio')}</Label>
                {editing === 'bio' ? (
                  <div className="space-y-2">
                    <Textarea
                      ref={textareaRef}
                      value={editValue}
                      onChange={e => setEditValue(e.target.value)}
                      maxLength={500}
                      rows={3}
                      autoFocus
                    />
                    {error && <p className="text-sm text-red-500">{error}</p>}
                    <div className="flex gap-2">
                      <Button size="sm" onClick={saveEdit} disabled={saving}>
                        {saving ? t('saving') : t('save')}
                      </Button>
                      <Button size="sm" variant="outline" onClick={cancelEdit} disabled={saving}>
                        {t('cancel')}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div
                    className="flex items-center gap-2 cursor-pointer group"
                    onClick={() => startEdit('bio')}
                  >
                    <span className="text-lg whitespace-pre-wrap">{user.bio || '-'}</span>
                    <span className="text-xs text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity">
                      {t('editHint')}
                    </span>
                  </div>
                )}
              </div>

              {/* Read-only fields */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div className="space-y-1">
                  <Label className="text-sm font-medium text-gray-600">{t('email')}</Label>
                  <p className="text-lg text-gray-800">{user.email}</p>
                </div>
                <div className="space-y-1">
                  <Label className="text-sm font-medium text-gray-600">{t('registrationTime')}</Label>
                  <p className="text-lg">{user.created_at ? new Date(user.created_at).toLocaleDateString() : '-'}</p>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div className="space-y-1">
                  <Label className="text-sm font-medium text-gray-600">{t('status')}</Label>
                  <Badge variant={user.status === "active" ? "default" : "secondary"}>
                    {user.status === "active" ? t('active') : t('inactive')}
                  </Badge>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-medium mb-3">{t('accessibleApps')}</h3>
                <div className="bg-gray-50 rounded-lg p-6 text-center text-gray-500">
                  <p>{t('noApps')}</p>
                  <p className="text-sm mt-2">{t('noAppsHint')}</p>
                </div>
              </div>

              {user.roles?.some(r => r.code === 'admin') && (
                <div className="pt-4 border-t">
                  <h3 className="font-medium mb-3">{t('quickLinks')}</h3>
                  <div className="space-y-2">
                    <Button variant="outline" className="w-full justify-start" onClick={() => router.push("/dashboard")}>
                      {t('adminPanel')}
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
          {/* Danger zone */}
          <AccountDeactivation />
      </main>
    </div>
  )
}
