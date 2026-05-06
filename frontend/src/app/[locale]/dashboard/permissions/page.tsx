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
import { AlertCircle } from "lucide-react"
import type { Permission } from "@/lib/types"

export default function PermissionsPage() {
  const t = useTranslations('permissions')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()

  const [permissions, setPermissions] = useState<Permission[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const pageSize = 10

  const [createOpen, setCreateOpen] = useState(false)
  const [editPerm, setEditPerm] = useState<Permission | null>(null)
  const [deletePerm, setDeletePerm] = useState<Permission | null>(null)

  const [formName, setFormName] = useState("")
  const [formCode, setFormCode] = useState("")
  const [formResource, setFormResource] = useState("")
  const [formAction, setFormAction] = useState("")
  const [formDescription, setFormDescription] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const canManage = hasPermission(user, "permissions:manage")

  const fetchPermissions = useCallback(async (p: number) => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listPermissions(p, pageSize)
      setPermissions(data.permissions || [])
      setTotal(data.total)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }, [getError])

  useEffect(() => {
    if (!authLoading && user && canManage) {
      fetchPermissions(page)
    }
  }, [authLoading, user, page, fetchPermissions, canManage])

  const totalPages = Math.ceil(total / pageSize)

  const handleCreate = async () => {
    setFormError(null)
    if (!formName.trim() || !formCode.trim() || !formResource.trim() || !formAction.trim()) {
      setFormError(t('validationAllRequired'))
      return
    }
    setSubmitting(true)
    try {
      await api.createPermission({
        name: formName.trim(), code: formCode.trim(),
        resource: formResource.trim(), action: formAction.trim(),
        description: formDescription.trim(),
      })
      setCreateOpen(false)
      resetForm()
      fetchPermissions(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (perm: Permission) => {
    setFormName(perm.name)
    setFormCode(perm.code)
    setFormResource(perm.resource)
    setFormAction(perm.action)
    setFormDescription(perm.description || "")
    setFormError(null)
    setEditPerm(perm)
  }

  const handleEdit = async () => {
    if (!editPerm) return
    setFormError(null)
    if (!formName.trim() || !formCode.trim() || !formResource.trim() || !formAction.trim()) {
      setFormError(t('validationAllRequired'))
      return
    }
    setSubmitting(true)
    try {
      await api.updatePermission(editPerm.id, {
        name: formName.trim(), code: formCode.trim(),
        resource: formResource.trim(), action: formAction.trim(),
        description: formDescription.trim(),
      })
      setEditPerm(null)
      resetForm()
      fetchPermissions(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deletePerm) return
    setSubmitting(true)
    try {
      await api.deletePermission(deletePerm.id)
      setDeletePerm(null)
      fetchPermissions(page)
    } catch (err) {
      setError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const resetForm = () => {
    setFormName("")
    setFormCode("")
    setFormResource("")
    setFormAction("")
    setFormDescription("")
    setFormError(null)
  }

  if (authLoading) return <div className="flex items-center justify-center py-8">{t('loading')}</div>
  if (!user || !canManage) return null

  const permFormFields = (
    <div className="space-y-4">
      <div>
        <Label>{t('name')}</Label>
        <Input placeholder={t('namePlaceholder')} value={formName} onChange={(e) => setFormName(e.target.value)} className="mt-1" />
      </div>
      <div>
        <Label>{t('code')}</Label>
        <Input placeholder={t('codePlaceholder')} value={formCode} onChange={(e) => setFormCode(e.target.value)} className="mt-1" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>{t('resource')}</Label>
          <Input placeholder={t('resourcePlaceholder')} value={formResource} onChange={(e) => setFormResource(e.target.value)} className="mt-1" />
        </div>
        <div>
          <Label>{t('action')}</Label>
          <Input placeholder={t('actionPlaceholder')} value={formAction} onChange={(e) => setFormAction(e.target.value)} className="mt-1" />
        </div>
      </div>
      <div>
        <Label>{t('description')}</Label>
        <Input placeholder={t('descriptionPlaceholder')} value={formDescription} onChange={(e) => setFormDescription(e.target.value)} className="mt-1" />
      </div>
      {formError && <p className="text-sm text-red-600">{formError}</p>}
    </div>
  )

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">{t('title')}</h1>
        <Button onClick={() => { resetForm(); setCreateOpen(true) }}>{t('addPermission')}</Button>
      </div>

      <Card>
        <CardHeader><CardTitle>{t('permissionsList')}</CardTitle></CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-3" />
              <p className="text-red-600 mb-4">{error}</p>
              <Button variant="outline" onClick={() => fetchPermissions(page)}>{t('refresh')}</Button>
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
                    <TableHead>{t('code')}</TableHead>
                    <TableHead>{t('resource')}</TableHead>
                    <TableHead>{t('action')}</TableHead>
                    <TableHead>{t('description')}</TableHead>
                    <TableHead>{t('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {permissions.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-8">{t('noPermissions')}</TableCell>
                    </TableRow>
                  ) : permissions.map((perm) => (
                    <TableRow key={perm.id}>
                      <TableCell>{perm.id}</TableCell>
                      <TableCell className="font-medium">{perm.name}</TableCell>
                      <TableCell><code className="text-sm bg-gray-100 px-1.5 py-0.5 rounded">{perm.code}</code></TableCell>
                      <TableCell>{perm.resource}</TableCell>
                      <TableCell>{perm.action}</TableCell>
                      <TableCell className="text-gray-500">{perm.description || "—"}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button variant="ghost" size="sm" onClick={() => openEdit(perm)}>{t('edit')}</Button>
                          <Button variant="ghost" size="sm" className="text-red-600 hover:text-red-700" onClick={() => setDeletePerm(perm)}>{t('delete')}</Button>
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

      {/* Create Permission Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('createPermission')}</DialogTitle></DialogHeader>
          {permFormFields}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleCreate} disabled={submitting}>{t('createPermission')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Permission Dialog */}
      <Dialog open={!!editPerm} onOpenChange={(open) => { if (!open) setEditPerm(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('editPermission')}</DialogTitle></DialogHeader>
          {permFormFields}
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleEdit} disabled={submitting}>{t('save')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={!!deletePerm} onOpenChange={(open) => { if (!open) setDeletePerm(null) }}>
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
