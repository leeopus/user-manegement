"use client"

import { useState, useEffect } from "react"
import { useRouter } from "@/i18n/routing"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PasswordStrength } from "@/components/ui/password-strength"
import { api } from "@/lib/api"
import { useErrorHandler } from "@/lib/use-error-handler"
import {
  validateEmail,
  validatePassword,
  PasswordStrength as StrengthLevel,
} from "@/lib/validation"
import { Link } from "@/i18n/routing"
import { Mail, Lock, Eye, EyeOff, Check } from "lucide-react"

interface ValidationErrors {
  email?: string
  password?: string
}

export default function RegisterPage() {
  const t = useTranslations('auth')
  const tc = useTranslations('common')
  const tv = useTranslations('validation')
  const router = useRouter()
  const { getError } = useErrorHandler()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({})
  const [passwordStrength, setPasswordStrength] = useState<{ strength: StrengthLevel; score: number }>({
    strength: StrengthLevel.Weak,
    score: 0,
  })
  const [showPassword, setShowPassword] = useState(false)

  // 翻译验证错误 key 为当前语言的文本
  const translateError = (errorKey?: string): string | undefined => {
    if (!errorKey) return undefined
    // errorKey 格式: "validation.username.required" -> 去掉 "validation." 前缀
    const path = errorKey.replace('validation.', '')
    const parts = path.split('.')
    if (parts.length === 2) {
      try {
        return tv(`${parts[0]}.${parts[1]}`)
      } catch {
        return errorKey
      }
    }
    return errorKey
  }

  // 实时验证邮箱
  useEffect(() => {
    if (email) {
      const result = validateEmail(email)
      setValidationErrors(prev => ({
        ...prev,
        email: result.valid ? undefined : result.error,
      }))
    } else {
      setValidationErrors(prev => ({ ...prev, email: undefined }))
    }
  }, [email])

  // 实时验证密码强度
  useEffect(() => {
    if (password) {
      const result = validatePassword(password, "")
      setPasswordStrength({
        strength: result.strength,
        score: result.strength + 1,
      })
      setValidationErrors(prev => ({
        ...prev,
        password: result.error,
      }))
    } else {
      setPasswordStrength({ strength: StrengthLevel.Weak, score: 0 })
      setValidationErrors(prev => ({ ...prev, password: undefined }))
    }
  }, [password])

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()

    const emailResult = validateEmail(email)
    const passwordResult = validatePassword(password, "")

    const errors: ValidationErrors = {}
    if (!emailResult.valid) errors.email = emailResult.error
    if (passwordResult.error) errors.password = passwordResult.error

    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors)
      return
    }

    setLoading(true)
    setError("")

    try {
      await api.register({ email, password })
      router.push("/login?registered=true")
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }

  const isFormValid = email && password && !validationErrors.email && !validationErrors.password

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          <h2 className="mt-6 text-2xl font-semibold text-gray-900">
            {t('createAccount')}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('registerDescription')}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8">
          <form onSubmit={handleRegister} className="space-y-6">
            {/* Email */}
            <div>
              <Label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-2">
                {t('email')}
              </Label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
                <Input
                  id="email"
                  type="email"
                  placeholder={t('emailPlaceholder')}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className={`pl-10 h-11 ${
                    validationErrors.email
                      ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500'
                  }`}
                  required
                />
                {email && !validationErrors.email && (
                  <Check className="absolute right-3 top-1/2 transform -translate-y-1/2 text-green-500 h-5 w-5" />
                )}
              </div>
              {validationErrors.email && (
                <p className="mt-1.5 text-sm text-red-600">
                  {translateError(validationErrors.email)}
                </p>
              )}
            </div>

            {/* Password */}
            <div>
              <Label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-2">
                {t('password')}
              </Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder={t('passwordPlaceholder')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={`pl-10 pr-10 h-11 ${
                    validationErrors.password
                      ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500'
                  }`}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  aria-label={showPassword ? t('hidePassword') : t('showPassword')}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                >
                  {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>
              {password && (
                <div className="mt-2">
                  <PasswordStrength strength={passwordStrength.strength} score={passwordStrength.score} />
                </div>
              )}
              {validationErrors.password && (
                <p className="mt-1.5 text-sm text-red-600">
                  {translateError(validationErrors.password)}
                </p>
              )}
            </div>

            {/* Password Requirements */}
            {password && (
              <div className="bg-blue-50 rounded-lg p-4 space-y-2">
                <p className="text-xs font-semibold text-gray-700">{tv('passwordRequirements')}</p>
                <ul className="text-xs text-gray-600 space-y-1.5">
                  <li className={`flex items-center ${password.length >= 8 ? 'text-green-600' : 'text-blue-600'}`}>
                    {password.length >= 8 ? (
                      <Check className="h-3 w-3 mr-2" />
                    ) : (
                      <div className="h-3 w-3 mr-2 border-2 border-blue-300 rounded-full" />
                    )}
                    {tv('requirements.length')}
                  </li>
                  <li className={`flex items-center ${/[a-z]/.test(password) && /\d/.test(password) ? 'text-green-600' : 'text-blue-600'}`}>
                    {/[a-z]/.test(password) && /\d/.test(password) ? (
                      <Check className="h-3 w-3 mr-2" />
                    ) : (
                      <div className="h-3 w-3 mr-2 border-2 border-blue-300 rounded-full" />
                    )}
                    {tv('requirements.complexity')}
                  </li>
                  <li className={`flex items-center ${passwordStrength.strength >= StrengthLevel.Good ? 'text-green-600' : 'text-blue-600'}`}>
                    {passwordStrength.strength >= StrengthLevel.Good ? (
                      <Check className="h-3 w-3 mr-2" />
                    ) : (
                      <div className="h-3 w-3 mr-2 border-2 border-blue-300 rounded-full" />
                    )}
                    {tv('passwordStrength')}
                  </li>
                </ul>
              </div>
            )}

            {/* Error Message */}
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {/* Sign Up Button */}
            <Button
              type="submit"
              className="w-full h-11 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
              disabled={loading || !isFormValid}
            >
              {loading ? t('signingUp') : t('signUp')}
            </Button>

            {/* Sign In Link */}
            <div className="text-center text-sm">
              <span className="text-gray-600">{t('hasAccount')}</span>{" "}
              <Link
                href="/login"
                className="font-medium text-blue-600 hover:text-blue-700"
              >
                {t('loginNow')}
              </Link>
            </div>
          </form>
        </div>

        {/* Footer */}
        <p className="mt-8 text-center text-xs text-gray-500">
          {tc('copyright')}
        </p>
      </div>
    </div>
  )
}
