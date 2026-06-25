import { useEffect, useState, useCallback } from 'react'
import { useDebounce } from '@/hooks/useDebounce'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Plus, RefreshCw, Headphones } from 'lucide-react'

interface ServiceRequest {
  id: number
  sr_number: string
  subject: string
  status: string
  category: string
  priority: string
  requester_name?: string
  assigned_to_name?: string
  fulfilled_at?: string
  created_at: string
}

const STATUS_COLORS: Record<string, string> = {
  New: 'bg-blue-100 text-blue-700',
  'In Progress': 'bg-indigo-100 text-indigo-700',
  'Pending Approval': 'bg-yellow-100 text-yellow-700',
  Approved: 'bg-green-100 text-green-700',
  Fulfilled: 'bg-green-200 text-green-800',
  Cancelled: 'bg-gray-100 text-gray-700',
  Rejected: 'bg-red-100 text-red-700',
}

const PRIORITY_COLORS: Record<string, string> = {
  Low: 'bg-blue-100 text-blue-700',
  Medium: 'bg-yellow-100 text-yellow-700',
  High: 'bg-orange-100 text-orange-700',
  Urgent: 'bg-red-100 text-red-700',
}

export default function ServiceRequestsPage() {
  const [requests, setRequests] = useState<ServiceRequest[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [isLoading, setIsLoading] = useState(true)
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('all')
  const [myOnly, setMyOnly] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ subject: '', description: '', category: 'Access Request', priority: 'Medium' })
  const limit = 20
  const debouncedQ = useDebounce(q, 300)

  const fetchRequests = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (status !== 'all') params.set('status', status)
    if (myOnly) params.set('my', 'true')
    if (debouncedQ.trim()) params.set('q', debouncedQ.trim())
    apiClient.get(`/service-requests?${params}`)
      .then(res => { setRequests(res.data.data ?? []); setTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat data service requests'))
      .finally(() => setIsLoading(false))
  }, [status, myOnly, debouncedQ])

  useEffect(() => { fetchRequests(page) }, [page, fetchRequests])
  useEffect(() => { setPage(1); fetchRequests(1) }, [status, myOnly, debouncedQ, fetchRequests])

  const handleCreate = () => {
    if (!form.subject) return toast.error('Subject wajib diisi')
    apiClient.post('/service-requests', form)
      .then(() => { toast.success('Service request berhasil dibuat'); setShowCreate(false); fetchRequests(1) })
      .catch(() => toast.error('Gagal membuat service request'))
  }

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Headphones className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-semibold">Service Requests</h1>
          <Badge variant="secondary">{total}</Badge>
        </div>
        <Button onClick={() => setShowCreate(true)}><Plus className="h-4 w-4 mr-1" />SR Baru</Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        <Input placeholder="Cari subject atau nomor SR..." value={q} onChange={e => setQ(e.target.value)} className="w-72" />
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-48"><SelectValue placeholder="Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            {['New','In Progress','Pending Approval','Approved','Fulfilled','Cancelled','Rejected'].map(s => (
              <SelectItem key={s} value={s}>{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant={myOnly ? 'default' : 'outline'} size="sm" onClick={() => setMyOnly(v => !v)}>
          {myOnly ? 'SR Saya' : 'Semua SR'}
        </Button>
        <Button variant="outline" size="icon" onClick={() => fetchRequests(page)}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-32">Nomor SR</TableHead>
              <TableHead>Subject</TableHead>
              <TableHead>Kategori</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Requester</TableHead>
              <TableHead>Ditugaskan</TableHead>
              <TableHead>Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
            ) : requests.length === 0 ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
            ) : requests.map(sr => (
              <TableRow key={sr.id}>
                <TableCell className="font-mono text-sm">{sr.sr_number}</TableCell>
                <TableCell className="font-medium">{sr.subject}</TableCell>
                <TableCell><Badge variant="outline">{sr.category}</Badge></TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATUS_COLORS[sr.status] ?? 'bg-gray-100 text-gray-600'}`}>{sr.status}</span>
                </TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${PRIORITY_COLORS[sr.priority] ?? ''}`}>{sr.priority}</span>
                </TableCell>
                <TableCell>{sr.requester_name ?? '-'}</TableCell>
                <TableCell>{sr.assigned_to_name ?? '-'}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{new Date(sr.created_at).toLocaleDateString('id-ID')}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Halaman {page} dari {totalPages} ({total} total)</span>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>Sebelumnya</Button>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Selanjutnya</Button>
          </div>
        </div>
      )}

      {/* Create Modal */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader><DialogTitle>Buat Service Request</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div><Label>Subject *</Label><Input value={form.subject} onChange={e => setForm(f => ({...f, subject: e.target.value}))} /></div>
            <div><Label>Deskripsi</Label><Textarea value={form.description} onChange={e => setForm(f => ({...f, description: e.target.value}))} rows={3} /></div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Kategori</Label>
                <Select value={form.category} onValueChange={v => setForm(f => ({...f, category: v}))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {['Access Request','Hardware Request','Software Request','Service Provision','Other'].map(c => (
                      <SelectItem key={c} value={c}>{c}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label>Priority</Label>
                <Select value={form.priority} onValueChange={v => setForm(f => ({...f, priority: v}))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>{['Low','Medium','High','Urgent'].map(p => <SelectItem key={p} value={p}>{p}</SelectItem>)}</SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowCreate(false)}>Batal</Button>
            <Button onClick={handleCreate}>Buat</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
