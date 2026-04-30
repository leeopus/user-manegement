"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"

export default function ClearDataPage() {
  const router = useRouter()

  useEffect(() => {
    // Clear all localStorage data
    localStorage.removeItem("access_token")
    localStorage.removeItem("refresh_token")
    localStorage.removeItem("user")

    // Show message
    alert("已清除所有登录数据！即将跳转到登录页...")

    // Redirect to login
    setTimeout(() => {
      router.push("/login")
    }, 1000)
  }, [router])

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white p-8 rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-4">清除数据</h1>
        <p className="text-gray-600">正在清除所有登录数据...</p>
        <p className="text-sm text-gray-500 mt-4">完成后将自动跳转到登录页</p>
      </div>
    </div>
  )
}
