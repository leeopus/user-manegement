"use client"

import { useState } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Link } from "@/i18n/routing"
import { api } from "@/lib/api"
import { useErrorHandler } from "@/lib/use-error-handler"
import { Mail, ArrowLeft } from "lucide-react"

export default function ForgotPasswordPage() {
  const t = useTranslations('auth')
  const tc = useTranslations('common')
  const { getError } = useErrorHandler()
  const [email, setEmail] = useState("")
  const [loading, setLoading] = useState(false)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")

    try {
      await api.requestPasswordReset(email)
      setSuccess(true)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md">
        {/* Logo and Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">{tc('appName')}</h1>
          <h2 className="mt-6 text-2xl font-semibold text-gray-900">
            {t('forgotPassword')}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('resetPasswordDescription')}
          </p>
        </div>

        {/* Form Card */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 px-8 py-8">
          {!success ? (
            <form onSubmit={handleSubmit} className="space-y-6">
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

              {/* Info Message */}
              <div className="bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded-lg text-sm">
                {t('resetPasswordInfo')}
              </div>

              {/* Error Message */}
              {error && (
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
                  {error}
                </div>
              )}

              {/* Submit Button */}
              <Button
                type="submit"
                className="w-full h-11 bg-blue-600 hover:bg-blue-700 text-white font-medium text-sm transition-colors"
                disabled={loading}
              >
                {loading ? t('sending') : t('sendResetLink')}
              </Button>

              {/* Back to Login Link */}
              <div className="text-center text-sm">
                <Link
                  href="/login"
                  className="inline-flex items-center font-medium text-blue-600 hover:text-blue-700"
                >
                  <ArrowLeft className="h-4 w-4 mr-1" />
                  {t('backToLogin')}
                </Link>
              </div>
            </form>
          ) : (
            <div className="text-center space-y-4">
              <div className="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-green-100 mb-4">
                <svg className="h-8 w-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-gray-900">
                {t('resetLinkSent')}
              </h3>
              <p className="text-sm text-gray-600">
                {t('resetLinkSentDescription')}
              </p>
              <div className="pt-4">
                <Link
                  href="/login"
                  className="inline-flex items-center font-medium text-blue-600 hover:text-blue-700 text-sm"
                >
                  <ArrowLeft className="h-4 w-4 mr-1" />
                  {t('backToLogin')}
                </Link>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="mt-8 text-center text-xs text-gray-500">
          {tc('copyright')}
        </p>
      </div>
    </div>
  )
}
