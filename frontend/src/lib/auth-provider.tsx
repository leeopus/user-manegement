"use client"

import React, { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { useRouter, usePathname } from '@/i18n/routing'
import { api } from './api'
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

// 不需要认证的页面路径
const PUBLIC_PATHS = ['/login', '/register', '/forgot-password', '/reset-password']

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const router = useRouter()
  const pathname = usePathname()

  // 从 localStorage 恢复用户状态
  useEffect(() => {
    const initAuth = async () => {
      try {
        // 尝试通过 cookie (httpOnly) 获取当前用户
        // 后端会从 cookie 中读取 access_token / refresh_token
        const currentUser = await api.getUserInfo('')
        setUser(currentUser)
        localStorage.setItem('user', JSON.stringify(currentUser))
      } catch {
        // cookie 无效或过期
        localStorage.removeItem('user')
        setUser(null)
      } finally {
        setLoading(false)
      }
    }

    initAuth()
  }, [])

  // 自动 token 刷新：每 12 分钟尝试刷新（access token 15 分钟过期）
  useEffect(() => {
    if (!user) return

    const interval = setInterval(async () => {
      try {
        await api.refreshToken('')
      } catch {
        // 刷新失败，清除状态并跳转登录
        setUser(null)
        localStorage.removeItem('user')
        router.push('/login')
      }
    }, 12 * 60 * 1000)

    return () => clearInterval(interval)
  }, [user, router])

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
    localStorage.setItem('user', JSON.stringify(data.user))

    if (rememberMe) {
      localStorage.setItem('rememberMe', 'true')
    } else {
      localStorage.removeItem('rememberMe')
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout('')
    } catch {
      // 即使 logout API 失败也要清除本地状态
    }
    setUser(null)
    localStorage.removeItem('user')
    router.push('/login')
  }, [router])

  const refreshUser = useCallback(async () => {
    try {
      const currentUser = await api.getUserInfo('')
      setUser(currentUser)
      localStorage.setItem('user', JSON.stringify(currentUser))
    } catch {
      setUser(null)
      localStorage.removeItem('user')
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
