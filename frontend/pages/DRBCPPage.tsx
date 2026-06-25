import { useEffect, useState, useCallback } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Plus, RefreshCw, Shield, ClipboardList } from 'lucide-react'

interface DRPlan {
  id: number
  name: string
  type: string
  status: string
  rto_hours?: number
  rpo_hours?: number
  owner_name?: string
  last_tested_at?: string
  next_test_due?: string
  created_at: string
}

interface DRTest {
  id: number
  plan_id: number
  plan_name?: string
  test_date: string
  status: string
  result?: string
  tested_by_name?: string
  duration_minutes?: number
  notes?: string
}

const PLAN_STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  active: 'bg-green-100 text-green-700',
  under_review: 'bg-yellow-100 text-yellow-700',
  retired: 'bg-gray-200 text-gray-600',
}

const TEST_STATUS_COLORS: Record<string, string> = {
  Scheduled: 'bg-blue-100 text-blue-700',
  'In Progress': 'bg-indigo-100 text-indigo-700',
  Completed: 'bg-green-100 text-green-700',
  Failed: 'bg-red-100 text-red-700',
  Cancelled: 'bg-gray-100 text-gray-700',
}

export default function DRBCPPage() {
  const [tab, setTab] = useState('plans')

  const [plans, setPlans] = useState<DRPlan[]>([])
  const [planTotal, setPlanTotal] = useState(0)
  const [planPage, setPlanPage] = useState(1)

  const [tests, setTests] = useState<DRTest[]>([])
  const [testTotal, setTestTotal] = useState(0)
  const [testPage, setTestPage] = useState(1)

  const [isLoading, setIsLoading] = useState(false)
  const [q, setQ] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [typeFilter, setTypeFilter] = useState('all')
  const [showCreate, setShowCreate] = useState(false)
  const [form, setForm] = useState({ name: '', type: 'DR', description: '', rto_hours: '', rpo_hours: '' })
  const limit = 20

  const fetchPlans = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (statusFilter !== 'all') params.set('status', statusFilter)
    if (typeFilter !== 'all') params.set('type', typeFilter)
    if (q.trim()) params.set('q', q.trim())
    apiClient.get(`/dr/plans?${params}`)
      .then(res => { setPlans(res.data.data ?? []); setPlanTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat DR plans'))
      .finally(() => setIsLoading(false))
  }, [statusFilter, typeFilter, q])

  const fetchTests = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (statusFilter !== 'all') params.set('status', statusFilter)
    apiClient.get(`/dr/tests?${params}`)
      .then(res => { setTests(res.data.data ?? []); setTestTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat DR tests'))
      .finally(() => setIsLoading(false))
  }, [statusFilter])

  useEffect(() => {
    if (tab === 'plans') fetchPlans(planPage)
    else fetchTests(testPage)
  }, [tab, planPage, testPage, fetchPlans, fetchTests])

  useEffect(() => {
    setPlanPage(1); setTestPage(1)
    if (tab === 'plans') fetchPlans(1)
    else fetchTests(1)
  }, [q, statusFilter, typeFilter])

  const handleCreate = () => {
    if (!form.name) return toast.error('Nama plan wajib diisi')
    const payload = {
      ...form,
      rto_hours: form.rto_hours ? parseInt(form.rto_hours) : null,
      rpo_hours: form.rpo_hours ? parseInt(form.rpo_hours) : null,
    }
    apiClient.post('/dr/plans', payload)
      .then(() => { toast.success('DR Plan berhasil dibuat'); setShowCreate(false); fetchPlans(1) })
      .catch(() => toast.error('Gagal membuat DR plan'))
  }

  const renderPagination = (page: number, total: number, setPage: (n: number) => void) => {
    const totalPages = Math.ceil(total / limit)
    if (totalPages <= 1) return null
    return (
      <div className="flex items-center justify-between mt-4">
        <span className="text-sm text-muted-foreground">Halaman {page} dari {totalPages} ({total} total)</span>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>Sebelumnya</Button>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>Selanjutnya</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Shield className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-semibold">DR / BCP Management</h1>
        </div>
        {tab === 'plans' && <Button onClick={() => setShowCreate(true)}><Plus className="h-4 w-4 mr-1" />Plan Baru</Button>}
      </div>

      <Tabs value={tab} onValueChange={t => { setTab(t); setStatusFilter('all'); setTypeFilter('all'); setQ('') }}>
        <TabsList>
          <TabsTrigger value="plans"><Shield className="h-4 w-4 mr-1" />DR/BCP Plans <Badge variant="secondary" className="ml-1">{planTotal}</Badge></TabsTrigger>
          <TabsTrigger value="tests"><ClipboardList className="h-4 w-4 mr-1" />Test Records <Badge variant="secondary" className="ml-1">{testTotal}</Badge></TabsTrigger>
        </TabsList>

        <div className="flex flex-wrap gap-2 mt-4">
          {tab === 'plans' && <Input placeholder="Cari nama plan..." value={q} onChange={e => setQ(e.target.value)} className="w-64" />}
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-40"><SelectValue placeholder="Status" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Status</SelectItem>
              {tab === 'plans'
                ? ['draft','active','under_review','retired'].map(s => <SelectItem key={s} value={s}>{s}</SelectItem>)
                : ['Scheduled','In Progress','Completed','Failed','Cancelled'].map(s => <SelectItem key={s} value={s}>{s}</SelectItem>)
              }
            </SelectContent>
          </Select>
          {tab === 'plans' && (
            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger className="w-36"><SelectValue placeholder="Tipe" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua Tipe</SelectItem>
                {['DR','BCP','COOP'].map(t => <SelectItem key={t} value={t}>{t}</SelectItem>)}
              </SelectContent>
            </Select>
          )}
          <Button variant="outline" size="icon" onClick={() => tab === 'plans' ? fetchPlans(planPage) : fetchTests(testPage)}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>

        <TabsContent value="plans" className="mt-4">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nama Plan</TableHead>
                  <TableHead>Tipe</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>RTO (jam)</TableHead>
                  <TableHead>RPO (jam)</TableHead>
                  <TableHead>Owner</TableHead>
                  <TableHead>Test Terakhir</TableHead>
                  <TableHead>Test Berikutnya</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : plans.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
                ) : plans.map(p => (
                  <TableRow key={p.id}>
                    <TableCell className="font-medium">{p.name}</TableCell>
                    <TableCell><Badge variant="outline">{p.type}</Badge></TableCell>
                    <TableCell><span className={`px-2 py-1 rounded-full text-xs font-medium ${PLAN_STATUS_COLORS[p.status] ?? 'bg-gray-100 text-gray-600'}`}>{p.status}</span></TableCell>
                    <TableCell>{p.rto_hours ?? '-'}</TableCell>
                    <TableCell>{p.rpo_hours ?? '-'}</TableCell>
                    <TableCell>{p.owner_name ?? '-'}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{p.last_tested_at ? new Date(p.last_tested_at).toLocaleDateString('id-ID') : '-'}</TableCell>
                    <TableCell className={`text-sm font-medium ${p.next_test_due && new Date(p.next_test_due) < new Date() ? 'text-red-600' : 'text-muted-foreground'}`}>
                      {p.next_test_due ? new Date(p.next_test_due).toLocaleDateString('id-ID') : '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {renderPagination(planPage, planTotal, setPlanPage)}
        </TabsContent>

        <TabsContent value="tests" className="mt-4">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Plan</TableHead>
                  <TableHead>Tanggal Test</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Hasil</TableHead>
                  <TableHead>Tested By</TableHead>
                  <TableHead>Durasi (mnt)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : tests.length === 0 ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
                ) : tests.map(t => (
                  <TableRow key={t.id}>
                    <TableCell className="font-medium">{t.plan_name ?? `Plan #${t.plan_id}`}</TableCell>
                    <TableCell className="text-sm">{new Date(t.test_date).toLocaleDateString('id-ID')}</TableCell>
                    <TableCell><span className={`px-2 py-1 rounded-full text-xs font-medium ${TEST_STATUS_COLORS[t.status] ?? 'bg-gray-100 text-gray-600'}`}>{t.status}</span></TableCell>
                    <TableCell>{t.result ? <Badge variant={t.result === 'Pass' ? 'default' : 'destructive'}>{t.result}</Badge> : '-'}</TableCell>
                    <TableCell>{t.tested_by_name ?? '-'}</TableCell>
                    <TableCell>{t.duration_minutes ?? '-'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {renderPagination(testPage, testTotal, setTestPage)}
        </TabsContent>
      </Tabs>

      {/* Create Plan Modal */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader><DialogTitle>Buat DR/BCP Plan</DialogTitle></DialogHeader>
          <div className="space-y-4">
            <div><Label>Nama Plan *</Label><Input value={form.name} onChange={e => setForm(f => ({...f, name: e.target.value}))} /></div>
            <div><Label>Deskripsi</Label><Textarea value={form.description} onChange={e => setForm(f => ({...f, description: e.target.value}))} rows={2} /></div>
            <div className="grid grid-cols-3 gap-3">
              <div>
                <Label>Tipe</Label>
                <Select value={form.type} onValueChange={v => setForm(f => ({...f, type: v}))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>{['DR','BCP','COOP'].map(t => <SelectItem key={t} value={t}>{t}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              <div><Label>RTO (jam)</Label><Input type="number" value={form.rto_hours} onChange={e => setForm(f => ({...f, rto_hours: e.target.value}))} /></div>
              <div><Label>RPO (jam)</Label><Input type="number" value={form.rpo_hours} onChange={e => setForm(f => ({...f, rpo_hours: e.target.value}))} /></div>
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
