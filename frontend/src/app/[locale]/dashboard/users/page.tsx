"use client"

import { useEffect, useState } from "react"
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
import { api } from "@/lib/api"
import { useAuth } from "@/lib/auth-provider"
import { useErrorHandler } from "@/lib/use-error-handler"
import { AlertCircle } from "lucide-react"

interface UserItem {
  id: number
  username: string
  email: string
  status: string
  created_at: string
}

export default function UsersPage() {
  const t = useTranslations('users')
  const { user, loading: authLoading } = useAuth()
  const { getError } = useErrorHandler()
  const [users, setUsers] = useState<UserItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!authLoading && user) {
      fetchUsers()
    }
  }, [authLoading, user])

  const fetchUsers = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.listUsers(1, 10)
      setUsers((data.users as unknown as UserItem[]) || [])
    } catch (err) {
      setError(getError(err))
    } finally {
      setLoading(false)
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
        <Button>{t('addUser')}</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('usersList')}</CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <div className="text-center py-8">
              <AlertCircle className="h-12 w-12 text-red-400 mx-auto mb-3" />
              <p className="text-red-600 mb-4">{error}</p>
              <Button variant="outline" onClick={fetchUsers}>{t('edit')}</Button>
            </div>
          ) : loading ? (
            <div className="text-center py-8">{t('loading')}</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('userDetails')}</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created At</TableHead>
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
                      <TableCell>{u.username}</TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            u.status === "active" ? "default" : "secondary"
                          }
                        >
                          {u.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {new Date(u.created_at).toLocaleDateString()}
                      </TableCell>
                      <TableCell>
                        <Button variant="ghost" size="sm">
                          {t('edit')}
                        </Button>
                        <Button variant="ghost" size="sm">
                          {t('delete')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
