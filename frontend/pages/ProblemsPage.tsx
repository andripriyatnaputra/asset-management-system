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
import { Plus, RefreshCw, Bug } from 'lucide-react'

interface Problem {
  id: number
  title: string
  status: string
  priority: string
  known_error: boolean
  assignee_name?: string
  related_asset_name?: string
  incident_count?: number
  created_at: string
  resolved_at?: string
}

const STATUS_COLORS: Record<string, string> = {
  Open: 'bg-red-100 text-red-700',
  'Under Investigation': 'bg-yellow-100 text-yellow-700',
  'Known Error': 'bg-orange-100 text-orange-700',
  Resolved: 'bg-green-100 text-green-700',
  Closed: 'bg-gray-100 text-gray-700',
}

const PRIORITY_COLORS: Record<string, string> = {
  Low: 'bg-blue-100 text-blue-700',
  Medium: 'bg-yellow-100 text-yellow-700',
  High: 'bg-orange-100 text-orange-700',
  Critical: 'bg-red-100 text-red-700',
}

export default function ProblemsPage() {
  const [problems, setProblems] = useState<Problem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [isLoading, setIsLoading] = useState(true)
  const [q, setQ] = useState('')
  const [status, setStatus] = useState('all')
  const [priority, setPriority] = useState('all')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ title: '', description: '', priority: 'Medium' })
  const limit = 20
  const debouncedQ = useDebounce(q, 300)

  const fetchProblems = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (status !== 'all') params.set('status', status)
    if (priority !== 'all') params.set('priority', priority)
    if (debouncedQ.trim()) params.set('q', debouncedQ.trim())
    apiClient.get(`/problems?${params}`)
      .then(res => { setProblems(res.data.data ?? []); setTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat data problems'))
      .finally(() => setIsLoading(false))
  }, [status, priority, debouncedQ])

  useEffect(() => { fetchProblems(page) }, [page, fetchProblems])
  useEffect(() => { setPage(1); fetchProblems(1) }, [status, priority, debouncedQ, fetchProblems])

  const handleCreate = () => {
    if (!form.title) return toast.error('Title wajib diisi')
    apiClient.post('/problems', form)
      .then(() => { toast.success('Problem berhasil dibuat'); setShowCreate(false); fetchProblems(1) })
      .catch(() => toast.error('Gagal membuat problem'))
  }

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Bug className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-semibold">Problem Management</h1>
          <Badge variant="secondary">{total}</Badge>
        </div>
        <Button onClick={() => setShowCreate(true)}><Plus className="h-4 w-4 mr-1" />Problem Baru</Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2">
        <Input placeholder="Cari judul..." value={q} onChange={e => setQ(e.target.value)} className="w-64" />
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-48"><SelectValue placeholder="Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            {['Open','Under Investigation','Known Error','Resolved','Closed'].map(s => (
              <SelectItem key={s} value={s}>{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={priority} onValueChange={setPriority}>
          <SelectTrigger className="w-40"><SelectValue placeholder="Priority" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Priority</SelectItem>
            {['Low','Medium','High','Critical'].map(p => (
              <SelectItem key={p} value={p}>{p}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="icon" onClick={() => fetchProblems(page)}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">ID</TableHead>
              <TableHead>Judul</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Known Error</TableHead>
              <TableHead>Assigned To</TableHead>
              <TableHead>Incidents</TableHead>
              <TableHead>Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
            ) : problems.length === 0 ? (
              <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
            ) : problems.map(p => (
              <TableRow key={p.id}>
                <TableCell className="font-mono text-sm">{p.id}</TableCell>
                <TableCell className="font-medium">{p.title}</TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${STATUS_COLORS[p.status] ?? 'bg-gray-100 text-gray-600'}`}>{p.status}</span>
                </TableCell>
                <TableCell>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${PRIORITY_COLORS[p.priority] ?? ''}`}>{p.priority}</span>
                </TableCell>
                <TableCell>{p.known_error ? <Badge variant="destructive">Ya</Badge> : <Badge variant="secondary">Tidak</Badge>}</TableCell>
                <TableCell>{p.assignee_name ?? '-'}</TableCell>
                <TableCell>{p.incident_count ?? 0}</TableCell>
                <TableCell className="text-sm text-muted-foreground">{new Date(p.created_at).toLocaleDateString('id-ID')}</TableCell>
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
          <DialogHeader><DialogTitle>Buat Problem Baru</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div><Label>Judul *</Label><Input value={form.title} onChange={e => setForm(f => ({...f, title: e.target.value}))} /></div>
            <div><Label>Deskripsi</Label><Textarea value={form.description} onChange={e => setForm(f => ({...f, description: e.target.value}))} rows={3} /></div>
            <div>
              <Label>Priority</Label>
              <Select value={form.priority} onValueChange={v => setForm(f => ({...f, priority: v}))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>{['Low','Medium','High','Critical'].map(p => <SelectItem key={p} value={p}>{p}</SelectItem>)}</SelectContent>
              </Select>
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
