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
import { Plus, RefreshCw, GitBranch } from 'lucide-react'

interface ChangeRequest {
  id: number
  cr_number: string
  title: string
  type: string
  status: string
  risk_level: string
  requester_name?: string
  approver_name?: string
  planned_date?: string
  created_at: string
}

const STATUS_COLORS: Record<string, string> = {
  Draft: 'bg-gray-100 text-gray-700',
  Submitted: 'bg-blue-100 text-blue-700',
  'Pending Approval': 'bg-yellow-100 text-yellow-700',
  Approved: 'bg-green-100 text-green-700',
  Rejected: 'bg-red-100 text-red-700',
  'In Progress': 'bg-indigo-100 text-indigo-700',
  Completed: 'bg-green-200 text-green-800',
  'Rolled Back': 'bg-orange-100 text-orange-700',
}

const RISK_COLORS: Record<string, string> = {
  Low: 'bg-blue-100 text-blue-700',
  Medium: 'bg-yellow-100 text-yellow-700',
  High: 'bg-orange-100 text-orange-700',
  Critical: 'bg-red-100 text-red-700',
}

export default function ChangeRequestsPage() {
  const [changes, setChanges] = useState<ChangeRequest[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [isLoading, setIsLoading] = useState(true)
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('all')
  const [type, setType] = useState('all')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ title: '', description: '', type: 'Normal', risk_level: 'Medium' })
  const limit = 20
  const debouncedQ = useDebounce(q, 300)

  const fetchChanges = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (status !== 'all') params.set('status', status)
    if (type !== 'all') params.set('type', type)
    if (debouncedQ.trim()) params.set('q', debouncedQ.trim())
    apiClient.get(`/change-requests?${params}`)
      .then(res => { setChanges(res.data.data ?? []); setTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat data change requests'))
      .finally(() => setIsLoading(false))
  }, [status, type, debouncedQ])

  useEffect(() => { fetchChanges(page) }, [page, fetchChanges])
  useEffect(() => { setPage(1); fetchChanges(1) }, [status, type, debouncedQ, fetchChanges])

  const handleCreate = () => {
    if (!form.title) return toast.error('Title wajib diisi')
    apiClient.post('/change-requests', form)
      .then(() => { toast.success('Change request berhasil dibuat'); setShowCreate(false); fetchChanges(1) })
      .catch(() => toast.error('Gagal membuat change request'))
  }

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GitBranch className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-semibold">Change Management</h1>
          <Badge variant="secondary">{total}</Badge>
        </div>
        <Button onClick={() => setShowCreate(true)}><Plus className="h-4 w-4 mr-1" />CR Baru</Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        <Input placeholder="Cari judul atau nomor CR..." value={q} onChange={e => setQ(e.target.value)} className="w-72" />
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-48"><SelectValue placeholder="Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            {['Draft','Submitted','Pending Approval','Approved','Rejected','In Progress','Completed','Rolled Back'].map(s => (
              <SelectItem key={s} value={s}>{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={type} onValueChange={setType}>
          <SelectTrigger className="w-36"><SelectValue placeholder="Tipe" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Tipe</SelectItem>
            {['Normal','Standard','Emergency'].map(t => (
              <SelectItem key={t} value={t}>{t}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="icon" onClick={() => fetchChanges(page)}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-32">Nomor CR</TableHead>
              <TableHead>Judul</TableHead>
              <TableHead>Tipe</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Risiko</TableHead>
              <TableHead>Requester</TableHead>
              <TableHead>Rencana</TableHead>
              <TableHead>Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
            ) : changes.length === 0 ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
            ) : changes.map(cr => (
              <TableRow key={cr.id}>
                <TableCell className="font-mono text-sm">{cr.cr_number}</TableCell>
                <TableCell className="font-medium">{cr.title}</TableCell>
                <TableCell><Badge variant="outline">{cr.type}</Badge></TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATUS_COLORS[cr.status] ?? 'bg-gray-100 text-gray-600'}`}>{cr.status}</span>
                </TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${RISK_COLORS[cr.risk_level] ?? ''}`}>{cr.risk_level}</span>
                </TableCell>
                <TableCell>{cr.requester_name ?? '-'}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{cr.planned_date ? new Date(cr.planned_date).toLocaleDateString('id-ID') : '-'}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{new Date(cr.created_at).toLocaleDateString('id-ID')}</TableCell>
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
          <DialogHeader><DialogTitle>Buat Change Request</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div><Label>Judul *</Label><Input value={form.title} onChange={e => setForm(f => ({...f, title: e.target.value}))} /></div>
            <div><Label>Deskripsi</Label><Textarea value={form.description} onChange={e => setForm(f => ({...f, description: e.target.value}))} rows={3} /></div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Tipe</Label>
                <Select value={form.type} onValueChange={v => setForm(f => ({...f, type: v}))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>{['Normal','Standard','Emergency'].map(t => <SelectItem key={t} value={t}>{t}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              <div>
                <Label>Level Risiko</Label>
                <Select value={form.risk_level} onValueChange={v => setForm(f => ({...f, risk_level: v}))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>{['Low','Medium','High','Critical'].map(r => <SelectItem key={r} value={r}>{r}</SelectItem>)}</SelectContent>
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
