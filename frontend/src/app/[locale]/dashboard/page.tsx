"use client"

import { useEffect, useState } from 'react'
import { useTranslations } from 'next-intl'
import { api } from '@/lib/api'

export default function DashboardPage() {
  const t = useTranslations('dashboard')
  const [stats, setStats] = useState({ users: 0, roles: 0, applications: 0 })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetchStats() {
      try {
        const [usersData, rolesData, appsData] = await Promise.allSettled([
          api.listUsers(1, 1),
          api.listRoles(1, 1),
          api.listApplications(1, 1),
        ])
        setStats({
          users: usersData.status === 'fulfilled' ? usersData.value.total : 0,
          roles: rolesData.status === 'fulfilled' ? rolesData.value.total : 0,
          applications: appsData.status === 'fulfilled' ? appsData.value.total : 0,
        })
      } catch {
        // keep default zeros
      } finally {
        setLoading(false)
      }
    }
    fetchStats()
  }, [])

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
