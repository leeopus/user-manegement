"use client"

import { useEffect, useState } from 'react'
import { useTranslations } from 'next-intl'
import { api } from '@/lib/api'
import { useAuth, hasPermission } from '@/lib/auth-provider'

export default function DashboardPage() {
  const t = useTranslations('dashboard')
  const { user, loading: authLoading } = useAuth()
  const [stats, setStats] = useState({ users: 0, roles: 0, applications: 0 })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (authLoading || !user) return

    async function fetchStats() {
      try {
        const promises: Promise<{ total: number } | null>[] = [
          hasPermission(user, "users:read") ? api.listUsers(1, 1).catch(() => null) : Promise.resolve(null),
          hasPermission(user, "roles:manage") ? api.listRoles(1, 1).catch(() => null) : Promise.resolve(null),
          hasPermission(user, "oauth:manage") ? api.listApplications(1, 1).catch(() => null) : Promise.resolve(null),
        ]
        const [usersData, rolesData, appsData] = await Promise.all(promises)
        setStats({
          users: (usersData as { total: number } | null)?.total ?? 0,
          roles: (rolesData as { total: number } | null)?.total ?? 0,
          applications: (appsData as { total: number } | null)?.total ?? 0,
        })
      } catch {
        // keep default zeros
      } finally {
        setLoading(false)
      }
    }
    fetchStats()
  }, [authLoading, user])

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">{t('overview')}</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="text-lg font-semibold mb-2">{t('totalUsers')}</h3>
          <p className="text-3xl font-bold text-blue-600">
            {loading ? '...' : stats.users}
          </p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="text-lg font-semibold mb-2">{t('totalRoles')}</h3>
          <p className="text-3xl font-bold text-green-600">
            {loading ? '...' : stats.roles}
          </p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <h3 className="text-lg font-semibold mb-2">{t('ssoApps')}</h3>
          <p className="text-3xl font-bold text-purple-600">
            {loading ? '...' : stats.applications}
          </p>
        </div>
      </div>
    </div>
  )
}
