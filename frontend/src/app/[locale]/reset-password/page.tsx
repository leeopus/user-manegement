"use client"

import { useState, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api } from "@/lib/api"
import { useErrorHandler } from "@/lib/use-error-handler"
import { Link } from "@/i18n/routing"
import { Lock, Eye, EyeOff, CheckCircle, XCircle, AlertCircle, Shield } from "lucide-react"

// 密码强度计算
function calculatePasswordStrength(password: string): { score: number; label: string; color: string; checks: boolean[] } {
  const checks = [
    password.length >= 8,
    /[a-z]/.test(password),
    /[0-9]/.test(password),
    /[^A-Za-z0-9]/.test(password),
  ]

  const score = checks.filter(Boolean).length

  let label = ""
  let color = ""

  if (score === 0) {
    label = "非常弱"
    color = "bg-red-500"
  } else if (score === 1) {
    label = "弱"
    color = "bg-red-400"
  } else if (score === 2) {
    label = "中等"
    color = "bg-yellow-400"
  } else if (score === 3) {
    label = "强"
    color = "bg-green-400"
  } else {
    label = "很强"
    color = "bg-green-500"
  }

  return { score, label, color, checks }
}

export default function ResetPasswordPage() {
  const t = useTranslations('auth')
  const tc = useTranslations('common')
  const te = useTranslations('errors')
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

  // 密码强度计算
  const passwordStrength = calculatePasswordStrength(password)
  const passwordsMatch = confirmPassword.length > 0 && password === confirmPassword
  const passwordLongEnough = password.length >= 8

  // 验证token
  useEffect(() => {
    const resetToken = searchParams.get("token")
    if (!resetToken) {
      setTokenValid(false)
      setValidating(false)
      return
    }

    setToken(resetToken)
    validateToken(resetToken)
  }, [searchParams])

  const validateToken = async (resetToken: string) => {
    try {
      await api.validateResetToken(resetToken)
      setTokenValid(true)
    } catch (err) {
      setTokenValid(false)
    } finally {
      setValidating(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    // 验证密码
    if (password.length < 8) {
      setError(te('VALIDATION_PASSWORD_TOO_SHORT'))
      setLoading(false)
      return
    }

    if (passwordStrength.score < 2) {
      setError(te('VALIDATION_PASSWORD_TOO_WEAK'))
      setLoading(false)
      return
    }

    if (password !== confirmPassword) {
      setError(t('passwordMismatch'))
      setLoading(false)
      return
    }

    try {
      await api.resetPassword(token, password)
      setSuccess(true)

      // 3秒后跳转到登录页
      setTimeout(() => {
        // 使用当前语言跳转到登录页
        const currentLocale = router.locale || 'zh'
        router.push(`/${currentLocale}/login`)
      }, 3000)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }

  if (validating) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="bg-white rounded-xl shadow-lg border border-gray-200 px-8 py-12 text-center">
            <div className="animate-spin rounded-full h-16 w-16 border-b-4 border-blue-600 mx-auto mb-6"></div>
            <h2 className="text-xl font-semibold text-gray-900 mb-2">
              {t('validatingToken')}
            </h2>
            <p className="text-sm text-gray-500">
              正在验证您的重置链接...
            </p>
          </div>
        </div>
      </div>
    )
  }

  if (tokenValid === false) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="bg-white rounded-xl shadow-lg border border-gray-200 px-8 py-12 text-center">
            <XCircle className="h-20 w-20 text-red-500 mx-auto mb-6" />
            <h2 className="text-2xl font-bold text-gray-900 mb-3">
              {t('invalidToken')}
            </h2>
            <p className="text-gray-600 mb-8">
              {t('invalidTokenDescription')}
            </p>
            <Link
              href="/forgot-password"
              className="inline-flex items-center justify-center w-full px-6 py-3 border border-transparent rounded-lg shadow-sm text-base font-medium text-white bg-blue-600 hover:bg-blue-700"
            >
              {t('requestNewToken')}
            </Link>
          </div>
        </div>
      </div>
    )
  }

  if (success) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <div className="bg-white rounded-xl shadow-lg border border-gray-200 px-8 py-12 text-center">
            <div className="mx-auto flex items-center justify-center h-20 w-20 rounded-full bg-green-100 mb-6">
              <CheckCircle className="h-10 w-10 text-green-600" />
            </div>
            <h2 className="text-2xl font-bold text-gray-900 mb-3">
              {t('passwordResetSuccess')}
            </h2>
            <p className="text-gray-600 mb-4">
              {t('passwordResetSuccessDescription')}
            </p>
            <div className="flex items-center justify-center text-blue-600 mb-8">
              <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600 mr-2"></div>
              <span className="text-sm font-medium">{t('redirectingToLogin')}</span>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <div className="mx-auto h-16 w-16 bg-blue-600 rounded-2xl flex items-center justify-center mb-4">
            <Shield className="h-8 w-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          <h2 className="text-2xl font-semibold text-gray-900">
            {t('resetPassword')}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('resetPasswordDescription')}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-white rounded-xl shadow-lg border border-gray-200 px-8 py-8">
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
                  className="pl-10 pr-10 h-12 border-gray-300 focus:border-blue-500 focus:ring-blue-500"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                >
                  {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>

              {/* Password Strength Indicator */}
              {password.length > 0 && (
                <div className="mt-3 space-y-2">
                  <div className="flex items-center justify-between text-xs mb-1">
                    <span className="text-gray-600">密码强度</span>
                    <span className={`font-medium ${
                      passwordStrength.score <= 1 ? 'text-red-600' :
                      passwordStrength.score === 2 ? 'text-yellow-600' :
                      passwordStrength.score === 3 ? 'text-green-600' :
                      'text-green-700'
                    }`}>
                      {passwordStrength.label}
                    </span>
                  </div>
                  <div className="w-full h-2 bg-gray-200 rounded-full overflow-hidden">
                    <div
                      className={`h-full ${passwordStrength.color} transition-all duration-300`}
                      style={{ width: `${(passwordStrength.score / 4) * 100}%` }}
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-xs">
                    <div className={`flex items-center ${passwordStrength.checks[0] ? 'text-green-600' : 'text-gray-400'}`}>
                      {passwordStrength.checks[0] ? <CheckCircle className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                      至少8位字符
                    </div>
                    <div className={`flex items-center ${passwordStrength.checks[1] ? 'text-green-600' : 'text-gray-400'}`}>
                      {passwordStrength.checks[1] ? <CheckCircle className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                      包含小写字母
                    </div>
                    <div className={`flex items-center ${passwordStrength.checks[2] ? 'text-green-600' : 'text-gray-400'}`}>
                      {passwordStrength.checks[2] ? <CheckCircle className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                      包含数字
                    </div>
                    <div className={`flex items-center ${passwordStrength.checks[3] ? 'text-green-600' : 'text-gray-400'}`}>
                      {passwordStrength.checks[3] ? <CheckCircle className="h-3 w-3 mr-1" /> : <XCircle className="h-3 w-3 mr-1" />}
                      包含特殊字符
                    </div>
                  </div>
                </div>
              )}
            </div>

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
                  className={`pl-10 pr-10 h-12 border-gray-300 focus:ring-blue-500 ${
                    confirmPassword.length > 0 && passwordsMatch
                      ? 'border-green-500 focus:border-green-500'
                      : confirmPassword.length > 0 && !passwordsMatch
                      ? 'border-red-500 focus:border-red-500'
                      : ''
                  }`}
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
                >
                  {showConfirmPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                </button>
              </div>

              {/* Password Match Indicator */}
              {confirmPassword.length > 0 && (
                <div className={`mt-2 flex items-center text-xs ${
                  passwordsMatch ? 'text-green-600' : 'text-red-600'
                }`}>
                  {passwordsMatch ? (
                    <>
                      <CheckCircle className="h-4 w-4 mr-1" />
                      密码匹配
                    </>
                  ) : (
                    <>
                      <XCircle className="h-4 w-4 mr-1" />
                      密码不匹配
                    </>
                  )}
                </div>
              )}
            </div>

            {/* Error Message */}
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm flex items-start">
                <AlertCircle className="h-5 w-5 mr-2 flex-shrink-0 mt-0.5" />
                <span>{error}</span>
              </div>
            )}

            {/* Submit Button */}
            <Button
              type="submit"
              className="w-full h-12 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
              disabled={loading || passwordStrength.score < 2 || !passwordsMatch}
            >
              {loading ? (
                <span className="flex items-center justify-center">
                  <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white mr-2"></div>
                  {t('resetting')}
                </span>
              ) : (
                t('resetPassword')
              )}
            </Button>

            {/* Back to Login Link */}
            <div className="text-center text-sm">
              <Link
                href="/login"
                className="text-blue-600 hover:text-blue-700 font-medium"
              >
                {t('backToLogin')}
              </Link>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
