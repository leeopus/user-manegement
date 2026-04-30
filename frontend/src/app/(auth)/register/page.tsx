"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { PasswordStrength } from "@/components/ui/password-strength"
import { api } from "@/lib/api"
import {
  validateUsername,
  validateEmail,
  validatePassword,
  PasswordStrength as StrengthLevel,
  debounce,
} from "@/lib/validation"

interface ValidationErrors {
  username?: string
  email?: string
  password?: string
  confirmPassword?: string
}

export default function RegisterPage() {
  const router = useRouter()
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [validationErrors, setValidationErrors] = useState<ValidationErrors>({})
  const [passwordStrength, setPasswordStrength] = useState<{ strength: StrengthLevel; score: number }>({
    strength: StrengthLevel.Weak,
    score: 0,
  })

  // 实时验证用户名
  useEffect(() => {
    if (username) {
      const result = validateUsername(username)
      setValidationErrors(prev => ({
        ...prev,
        username: result.valid ? undefined : result.error,
      }))
    } else {
      setValidationErrors(prev => ({ ...prev, username: undefined }))
    }
  }, [username])

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
      const result = validatePassword(password, username)
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
  }, [password, username])

  // 验证确认密码
  useEffect(() => {
    if (confirmPassword) {
      setValidationErrors(prev => ({
        ...prev,
        confirmPassword: password !== confirmPassword ? "两次输入的密码不一致" : undefined,
      }))
    } else {
      setValidationErrors(prev => ({ ...prev, confirmPassword: undefined }))
    }
  }, [confirmPassword, password])

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    // 最终验证
    const usernameResult = validateUsername(username)
    const emailResult = validateEmail(email)
    const passwordResult = validatePassword(password, username)

    const errors: ValidationErrors = {}
    if (!usernameResult.valid) errors.username = usernameResult.error
    if (!emailResult.valid) errors.email = emailResult.error
    if (passwordResult.error) errors.password = passwordResult.error
    if (password !== confirmPassword) errors.confirmPassword = "两次输入的密码不一致"

    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors)
      return
    }

    setLoading(true)

    try {
      const response = await api.register({ username, email, password })

      if (response.code === 0) {
        // 注册成功，跳转到登录页
        router.push("/login?registered=true")
      } else {
        setError(response.message || "注册失败")
      }
    } catch (err) {
      console.error("Registration error:", err)
      setError("网络错误，请检查您的连接")
    } finally {
      setLoading(false)
    }
  }

  const isFormValid =
    username &&
    email &&
    password &&
    confirmPassword &&
    Object.values(validationErrors).every(err => !err) &&
    passwordStrength.strength >= StrengthLevel.Fair

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl">创建账号</CardTitle>
          <CardDescription>
            填写信息以创建新账号
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleRegister} className="space-y-4">
            {/* 用户名 */}
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                type="text"
                placeholder="输入用户名"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className={validationErrors.username ? "border-red-500" : ""}
                required
              />
              {validationErrors.username && (
                <p className="text-xs text-red-600">{validationErrors.username}</p>
              )}
            </div>

            {/* 邮箱 */}
            <div className="space-y-2">
              <Label htmlFor="email">邮箱</Label>
              <Input
                id="email"
                type="email"
                placeholder="name@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={validationErrors.email ? "border-red-500" : ""}
                required
              />
              {validationErrors.email && (
                <p className="text-xs text-red-600">{validationErrors.email}</p>
              )}
            </div>

            {/* 密码 */}
            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                placeholder="创建密码"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={validationErrors.password ? "border-red-500" : ""}
                required
              />
              {password && <PasswordStrength strength={passwordStrength.strength} score={passwordStrength.score} />}
              {validationErrors.password && (
                <p className="text-xs text-red-600">{validationErrors.password}</p>
              )}
            </div>

            {/* 确认密码 */}
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">确认密码</Label>
              <Input
                id="confirmPassword"
                type="password"
                placeholder="再次输入密码"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className={validationErrors.confirmPassword ? "border-red-500" : ""}
                required
              />
              {validationErrors.confirmPassword && (
                <p className="text-xs text-red-600">{validationErrors.confirmPassword}</p>
              )}
            </div>

            {/* 密码要求提示 */}
            <div className="text-xs text-gray-600 bg-gray-50 p-3 rounded">
              <p className="font-semibold mb-1">密码要求：</p>
              <ul className="list-disc list-inside space-y-1">
                <li>至少 8 位字符</li>
                <li>包含小写字母和数字</li>
                <li>不能包含用户名</li>
                <li>建议使用特殊字符增强强度</li>
              </ul>
            </div>

            {/* 全局错误 */}
            {error && (
              <div className="text-sm text-red-600 bg-red-50 p-3 rounded">
                {error}
              </div>
            )}

            <Button
              type="submit"
              className="w-full"
              disabled={loading || !isFormValid}
            >
              {loading ? "创建中..." : "创建账号"}
            </Button>
          </form>

          <div className="mt-4 text-center text-sm">
            已有账号？{" "}
            <a href="/login" className="text-blue-600 hover:underline">
              立即登录
            </a>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
