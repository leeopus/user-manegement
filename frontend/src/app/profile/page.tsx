"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"

interface User {
  ID: number
  Username: string
  Email: string
  Avatar?: string
  Status?: string
  CreatedAt: string
}

export default function ProfilePage() {
  const router = useRouter()
  const [user, setUser] = useState<User | null>(null)

  useEffect(() => {
    const userStr = localStorage.getItem("user")
    if (!userStr || userStr === "undefined") {
      router.push("/login")
      return
    }
    try {
      const user = JSON.parse(userStr)
      if (!user || !user.ID) {
        router.push("/login")
        return
      }
      setUser(user)
    } catch (err) {
      console.error("Failed to parse user data:", err)
      router.push("/login")
    }
  }, [router])

  const handleLogout = () => {
    localStorage.removeItem("access_token")
    localStorage.removeItem("refresh_token")
    localStorage.removeItem("user")
    router.push("/login")
  }

  if (!user) {
    return <div className="min-h-screen flex items-center justify-center">Loading...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">用户中心</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              {user.Username || user.Email}
            </span>
            <Button variant="outline" size="sm" onClick={handleLogout}>
              退出登录
            </Button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        <div className="max-w-2xl mx-auto">
          <Card>
            <CardHeader>
              <CardTitle>个人信息</CardTitle>
              <CardDescription>
                您的账户信息和可访问的应用
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-gray-600">用户名</label>
                  <p className="text-lg">{user.Username}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">邮箱</label>
                  <p className="text-lg">{user.Email}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">状态</label>
                  <div>
                    <Badge variant={user.Status === "active" ? "default" : "secondary"}>
                      {user.Status === "active" ? "正常" : "未激活"}
                    </Badge>
                  </div>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-600">注册时间</label>
                  <p className="text-lg">{new Date(user.CreatedAt).toLocaleDateString()}</p>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-medium mb-3">可访问的应用</h3>
                <div className="bg-gray-50 rounded-lg p-6 text-center text-gray-500">
                  <p>暂无可访问的应用</p>
                  <p className="text-sm mt-2">请联系管理员添加应用访问权限</p>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-medium mb-3">快速链接</h3>
                <div className="space-y-2">
                  <Button variant="outline" className="w-full justify-start" onClick={() => router.push("/dashboard")}>
                    🛠️ 管理后台
                  </Button>
                  <p className="text-xs text-gray-500">仅管理员可访问</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  )
}
