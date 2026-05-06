"use client"

import { useEffect, useState, useCallback } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose, DialogDescription,
} from "@/components/ui/dialog"
import { api } from "@/lib/api"
import { useAuth, hasPermission } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { AlertCircle, Copy, Check } from "lucide-react"
import type { OAuthApplication, OAuthApplicationCreateResult } from "@/lib/types"

export default function ApplicationsPage() {
  const t = useTranslations('applications')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()

  const [apps, setApps] = useState<OAuthApplication[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const pageSize = 10

  const [createOpen, setCreateOpen] = useState(false)
  const [editApp, setEditApp] = useState<OAuthApplication | null>(null)
  const [deleteApp, setDeleteApp] = useState<OAuthApplication | null>(null)
  const [newSecret, setNewSecret] = useState<string | null>(null)

  const [formName, setFormName] = useState("")
  const [formRedirectUris, setFormRedirectUris] = useState("")
  const [formScopes, setFormScopes] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [copiedId, setCopiedId] = useState<string | null>(null)

  const canManage = hasPermission(user, "oauth:manage")

  const fetchApps = useCallback(async (p: number) => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listApplications(p, pageSize)
      setApps(data.applications || [])
      setTotal(data.total)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }, [getError])

  useEffect(() => {
    if (!authLoading && user && canManage) {
      fetchApps(page)
    }
  }, [authLoading, user, page, fetchApps, canManage])

  const totalPages = Math.ceil(total / pageSize)

  const handleCreate = async () => {
    setFormError(null)
    if (!formName.trim() || !formRedirectUris.trim()) {
      setFormError(t('validationNameRedirectRequired'))
      return
    }
    setSubmitting(true)
    try {
      const result = await api.createApplication({
        name: formName.trim(),
        redirect_uris: formRedirectUris.trim(),
        scopes: formScopes.trim() || undefined,
      })
      setNewSecret(result.client_secret || null)
      setCreateOpen(false)
      resetForm()
      fetchApps(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (app: OAuthApplication) => {
    setFormName(app.name)
    setFormRedirectUris(app.redirect_uris)
    setFormError(null)
    setEditApp(app)
  }

  const handleEdit = async () => {
    if (!editApp) return
    setFormError(null)
    if (!formName.trim() || !formRedirectUris.trim()) {
      setFormError(t('validationNameRedirectRequired'))
      return
    }
    setSubmitting(true)
    try {
      await api.updateApplication(editApp.id, {
        name: formName.trim(),
        redirect_uris: formRedirectUris.trim(),
      })
      setEditApp(null)
      resetForm()
      fetchApps(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteApp) return
    setSubmitting(true)
    try {
      await api.deleteApplication(deleteApp.id)
      setDeleteApp(null)
      fetchApps(page)
    } catch (err) {
      setError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const copyToClipboard = async (text: string, id: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedId(id)
      setTimeout(() => setCopiedId(null), 2000)
    } catch {
      // ignore
    }
  }

  const resetForm = () => {
    setFormName("")
    setFormRedirectUris("")
    setFormScopes("")
    setFormError(null)
  }

  if (authLoading) return <div className="flex items-center justify-center py-8">{t('loading')}</div>
  if (!user || !canManage) return null

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">{t('title')}</h1>
        <Button onClick={() => { resetForm(); setNewSecret(null); setCreateOpen(true) }}>{t('addApp')}</Button>
      </div>

      {/* New secret banner */}
      {newSecret && (
        <div className="mb-4 p-4 bg-amber-50 border border-amber-200 rounded-lg">
          <p className="text-sm text-amber-800 mb-2">{t('secretWarning')}</p>
          <div className="flex items-center gap-2">
            <code className="text-sm bg-white px-2 py-1 rounded border flex-1 break-all">{newSecret}</code>
            <Button size="sm" variant="outline" onClick={() => copyToClipboard(newSecret, "secret")}>
              {copiedId === "secret" ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
          <Button size="sm" variant="ghost" className="mt-2" onClick={() => setNewSecret(null)}>Dismiss</Button>
        </div>
      )}

      <Card>
        <CardHeader><CardTitle>{t('appsList')}</CardTitle></CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-3" />
              <p className="text-red-600 mb-4">{error}</p>
              <Button variant="outline" onClick={() => fetchApps(page)}>{t('refresh')}</Button>
            </div>
          ) : loading ? (
            <div className="text-center py-8">{t('loading')}</div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t('name')}</TableHead>
                    <TableHead>{t('clientId')}</TableHead>
                    <TableHead>{t('redirectUris')}</TableHead>
                    <TableHead>{t('createdAt')}</TableHead>
                    <TableHead>{t('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {apps.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8">{t('noApps')}</TableCell>
                    </TableRow>
                  ) : apps.map((app) => (
                    <TableRow key={app.id}>
                      <TableCell>{app.id}</TableCell>
                      <TableCell className="font-medium">{app.name}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{app.client_id}</code>
                          <button onClick={() => copyToClipboard(app.client_id, String(app.id))} className="p-0.5 hover:text-gray-600">
                            {copiedId === String(app.id) ? <Check className="h-3 w-3 text-green-600" /> : <Copy className="h-3 w-3" />}
                          </button>
                        </div>
                      </TableCell>
                      <TableCell className="max-w-xs truncate">{app.redirect_uris}</TableCell>
                      <TableCell>{new Date(app.created_at).toLocaleDateString()}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button variant="ghost" size="sm" onClick={() => openEdit(app)}>{t('edit')}</Button>
                          <Button variant="ghost" size="sm" className="text-red-600 hover:text-red-700" onClick={() => setDeleteApp(app)}>{t('delete')}</Button>
                        </div>
                      </TableCell>
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

      {/* Create App Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('createApp')}</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('name')}</Label>
              <Input placeholder={t('namePlaceholder')} value={formName} onChange={(e) => setFormName(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('redirectUris')}</Label>
              <Input placeholder={t('redirectUrisPlaceholder')} value={formRedirectUris} onChange={(e) => setFormRedirectUris(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('scopes')}</Label>
              <Input placeholder={t('scopesPlaceholder')} value={formScopes} onChange={(e) => setFormScopes(e.target.value)} className="mt-1" />
            </div>
            {formError && <p className="text-sm text-red-600">{formError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleCreate} disabled={submitting}>{t('createApp')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit App Dialog */}
      <Dialog open={!!editApp} onOpenChange={(open) => { if (!open) setEditApp(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('editApp')}</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('name')}</Label>
              <Input value={formName} onChange={(e) => setFormName(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('redirectUris')}</Label>
              <Input value={formRedirectUris} onChange={(e) => setFormRedirectUris(e.target.value)} className="mt-1" />
            </div>
            {formError && <p className="text-sm text-red-600">{formError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleEdit} disabled={submitting}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={!!deleteApp} onOpenChange={(open) => { if (!open) setDeleteApp(null) }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader><DialogTitle>{t('delete')}</DialogTitle></DialogHeader>
          <DialogDescription>{t('confirmDelete')}</DialogDescription>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button variant="destructive" onClick={handleDelete} disabled={submitting}>{t('delete')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
