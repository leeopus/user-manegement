"use client"

import { useEffect, useState, useCallback, useRef } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api"
import { useAuth, hasPermission } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { AlertCircle, Search } from "lucide-react"
import type { AuditLog } from "@/lib/types"

export default function AuditLogsPage() {
  const t = useTranslations('auditLogs')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()

  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionFilter, setActionFilter] = useState<string>("")
  const [resourceFilter, setResourceFilter] = useState<string>("")
  const [searchQuery, setSearchQuery] = useState("")
  const pageSize = 10

  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const canRead = hasPermission(user, "audit:read")

  const fetchLogs = useCallback(async (p: number, action: string, resource: string, search: string) => {
    setLoading(true)
    setError(null)
    try {
      const filters: { action?: string; resource?: string; search?: string } = {}
      if (action) filters.action = action
      if (resource) filters.resource = resource
      if (search.trim()) filters.search = search.trim()
      const data = await api.listAuditLogs(p, pageSize, filters)
      setLogs(data.logs || [])
      setTotal(data.total)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }, [getError])

  useEffect(() => {
    if (!authLoading && user && canRead) {
      fetchLogs(page, actionFilter, resourceFilter, searchQuery)
    }
  }, [authLoading, user, page, actionFilter, fetchLogs, canRead])

  const handleSearchChange = (value: string) => {
    setSearchQuery(value)
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current)
    searchTimerRef.current = setTimeout(() => {
      setPage(1)
      fetchLogs(1, actionFilter, resourceFilter, value)
    }, 300)
  }

  const handleActionFilterChange = (value: string | null) => {
    const action = value ?? ""
    setActionFilter(action)
    setPage(1)
    fetchLogs(1, action, resourceFilter, searchQuery)
  }

  const handleResourceFilterChange = (value: string | null) => {
    const resource = value ?? ""
    setResourceFilter(resource)
    setPage(1)
    fetchLogs(1, actionFilter, resource, searchQuery)
  }

  const totalPages = Math.ceil(total / pageSize)

  // Extract unique action/resource values from current data for filter options
  const actionOptions = [...new Set(logs.map(l => l.action).filter(Boolean))]
  const resourceOptions = [...new Set(logs.map(l => l.resource).filter(Boolean))]

  if (authLoading) return <div className="flex items-center justify-center py-8">{t('loading')}</div>
  if (!user || !canRead) return null

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">{t('title')}</h1>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <CardTitle>{t('logsList')}</CardTitle>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
                <Input
                  placeholder={t('searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  className="pl-9 w-52"
                />
              </div>
              <Select value={actionFilter || "__all__"} onValueChange={(v) => handleActionFilterChange(v === "__all__" ? "" : v)}>
                <SelectTrigger className="w-36">
                  <SelectValue placeholder={t('allActions')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">{t('allActions')}</SelectItem>
                  {actionOptions.map(a => <SelectItem key={a} value={a}>{a}</SelectItem>)}
                </SelectContent>
              </Select>
              <Select value={resourceFilter || "__all__"} onValueChange={(v) => handleResourceFilterChange(v === "__all__" ? "" : v)}>
                <SelectTrigger className="w-36">
                  <SelectValue placeholder={t('allResources')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">{t('allResources')}</SelectItem>
                  {resourceOptions.map(r => <SelectItem key={r} value={r}>{r}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-3" />
              <p className="text-red-600 mb-4">{error}</p>
              <Button variant="outline" onClick={() => fetchLogs(page, actionFilter, resourceFilter, searchQuery)}>{t('refresh')}</Button>
            </div>
          ) : loading ? (
            <div className="text-center py-8">{t('loading')}</div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('id')}</TableHead>
                    <TableHead>{t('user')}</TableHead>
                    <TableHead>{t('action')}</TableHead>
                    <TableHead>{t('resource')}</TableHead>
                    <TableHead>{t('resourceId')}</TableHead>
                    <TableHead>{t('ipAddress')}</TableHead>
                    <TableHead>{t('createdAt')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {logs.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-8">{t('noLogs')}</TableCell>
                    </TableRow>
                  ) : logs.map((log) => (
                    <TableRow key={log.id}>
                      <TableCell>{log.id}</TableCell>
                      <TableCell className="font-medium">{log.username || `User #${log.user_id}`}</TableCell>
                      <TableCell><Badge variant="secondary" className="text-xs">{log.action}</Badge></TableCell>
                      <TableCell>{log.resource}</TableCell>
                      <TableCell>{log.resource_id || "—"}</TableCell>
                      <TableCell className="text-gray-500 font-mono text-xs">{log.ip_address}</TableCell>
                      <TableCell>{new Date(log.created_at).toLocaleString()}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4 pt-4 border-t">
                  <p className="text-sm text-gray-500">{t('total', { count: total })}</p>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>{t('prev')}</Button>
                    <span className="text-sm text-gray-600">{page} / {totalPages}</span>
                    <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>{t('next')}</Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
