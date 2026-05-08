"use client"

import { useState, useEffect } from "react"
import { useRouter } from "@/i18n/routing"
import { useSearchParams } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PasswordStrength } from "@/components/ui/password-strength"
import { api } from "@/lib/api"
import { useErrorHandler } from "@/lib/use-error-handler"
import { validatePassword, PasswordStrength as StrengthLevel } from "@/lib/validation"
import { Link } from "@/i18n/routing"
import { Lock, Eye, EyeOff, Check, CheckCircle, XCircle } from "lucide-react"

export default function ResetPasswordPage() {
  const t = useTranslations('auth')
  const tc = useTranslations('common')
  const tv = useTranslations('validation')
  const { getError } = useErrorHandler()
  const router = useRouter()
  const searchParams = useSearchParams()

  const [token, setToken] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [success, setSuccess] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)
  const [tokenValid, setTokenValid] = useState<boolean | null>(null)
  const [validating, setValidating] = useState(true)
  const [passwordError, setPasswordError] = useState<string | undefined>()

  // 使用与注册页面完全相同的密码验证逻辑
  const passwordResult = validatePassword(password)
  const passwordsMatch = confirmPassword.length > 0 && password === confirmPassword

  // 实时验证密码
  useEffect(() => {
    if (password) {
      setPasswordError(passwordResult.error)
    } else {
      setPasswordError(undefined)
    }
  }, [password, passwordResult.error])

  // 验证token — 仅从 hash fragment 读取（不泄露到服务器日志/Referrer）
  useEffect(() => {
    let resetToken: string | null = null

    // 从 hash 中读取 (#token=xxx)
    const hash = window.location.hash
    if (hash && hash.startsWith('#token=')) {
      resetToken = hash.substring('#token='.length)
    }

    if (!resetToken) {
      setTokenValid(false)
      setValidating(false)
      return
    }
    // 清除 URL 中的 token，防止泄露到浏览器历史/Referrer/日志
    window.history.replaceState({}, '', window.location.pathname)
    setToken(resetToken)
    validateTokenFn(resetToken)
  }, [searchParams])

  const validateTokenFn = async (resetToken: string) => {
    try {
      await api.validateResetToken(resetToken)
      setTokenValid(true)
    } catch {
      setTokenValid(false)
    } finally {
      setValidating(false)
    }
  }

  const translateError = (errorKey?: string): string | undefined => {
    if (!errorKey) return undefined
    const path = errorKey.replace('validation.', '')
    const parts = path.split('.')
    if (parts.length === 2) {
      try { return tv(`${parts[0]}.${parts[1]}`) } catch { return errorKey }
    }
    return errorKey
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    if (passwordError) {
      setError(translateError(passwordError) || "")
      setLoading(false)
      return
    }

    if (password !== confirmPassword) {
      setError(tv('password.mismatch'))
      setLoading(false)
      return
    }

    try {
      await api.resetPassword(token, password)
      setSuccess(true)
      setTimeout(() => { router.push("/login") }, 3000)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }

  const isFormValid = password && confirmPassword && !passwordError && passwordsMatch
  const passwordStrengthState = { strength: passwordResult.strength, score: passwordResult.strength + 1 }

  // ---- 状态页：验证中 ----
  if (validating) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-12 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-4 border-blue-600 mx-auto mb-4"></div>
            <p className="text-sm text-gray-500">{t('validatingToken')}</p>
          </div>
        </div>
      </div>
    )
  }

  // ---- 状态页：token 无效 ----
  if (tokenValid === false) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8 text-center">
            <XCircle className="h-16 w-16 text-red-500 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-900 mb-2">
              {t('invalidToken')}
            </h2>
            <p className="text-sm text-gray-600 mb-6">
              {t('invalidTokenDescription')}
            </p>
            <Link
              href="/forgot-password"
              className="inline-flex items-center justify-center w-full h-11 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
            >
              {t('requestNewToken')}
            </Link>
          </div>
          <p className="mt-8 text-center text-xs text-gray-500">{tc('copyright')}</p>
        </div>
      </div>
    )
  }

  // ---- 状态页：重置成功 ----
  if (success) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8 text-center">
            <CheckCircle className="h-16 w-16 text-green-500 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-900 mb-2">
              {t('passwordResetSuccess')}
            </h2>
            <p className="text-sm text-gray-600 mb-4">
              {t('passwordResetSuccessDescription')}
            </p>
            <div className="flex items-center justify-center text-blue-600">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
              <span className="text-sm font-medium">{t('redirectingToLogin')}</span>
            </div>
          </div>
          <p className="mt-8 text-center text-xs text-gray-500">{tc('copyright')}</p>
        </div>
      </div>
    )
  }

  // ---- 主表单 ----
  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          <h2 className="mt-6 text-2xl font-semibold text-gray-900">
            {t('resetPassword')}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('resetPasswordDescription')}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* New Password */}
            <div>
              <Label htmlFor="password" className="block text-sm font-medium text-gray-700 mb-2">
                {t('newPassword')}
              </Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder={t('newPasswordPlaceholder')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className={`pl-10 pr-10 h-11 ${
                    passwordError
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
                  <PasswordStrength strength={passwordStrengthState.strength} score={passwordStrengthState.score} />
                </div>
              )}
              {passwordError && (
                <p className="mt-1.5 text-sm text-red-600">
                  {translateError(passwordError)}
                </p>
              )}
            </div>

            {/* Password Requirements — 与注册页完全一致 */}
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
                </ul>
              </div>
            )}

            {/* Confirm Password */}
            <div>
              <Label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 mb-2">
                {t('confirmPassword')}
              </Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? "text" : "password"}
                  placeholder={t('confirmPasswordPlaceholder')}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className={`pl-10 pr-10 h-11 ${
                    confirmPassword.length > 0 && passwordsMatch
                      ? 'border-green-300 focus:border-green-500 focus:ring-green-500'
                      : confirmPassword.length > 0 && !passwordsMatch
                      ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
                      : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500'
                  }`}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  aria-label={showConfirmPassword ? t('hidePassword') : t('showPassword')}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                >
                  {showConfirmPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>
              {confirmPassword.length > 0 && (
                <p className={`mt-1.5 text-sm flex items-center ${passwordsMatch ? 'text-green-600' : 'text-red-600'}`}>
                  {passwordsMatch ? (
                    <><CheckCircle className="h-4 w-4 mr-1.5" />{tv('password.match')}</>
                  ) : (
                    <><XCircle className="h-4 w-4 mr-1.5" />{tv('password.mismatch')}</>
                  )}
                </p>
              )}
            </div>

            {/* Error Message — 与注册页风格一致 */}
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {/* Submit Button — 与其他页面风格一致 */}
            <Button
              type="submit"
              className="w-full h-11 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
              disabled={loading || !isFormValid}
            >
              {loading ? t('resetting') : t('resetPassword')}
            </Button>

            {/* Back to Login Link — 与注册页/登录页风格一致 */}
            <div className="text-center text-sm">
              <Link
                href="/login"
                className="font-medium text-blue-600 hover:text-blue-700"
              >
                {t('backToLogin')}
              </Link>
            </div>
          </form>
        </div>

        {/* Footer — 与登录页/注册页一致 */}
        <p className="mt-8 text-center text-xs text-gray-500">
          {tc('copyright')}
        </p>
      </div>
    </div>
  )
}
