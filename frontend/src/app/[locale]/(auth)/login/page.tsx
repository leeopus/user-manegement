"use client"

import { useState, useRef, useCallback, useMemo } from "react"
import { useRouter } from "@/i18n/routing"
import { useSearchParams } from "next/navigation"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useAuth } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { Link } from "@/i18n/routing"
import { Mail, Lock, Eye, EyeOff } from "lucide-react"
import { OpusBrandLogo } from "@/components/opus-logo"

const MAX_LOGIN_ATTEMPTS = 5
const BASE_DELAY_MS = 1000

export default function LoginPage() {
  const t = useTranslations('auth')
  const tc = useTranslations('common')
  const { getError } = useErrorHandler()
  const { login } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()
  const redirect = useMemo(() => searchParams.get('redirect') || '', [searchParams])
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [rememberMe, setRememberMe] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [retryAfter, setRetryAfter] = useState(0)

  const failCountRef = useRef(0)
  const cooldownRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const getBackoffMs = useCallback((failCount: number) => {
    return Math.min(BASE_DELAY_MS * Math.pow(2, failCount - 1), 30000)
  }, [])

  const startCooldown = useCallback((ms: number) => {
    setRetryAfter(Math.ceil(ms / 1000))
    const interval = setInterval(() => {
      setRetryAfter(prev => {
        if (prev <= 1) {
          clearInterval(interval)
          return 0
        }
        return prev - 1
      })
    }, 1000)
    cooldownRef.current = setTimeout(() => {
      cooldownRef.current = null
    }, ms)
  }, [])

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()

    if (cooldownRef.current) return

    setLoading(true)
    setError("")

    try {
      await login(email, password, rememberMe)
      failCountRef.current = 0
      if (redirect) {
        if (redirect.startsWith('http') || redirect.startsWith('/api')) {
          window.location.href = redirect
        } else {
          router.push(redirect)
        }
      } else {
        router.push("/profile")
      }
    } catch (err) {
      failCountRef.current += 1
      setError(getError(err))
      if (failCountRef.current >= MAX_LOGIN_ATTEMPTS) {
        const backoff = getBackoffMs(failCountRef.current)
        setError(t('tooManyAttempts'))
        startCooldown(backoff)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center mb-4">
            <OpusBrandLogo size="lg" system="account" />
          </div>
          <h2 className="text-xl font-semibold text-gray-900">
            {t('welcomeBack')}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('loginDescription')}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8">
          <form onSubmit={handleLogin} className="space-y-6">
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
                  className="pl-10 h-11 border-gray-300 focus:border-blue-500 focus:ring-blue-500"
                  required
                />
              </div>
            </div>

            {/* Password */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <Label htmlFor="password" className="block text-sm font-medium text-gray-700">
                  {t('password')}
                </Label>
                <Link
                  href="/forgot-password"
                  className="text-sm font-medium text-blue-600 hover:text-blue-700"
                >
                  {t('forgotPassword')}
                </Link>
              </div>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-5 w-5" />
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder={t('passwordPlaceholder')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pl-10 pr-10 h-11 border-gray-300 focus:border-blue-500 focus:ring-blue-500"
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
            </div>

            {/* Remember Me */}
            <div className="flex items-center">
              <input
                id="remember"
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <label htmlFor="remember" className="ml-2 block text-sm text-gray-700">
                {t('rememberMe')}
              </label>
            </div>

            {/* Error Message */}
            {error && (
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
                {error}
              </div>
            )}

            {/* Sign In Button */}
            <Button
              type="submit"
              className="w-full h-11 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
              disabled={loading || retryAfter > 0}
            >
              {loading ? t('signingIn') : retryAfter > 0 ? `${t('signIn')} (${retryAfter}s)` : t('signIn')}
            </Button>

            {/* Sign Up Link */}
            <div className="text-center text-sm">
              <span className="text-gray-600">{t('noAccount')}</span>{" "}
              <Link
                href={redirect ? `/register?redirect=${encodeURIComponent(redirect)}` : "/register"}
                className="font-medium text-blue-600 hover:text-blue-700"
              >
                {t('register')}
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
