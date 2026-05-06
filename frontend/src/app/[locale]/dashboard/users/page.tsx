"use client"

import { useEffect, useState, useCallback, useRef } from "react"
import { useTranslations } from 'next-intl'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
  DialogDescription,
} from "@/components/ui/dialog"
import { api } from "@/lib/api"
import { useAuth, hasPermission } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { AlertCircle, Search, X } from "lucide-react"
import type { Role } from "@/lib/types"

interface UserItem {
  id: number
  username: string
  email: string
  status: string
  created_at: string
  roles?: Role[]
}

export default function UsersPage() {
  const t = useTranslations('users')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()

  const [users, setUsers] = useState<UserItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [searchQuery, setSearchQuery] = useState("")
  const pageSize = 10

  // Dialog states
  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<UserItem | null>(null)
  const [deleteUser, setDeleteUser] = useState<UserItem | null>(null)
  const [roleUser, setRoleUser] = useState<UserItem | null>(null)

  // Form states
  const [formUsername, setFormUsername] = useState("")
  const [formEmail, setFormEmail] = useState("")
  const [formPassword, setFormPassword] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  // Role assignment states
  const [allRoles, setAllRoles] = useState<Role[]>([])
  const [selectedRoleId, setSelectedRoleId] = useState<string>("")
  const [roleLoading, setRoleLoading] = useState(false)

  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const canWrite = hasPermission(user, "users:write")
  const canDelete = hasPermission(user, "users:delete")

  const fetchUsers = useCallback(async (p: number, status: string, search: string) => {
    setLoading(true)
    setError(null)
    try {
      const filters: { status?: string; search?: string } = {}
      if (status && status !== "all") filters.status = status
      if (search.trim()) filters.search = search.trim()
      const data = await api.listUsers(p, pageSize, filters)
      setUsers((data.users as unknown as UserItem[]) || [])
      setTotal(data.total)
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
    }
  }, [getError])

  useEffect(() => {
    if (!authLoading && user) {
      fetchUsers(page, statusFilter, searchQuery)
    }
  }, [authLoading, user, page, statusFilter, fetchUsers])

  const handleSearchChange = (value: string) => {
    setSearchQuery(value)
    if (searchTimerRef.current) clearTimeout(searchTimerRef.current)
    searchTimerRef.current = setTimeout(() => {
      setPage(1)
      fetchUsers(1, statusFilter, value)
    }, 300)
  }

  const handleStatusFilterChange = (value: string | null) => {
    const status = value ?? "all"
    setStatusFilter(status)
    setPage(1)
    fetchUsers(1, status, searchQuery)
  }

  const handleToggleStatus = async (userId: number, currentStatus: string) => {
    const newStatus = currentStatus === "active" ? "disabled" : "active"
    try {
      await api.updateUserStatus(userId, newStatus)
      fetchUsers(page, statusFilter, searchQuery)
    } catch (err) {
      setError(getError(err))
    }
  }

  // Create user
  const handleCreate = async () => {
    setFormError(null)
    if (!formUsername.trim() || !formEmail.trim() || !formPassword.trim()) {
      setFormError(t('validationAllRequired'))
      return
    }
    setSubmitting(true)
    try {
      await api.createUser({ username: formUsername.trim(), email: formEmail.trim(), password: formPassword })
      setCreateOpen(false)
      resetForm()
      fetchUsers(page, statusFilter, searchQuery)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  // Edit user
  const openEdit = (u: UserItem) => {
    setFormUsername(u.username)
    setFormEmail(u.email)
    setFormError(null)
    setEditUser(u)
  }

  const handleEdit = async () => {
    if (!editUser) return
    setFormError(null)
    if (!formUsername.trim() || !formEmail.trim()) {
      setFormError(t('validationUsernameEmailRequired'))
      return
    }
    setSubmitting(true)
    try {
      await api.updateUser(editUser.id, { username: formUsername.trim(), email: formEmail.trim() })
      setEditUser(null)
      resetForm()
      fetchUsers(page, statusFilter, searchQuery)
    } catch (err) {
      setFormError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  // Delete user
  const handleDelete = async () => {
    if (!deleteUser) return
    setSubmitting(true)
    try {
      await api.deleteUser(deleteUser.id)
      setDeleteUser(null)
      fetchUsers(page, statusFilter, searchQuery)
    } catch (err) {
      setError(getError(err))
    } finally {
      setSubmitting(false)
    }
  }

  // Role assignment
  const openRoleDialog = async (u: UserItem) => {
    setRoleUser(u)
    setSelectedRoleId("")
    setRoleLoading(true)
    try {
      const data = await api.listRoles(1, 100)
      setAllRoles(data.roles || [])
    } catch (err) {
      setError(getError(err))
    } finally {
      setRoleLoading(false)
    }
  }

  const handleAssignRole = async () => {
    if (!roleUser || !selectedRoleId) return
    setRoleLoading(true)
    try {
      await api.assignRole(roleUser.id, Number(selectedRoleId))
      const data = await api.listRoles(1, 100)
      setAllRoles(data.roles || [])
      // Refresh user with updated roles
      const updated = await api.getUser(roleUser.id)
      setRoleUser({ ...roleUser, roles: (updated as unknown as UserItem).roles })
      fetchUsers(page, statusFilter, searchQuery)
      setSelectedRoleId("")
    } catch (err) {
      setError(getError(err))
    } finally {
      setRoleLoading(false)
    }
  }

  const handleRemoveRole = async (roleId: number) => {
    if (!roleUser) return
    setRoleLoading(true)
    try {
      await api.removeRole(roleUser.id, roleId)
      const updated = await api.getUser(roleUser.id)
      setRoleUser({ ...roleUser, roles: (updated as unknown as UserItem).roles })
      fetchUsers(page, statusFilter, searchQuery)
    } catch (err) {
      setError(getError(err))
    } finally {
      setRoleLoading(false)
    }
  }

  const resetForm = () => {
    setFormUsername("")
    setFormEmail("")
    setFormPassword("")
    setFormError(null)
  }

  const totalPages = Math.ceil(total / pageSize)

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active":
        return <Badge variant="default">{t('statusActive')}</Badge>
      case "disabled":
        return <Badge variant="secondary" className="bg-red-100 text-red-700">{t('statusDisabled')}</Badge>
      default:
        return <Badge variant="secondary">{status}</Badge>
    }
  }

  if (authLoading) {
    return <div className="flex items-center justify-center py-8">{t('loading')}</div>
  }

  if (!user) {
    return null
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">{t('title')}</h1>
        {canWrite && <Button onClick={() => { resetForm(); setCreateOpen(true) }}>{t('addUser')}</Button>}
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <CardTitle>{t('usersList')}</CardTitle>
            <div className="flex items-center gap-3">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
                <Input
                  placeholder={t('searchPlaceholder')}
                  value={searchQuery}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
              <Select value={statusFilter} onValueChange={handleStatusFilterChange}>
                <SelectTrigger className="w-36">
                  <SelectValue placeholder={t('allStatus')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('allStatus')}</SelectItem>
                  <SelectItem value="active">{t('statusActive')}</SelectItem>
                  <SelectItem value="disabled">{t('statusDisabled')}</SelectItem>
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
              <Button variant="outline" onClick={() => fetchUsers(page, statusFilter, searchQuery)}>{t('refresh')}</Button>
            </div>
          ) : loading ? (
            <div className="text-center py-8">{t('loading')}</div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t('userDetails')}</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>{t('status')}</TableHead>
                    <TableHead>{t('createdAt')}</TableHead>
                    <TableHead>{t('actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {users.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6} className="text-center py-8">
                        {t('noUsers')}
                      </TableCell>
                    </TableRow>
                  ) : (
                    users.map((u) => (
                      <TableRow key={u.id}>
                        <TableCell>{u.id}</TableCell>
                        <TableCell className="font-medium">{u.username}</TableCell>
                        <TableCell>{u.email}</TableCell>
                        <TableCell>{getStatusBadge(u.status)}</TableCell>
                        <TableCell>{new Date(u.created_at).toLocaleDateString()}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            {canWrite && (
                              <Button variant="ghost" size="sm" onClick={() => openEdit(u)}>
                                {t('edit')}
                              </Button>
                            )}
                            {canWrite && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleToggleStatus(u.id, u.status)}
                              >
                                {u.status === "active" ? t('disable') : t('enable')}
                              </Button>
                            )}
                            {canWrite && (
                              <Button variant="ghost" size="sm" onClick={() => openRoleDialog(u)}>
                                {t('assignRole')}
                              </Button>
                            )}
                            {canDelete && (
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-red-600 hover:text-red-700"
                                onClick={() => setDeleteUser(u)}
                              >
                                {t('delete')}
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4 pt-4 border-t">
                  <p className="text-sm text-gray-500">
                    {t('total', { count: total })}
                  </p>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page <= 1}
                      onClick={() => setPage(p => p - 1)}
                    >
                      {t('prev')}
                    </Button>
                    <span className="text-sm text-gray-600">
                      {page} / {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={page >= totalPages}
                      onClick={() => setPage(p => p + 1)}
                    >
                      {t('next')}
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* Create User Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('createUser')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('username')}</Label>
              <Input
                placeholder={t('usernamePlaceholder')}
                value={formUsername}
                onChange={(e) => setFormUsername(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t('email')}</Label>
              <Input
                type="email"
                placeholder={t('emailPlaceholder')}
                value={formEmail}
                onChange={(e) => setFormEmail(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t('password')}</Label>
              <Input
                type="password"
                placeholder={t('passwordPlaceholder')}
                value={formPassword}
                onChange={(e) => setFormPassword(e.target.value)}
                className="mt-1"
              />
            </div>
            {formError && <p className="text-sm text-red-600">{formError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleCreate} disabled={submitting}>
              {t('createUser')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit User Dialog */}
      <Dialog open={!!editUser} onOpenChange={(open) => { if (!open) setEditUser(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('editUser')}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>{t('username')}</Label>
              <Input
                placeholder={t('usernamePlaceholder')}
                value={formUsername}
                onChange={(e) => setFormUsername(e.target.value)}
                className="mt-1"
              />
            </div>
            <div>
              <Label>{t('email')}</Label>
              <Input
                type="email"
                placeholder={t('emailPlaceholder')}
                value={formEmail}
                onChange={(e) => setFormEmail(e.target.value)}
                className="mt-1"
              />
            </div>
            {formError && <p className="text-sm text-red-600">{formError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button onClick={handleEdit} disabled={submitting}>
              {t('save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deleteUser} onOpenChange={(open) => { if (!open) setDeleteUser(null) }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{t('deleteConfirm')}</DialogTitle>
          </DialogHeader>
          <DialogDescription>{t('confirmDeleteMessage')}</DialogDescription>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
            <Button variant="destructive" onClick={handleDelete} disabled={submitting}>
              {t('delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Role Assignment Dialog */}
      <Dialog open={!!roleUser} onOpenChange={(open) => { if (!open) setRoleUser(null) }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('assignRole')} — {roleUser?.username}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {/* Current roles */}
            <div>
              <Label className="mb-2 block">{t('roles')}</Label>
              <div className="flex flex-wrap gap-2">
                {roleUser?.roles && roleUser.roles.length > 0 ? (
                  roleUser.roles.map((role) => (
                    <Badge key={role.id} variant="secondary" className="gap-1">
                      {role.name}
                      <button
                        onClick={() => handleRemoveRole(role.id)}
                        className="ml-1 hover:text-red-600"
                        disabled={roleLoading}
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))
                ) : (
                  <span className="text-sm text-gray-500">{t('noRoles')}</span>
                )}
              </div>
            </div>
            {/* Add role */}
            <div className="flex items-center gap-2">
              <Select value={selectedRoleId} onValueChange={(v) => setSelectedRoleId(v ?? "")}>
                <SelectTrigger className="flex-1">
                  <SelectValue placeholder={t('selectRole')} />
                </SelectTrigger>
                <SelectContent>
                  {allRoles
                    .filter(r => !roleUser?.roles?.some(ur => ur.id === r.id))
                    .map((role) => (
                      <SelectItem key={role.id} value={String(role.id)}>
                        {role.name} ({role.code})
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              <Button onClick={handleAssignRole} disabled={!selectedRoleId || roleLoading} size="sm">
                {t('assignRole')}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" />}>{t('cancel')}</DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
