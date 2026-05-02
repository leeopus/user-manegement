"use client"

import { useTranslations } from 'next-intl'
import { PasswordStrength as StrengthLevel } from "@/lib/validation"

interface PasswordStrengthProps {
  strength: StrengthLevel
  score?: number
}

export function PasswordStrength({ strength, score = 0 }: PasswordStrengthProps) {
  const t = useTranslations('passwordStrength')

  const getStrengthColor = (level: StrengthLevel) => {
    switch (level) {
      case StrengthLevel.Weak:
        return "bg-red-500"
      case StrengthLevel.Fair:
        return "bg-yellow-500"
      case StrengthLevel.Good:
        return "bg-green-500"
      case StrengthLevel.Strong:
        return "bg-emerald-600"
      default:
        return "bg-gray-300"
    }
  }

  const getStrengthText = (level: StrengthLevel) => {
    switch (level) {
      case StrengthLevel.Weak:
        return t('weak')
      case StrengthLevel.Fair:
        return t('medium')
      case StrengthLevel.Good:
        return t('strong')
      case StrengthLevel.Strong:
        return t('veryStrong')
      default:
        return t('unknown')
    }
  }

  const getStrengthBarWidth = (level: StrengthLevel) => {
    switch (level) {
      case StrengthLevel.Weak:
        return "w-1/4"
      case StrengthLevel.Fair:
        return "w-2/4"
      case StrengthLevel.Good:
        return "w-3/4"
      case StrengthLevel.Strong:
        return "w-full"
      default:
      return "w-0"
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex gap-1">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className={`h-1.5 flex-1 rounded-full transition-colors ${
              i <= (score || 0) ? getStrengthColor(strength) : "bg-gray-200"
            }`}
          />
        ))}
      </div>
      <p className="text-xs text-gray-600">
        {t('label', { fallback: '密码强度：' })} {getStrengthText(strength)}
      </p>
    </div>
  )
}
