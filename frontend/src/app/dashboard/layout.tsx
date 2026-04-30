"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const router = useRouter()
  const [user, setUser] = useState<any>(null)

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
    return <div>Loading...</div>
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">User Management System</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              Welcome, {user.Username || user.Email}
            </span>
            <Button variant="outline" size="sm" onClick={handleLogout}>
              Logout
            </Button>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b border-gray-200">
        <div className="container mx-auto px-4">
          <div className="flex space-x-8">
            <a
              href="/dashboard"
              className="py-4 px-2 text-sm font-medium text-gray-900 border-b-2 border-gray-900"
            >
              Home
            </a>
            <a
              href="/dashboard/users"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Users
            </a>
            <a
              href="/dashboard/roles"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Roles
            </a>
            <a
              href="/dashboard/permissions"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              Permissions
            </a>
            <a
              href="/dashboard/applications"
              className="py-4 px-2 text-sm font-medium text-gray-600 hover:text-gray-900"
            >
              SSO Apps
            </a>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="container mx-auto px-4 py-8">
        {children}
      </main>
    </div>
  )
}
