"use client"

import { useRef, useState } from "react"
import { useTranslations } from "next-intl"
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth-provider"

export function AvatarUpload() {
  const t = useTranslations('profile')
  const { user, refreshUser } = useAuth()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState("")

  if (!user) return null

  const handleClick = () => {
    fileInputRef.current?.click()
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setError("")

    if (file.size > 2 * 1024 * 1024) {
      setError(t('avatarTooLarge'))
      return
    }

    const allowed = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
    if (!allowed.includes(file.type)) {
      setError(t('avatarInvalidType'))
      return
    }

    setUploading(true)
    try {
      await api.uploadAvatar(file)
      await refreshUser()
    } catch (err: any) {
      setError(err?.message || t('avatarUploadFailed'))
    } finally {
      setUploading(false)
      // Reset input so same file can be re-selected
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  return (
    <div className="relative group">
      <div
        className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center overflow-hidden cursor-pointer relative"
        onClick={handleClick}
      >
        {user.avatar ? (
          <img src={user.avatar} alt={user.username} className="w-full h-full object-cover" />
        ) : (
          <span className="text-2xl font-bold text-gray-500">
            {(user.nickname || user.username || '?')[0].toUpperCase()}
          </span>
        )}
        <div className="absolute inset-0 bg-black bg-opacity-40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
          <span className="text-white text-xs">{uploading ? '...' : t('changeAvatar', { fallback: 'Change' })}</span>
        </div>
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        onChange={handleFileChange}
        className="hidden"
      />
      {error && <p className="text-xs text-red-500 mt-1 max-w-[80px]">{error}</p>}
    </div>
  )
}
