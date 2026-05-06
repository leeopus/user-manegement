"use client"

import { useEffect, useState, useCallback } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose, DialogDescription,
} from "@/components/ui/dialog"
import { api } from "@/lib/api"
import { useAuth, hasPermission } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { AlertCircle, X } from "lucide-react"
import type { Role, Permission } from "@/lib/types"

export default function RolesPage() {
  const t = useTranslations('roles')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()

  const [roles, setRoles] = useState<Role[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const pageSize = 10

  const [createOpen, setCreateOpen] = useState(false)
  const [editRole, setEditRole] = useState<Role | null>(null)
  const [deleteRole, setDeleteRole] = useState<Role | null>(null)
  const [permDialogRole, setPermDialogRole] = useState<Role | null>(null)

  const [formName, setFormName] = useState("")
  const [formCode, setFormCode] = useState("")
  const [formDescription, setFormDescription] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [allPermissions, setAllPermissions] = useState<Permission[]>([])
  const [permLoading, setPermLoading] = useState(false)

  const canManage = hasPermission(user, "roles:manage")

  const fetchRoles = useCallback(async (p: number) => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listRoles(p, pageSize)
      setRoles(data.roles || [])
      setTotal(data.total)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }, [getError])

  useEffect(() => {
    if (!authLoading && user && canManage) {
      fetchRoles(page)
    }
  }, [authLoading, user, page, fetchRoles, canManage])

  const totalPages = Math.ceil(total / pageSize)

  const handleCreate = async () => {
    setFormError(null)
    if (!formName.trim() || !formCode.trim()) {
      setFormError("Name and code are required")
      return
    }
    setSubmitting(true)
    try {
      await api.createRole({ name: formName.trim(), code: formCode.trim().toLowerCase(), description: formDescription.trim() })
      setCreateOpen(false)
      resetForm()
      fetchRoles(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (role: Role) => {
    setFormName(role.name)
    setFormCode(role.code)
    setFormDescription(role.description || "")
    setFormError(null)
    setEditRole(role)
  }

  const handleEdit = async () => {
    if (!editRole) return
    setFormError(null)
    if (!formName.trim() || !formCode.trim()) {
      setFormError("Name and code are required")
      return
    }
    setSubmitting(true)
    try {
      await api.updateRole(editRole.id, { name: formName.trim(), code: formCode.trim().toLowerCase(), description: formDescription.trim() })
      setEditRole(null)
      resetForm()
      fetchRoles(page)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteRole) return
    setSubmitting(true)
    try {
      await api.deleteRole(deleteRole.id)
      setDeleteRole(null)
      fetchRoles(page)
    } catch (err) {
      setError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  const openPermDialog = async (role: Role) => {
    setPermDialogRole(role)
    setPermLoading(true)
    try {
      const data = await api.listPermissions(1, 100)
      setAllPermissions(data.permissions || [])
    } catch (err) {
      setError(getError(err))
    } finally {
      setPermLoading(false)
    }
  }

  const handleTogglePermission = async (permId: number, assigned: boolean) => {
    if (!permDialogRole) return
    setPermLoading(true)
    try {
      if (assigned) {
        await api.removePermission(permDialogRole.id, permId)
      } else {
        await api.assignPermission(permDialogRole.id, permId)
      }
      // Refresh role with updated permissions
      const updated = await api.getRole(permDialogRole.id)
      setPermDialogRole(updated)
      fetchRoles(page)
    } catch (err) {
      setError(getError(err))
    } finally {
      setPermLoading(false)
    }
  }

  const resetForm = () => {
    setFormName("")
    setFormCode("")
    setFormDescription("")
    setFormError(null)
  }

  if (authLoading) return <div className="flex items-center justify-center py-8">{t('loading')}</div>
  if (!user || !canManage) return null

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">{t('title')}</h1>
        <Button onClick={() => { resetForm(); setCreateOpen(true) }}>{t('addRole')}</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('rolesList')}</CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-3" />
              <p className="text-red-600 mb-4">{error}</p>
              <Button variant="outline" onClick={() => fetchRoles(page)}>{t('refresh')}</Button>
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
                    <TableHead>{t('permissions')}</TableHead>
                    <TableHead>{t('createdAt')}</TableHead>
                    <TableHead>{t('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {roles.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8">{t('noRoles')}</TableCell>
                    </TableRow>
                  ) : roles.map((role) => (
                    <TableRow key={role.id}>
                      <TableCell>{role.id}</TableCell>
                      <TableCell className="font-medium">{role.name}</TableCell>
                      <TableCell><code className="text-sm bg-gray-100 px-1.5 py-0.5 rounded">{role.code}</code></TableCell>
                      <TableCell>
                        <Badge variant="secondary">{t('permissionCount', { count: role.permissions?.length || 0 })}</Badge>
                      </TableCell>
                      <TableCell>{new Date(role.created_at || '').toLocaleDateString()}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button variant="ghost" size="sm" onClick={() => openEdit(role)}>{t('edit')}</Button>
                          <Button variant="ghost" size="sm" onClick={() => openPermDialog(role)}>{t('assignPermissions')}</Button>
                          <Button variant="ghost" size="sm" className="text-red-600 hover:text-red-700" onClick={() => setDeleteRole(role)}>{t('delete')}</Button>
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

      {/* Create Role Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('createRole')}</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('roleName')}</Label>
              <Input placeholder={t('namePlaceholder')} value={formName} onChange={(e) => setFormName(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('roleCode')}</Label>
              <Input placeholder={t('codePlaceholder')} value={formCode} onChange={(e) => setFormCode(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('roleDescription')}</Label>
              <Input placeholder={t('descriptionPlaceholder')} value={formDescription} onChange={(e) => setFormDescription(e.target.value)} className="mt-1" />
            </div>
            {formError && <p className="text-sm text-red-600">{formError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleCreate} disabled={submitting}>{t('createRole')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Role Dialog */}
      <Dialog open={!!editRole} onOpenChange={(open) => { if (!open) setEditRole(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader><DialogTitle>{t('editRole')}</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('roleName')}</Label>
              <Input value={formName} onChange={(e) => setFormName(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('roleCode')}</Label>
              <Input value={formCode} onChange={(e) => setFormCode(e.target.value)} className="mt-1" />
            </div>
            <div>
              <Label>{t('roleDescription')}</Label>
              <Input value={formDescription} onChange={(e) => setFormDescription(e.target.value)} className="mt-1" />
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
      <Dialog open={!!deleteRole} onOpenChange={(open) => { if (!open) setDeleteRole(null) }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader><DialogTitle>{t('delete')}</DialogTitle></DialogHeader>
          <DialogDescription>{t('confirmDelete')}</DialogDescription>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button variant="destructive" onClick={handleDelete} disabled={submitting}>{t('delete')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Permission Assignment Dialog */}
      <Dialog open={!!permDialogRole} onOpenChange={(open) => { if (!open) setPermDialogRole(null) }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('assignPermissions')} — {permDialogRole?.name}</DialogTitle>
          </DialogHeader>
          <div className="max-h-80 overflow-y-auto space-y-2">
            {allPermissions.map((perm) => {
              const assigned = permDialogRole?.permissions?.some(p => p.id === perm.id)
              return (
                <label key={perm.id} className="flex items-center gap-3 p-2 rounded hover:bg-gray-50 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={!!assigned}
                    onChange={() => handleTogglePermission(perm.id, !!assigned)}
                    disabled={permLoading}
                    className="rounded border-gray-300"
                  />
                  <div className="flex-1">
                    <span className="text-sm font-medium">{perm.name}</span>
                    <span className="text-xs text-gray-500 ml-2">{perm.code}</span>
                  </div>
                  <span className="text-xs text-gray-400">{perm.resource}:{perm.action}</span>
                </label>
              )
            })}
            {allPermissions.length === 0 && <p className="text-sm text-gray-500 text-center py-4">No permissions available</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
