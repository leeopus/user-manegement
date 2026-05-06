"use client"

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { useRouter, usePathname } from '@/i18n/routing'
import { api } from './api'
import { isAuthError, isNetworkError, isServerError } from './errors'
import type { User } from './types'

interface AuthContextType {
  user: User | null
  loading: boolean
  isAuthenticated: boolean
  login: (email: string, password: string, rememberMe?: boolean) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

const PUBLIC_PATHS = ['/login', '/register', '/forgot-password', '/reset-password']

// Retry config for transient network errors
const INIT_RETRY_COUNT = 2
const INIT_RETRY_DELAY_MS = 1000

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const router = useRouter()
  const pathname = usePathname()

  // 初始化：通过 httpOnly cookie 获取当前用户，区分网络错误和认证失败
  useEffect(() => {
    const initAuth = async (retriesLeft = INIT_RETRY_COUNT) => {
      let shouldKeepLoading = false
      try {
        const currentUser = await api.getUserInfo()
        setUser(currentUser)
      } catch (error) {
        if (isNetworkError(error) && retriesLeft > 0) {
          // 网络错误：延迟重试，不登出用户，保持 loading 状态
          shouldKeepLoading = true
          setTimeout(() => initAuth(retriesLeft - 1), INIT_RETRY_DELAY_MS)
          return
        }
        if (isServerError(error)) {
          // 服务端 5xx 错误：临时故障，不登出用户，保持 loading 以显示错误状态
          shouldKeepLoading = true
          setTimeout(() => initAuth(retriesLeft - 1), INIT_RETRY_DELAY_MS * 3)
          return
        }
        if (isAuthError(error)) {
          // 认证失败（401/403）：确实没有有效 session
          setUser(null)
        } else {
          // 其他不可恢复的错误：清除用户状态
          setUser(null)
        }
      } finally {
        if (!shouldKeepLoading) {
          setLoading(false)
        }
      }
    }

    initAuth()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 路由守卫
  useEffect(() => {
    if (loading) return

    const isPublicPath = PUBLIC_PATHS.some(
      (p) => pathname === p || pathname.startsWith(p + '/')
    )

    if (!user && !isPublicPath) {
      router.push('/login')
    }
  }, [user, loading, pathname, router])

  const login = useCallback(async (email: string, password: string, rememberMe = false) => {
    const data = await api.login({ email, password, remember_me: rememberMe })
    setUser(data.user)
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } catch {
      // 即使 logout API 失败也要清除本地状态
    }
    setUser(null)
    router.push('/login')
  }, [router])

  const refreshUser = useCallback(async () => {
    try {
      const currentUser = await api.getUserInfo()
      setUser(currentUser)
    } catch (error) {
      if (isNetworkError(error) || isServerError(error)) {
        // 网络波动或服务端临时故障，保持现有用户状态
        return
      }
      // 认证错误（401/403）等不可恢复错误，清除用户状态
      setUser(null)
    }
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: !!user,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export function hasPermission(user: User | null, permissionCode: string): boolean {
  if (!user?.roles) return false
  return user.roles.some(role =>
    role.permissions?.some(p => p.code === permissionCode)
  )
}
