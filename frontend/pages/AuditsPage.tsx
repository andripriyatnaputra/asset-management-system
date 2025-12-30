import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { AuditSession } from '@/types'

import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import CreateAuditModal from '@/components/CreateAuditModal'

function toArray<T>(data: any): T[] {
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.data)) return data.data
  return []
}

export default function AuditsPage() {
  const [sessions, setSessions] = useState<AuditSession[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const navigate = useNavigate()

  const fetchSessions = () => {
    setIsLoading(true)
    apiClient.get('/audits')
      .then(res => setSessions(toArray<AuditSession>(res.data)))
      .catch(() => toast.error('Gagal memuat sesi audit.'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { fetchSessions() }, [])

  const handleSuccess = () => {
    setIsModalOpen(false)
    fetchSessions()
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">Sesi Audit Aset</h1>
        <Button onClick={() => setIsModalOpen(true)}>+ Mulai Sesi Baru</Button>
      </div>

      <div className="border bg-card rounded-lg p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-muted-foreground">Nama Sesi</TableHead>
              <TableHead className="text-muted-foreground">Status</TableHead>
              <TableHead className="text-muted-foreground">Tanggal Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">Memuat…</TableCell>
              </TableRow>
            ) : sessions.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">Belum ada sesi audit.</TableCell>
              </TableRow>
            ) : (
              sessions.map((session) => (
                <TableRow
                  key={session.id}
                  onClick={() => navigate(`/audits/${session.id}`)}
                  className="cursor-pointer hover:bg-muted/50"
                >
                  <TableCell className="font-medium">{session.name}</TableCell>
                  <TableCell>
                    <Badge variant={session.status === 'Completed' ? 'secondary' : 'default'}>
                      {session.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{new Date(session.created_at).toLocaleDateString('id-ID')}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <CreateAuditModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSuccess={handleSuccess}
      />
    </div>
  )
}
