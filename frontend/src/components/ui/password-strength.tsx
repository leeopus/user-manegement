"use client"

import { PasswordStrength as StrengthLevel } from "@/lib/validation"

interface PasswordStrengthProps {
  strength: StrengthLevel
  score?: number
}

export function PasswordStrength({ strength, score = 0 }: PasswordStrengthProps) {
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
        return "弱"
      case StrengthLevel.Fair:
        return "中等"
      case StrengthLevel.Good:
        return "强"
      case StrengthLevel.Strong:
        return "很强"
      default:
        return "未知"
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
        密码强度: <span className={`font-semibold ${
          strength === StrengthLevel.Weak ? "text-red-600" :
          strength === StrengthLevel.Fair ? "text-yellow-600" :
          strength === StrengthLevel.Good ? "text-green-600" :
          "text-emerald-600"
        }`}>{getStrengthText(strength)}</span>
      </p>
    </div>
  )
}
