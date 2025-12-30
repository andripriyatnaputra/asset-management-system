import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { AuditSession } from '@/types'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle
} from '@/components/ui/alert-dialog'

interface AuditedAssetInfo {
  asset_name: string
  asset_tag: string
  audit_status: string
}

function toArray<T>(data: any): T[] {
  if (Array.isArray(data)) return data
  if (Array.isArray(data?.data)) return data.data
  return []
}

export default function AuditSessionPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [session, setSession] = useState<AuditSession | null>(null)
  const [items, setItems] = useState<AuditedAssetInfo[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [scanInput, setScanInput] = useState('')
  const [isConfirmOpen, setIsConfirmOpen] = useState(false)

  const fetchDetails = useCallback(() => {
    if (!id) return
    setIsLoading(true)
    apiClient.get(`/audits/${id}`)
      .then(res => {
        // dukung bentuk {session, items} atau {data:{session,items}}
        const payload = res.data?.session ? res.data : res.data?.data
        setSession(payload?.session || null)
        setItems(toArray<AuditedAssetInfo>(payload?.items))
      })
      .catch(() => toast.error('Gagal memuat detail sesi audit.'))
      .finally(() => setIsLoading(false))
  }, [id])

  useEffect(() => { fetchDetails() }, [fetchDetails])

  const handleScan = (e: React.FormEvent) => {
    e.preventDefault()
    const tag = scanInput.trim()
    if (!tag) return

    const p = apiClient.post(`/audits/${id}/scan`, { asset_tag: tag })
    toast.promise(p, {
      loading: 'Memindai...',
      success: () => {
        setScanInput('')
        fetchDetails()
        return `Aset ${tag} berhasil ditemukan!`
      },
      error: (err) => err?.response?.data?.error || 'Gagal memindai aset.',
    })
  }

  const handleCompleteAudit = () => {
    const p = apiClient.put(`/audits/${id}/complete`)
    toast.promise(p, {
      loading: 'Menyelesaikan sesi...',
      success: () => {
        navigate('/audits')
        return 'Sesi audit telah selesai!'
      },
      error: 'Gagal menyelesaikan sesi.',
    })
  }

  const foundCount = items.filter(i => i.audit_status === 'Found').length
  const missingCount = items.length - foundCount
  const totalCount = items.length

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">{session?.name ?? 'Sesi Audit'}</h1>
          <Badge variant={session?.status === 'Completed' ? 'secondary' : 'default'}>
            {session?.status ?? '-'}
          </Badge>
        </div>
        {session?.status !== 'Completed' && (
          <Button variant="destructive" onClick={() => setIsConfirmOpen(true)}>Selesaikan Audit</Button>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Kolom kanan: scan & progress */}
        <div className="lg:col-span-1 space-y-4">
          {session?.status !== 'Completed' && (
            <Card>
              <CardHeader><CardTitle>Pindai Aset</CardTitle></CardHeader>
              <CardContent>
                <form onSubmit={handleScan} className="space-y-2">
                  <Input
                    placeholder="Ketik / scan Asset Tag…"
                    value={scanInput}
                    onChange={e => setScanInput(e.target.value)}
                  />
                  <Button type="submit" className="w-full">Pindai</Button>
                </form>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader><CardTitle>Laporan Ringkas</CardTitle></CardHeader>
            <CardContent className="grid grid-cols-3 gap-2 text-center">
              <div>
                <p className="text-2xl font-bold">{foundCount}</p>
                <p className="text-sm text-muted-foreground">Ditemukan</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-destructive">{missingCount}</p>
                <p className="text-sm text-muted-foreground">Hilang</p>
              </div>
              <div>
                <p className="text-2xl font-bold">{totalCount}</p>
                <p className="text-sm text-muted-foreground">Total</p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Kolom kiri: daftar aset */}
        <div className="lg:col-span-2 border bg-card rounded-lg p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-muted-foreground">Nama Aset</TableHead>
                <TableHead className="text-muted-foreground">Tag Aset</TableHead>
                <TableHead className="text-muted-foreground">Status Audit</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">Memuat…</TableCell>
                </TableRow>
              ) : items.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">Belum ada item yang diaudit.</TableCell>
                </TableRow>
              ) : (
                items.map((item, idx) => (
                  <TableRow key={idx} className="hover:bg-muted/40">
                    <TableCell className="font-medium">{item.asset_name}</TableCell>
                    <TableCell className="font-mono">{item.asset_tag}</TableCell>
                    <TableCell>
                      <Badge variant={item.audit_status === 'Found' ? 'default' : 'destructive'}>
                        {item.audit_status}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Selesaikan Sesi Audit?</AlertDialogTitle>
            <AlertDialogDescription>
              Setelah diselesaikan, Anda tidak bisa lagi memindai aset di sesi ini. Yakin?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleCompleteAudit}>Ya, Selesaikan</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
