import { useEffect, useState, useCallback } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { RefreshCw, ShieldCheck, Trash2, FileText, FileDown } from 'lucide-react'
import { downloadFile } from '@/lib/utils'

interface FrameworkSummary {
  id: number
  code: string
  name: string
  total_controls: number
  covered_controls: number
  coverage_pct?: number
}

interface DisposalCompliance {
  asset_id: number
  asset_name: string
  asset_tag: string
  lifecycle_stage: string
  disposal_method?: string
  data_wipe_completed?: boolean
  environmental_compliant?: boolean
  certificate_number?: string
  date_disposed?: string
  compliance_status: string
}

interface AuditLog {
  id: number
  entity_type: string
  entity_id: number
  action: string
  changed_by?: number
  old_values?: Record<string, unknown>
  new_values?: Record<string, unknown>
  changed_at: string
}

const DISPOSAL_STATUS_COLORS: Record<string, string> = {
  compliant: 'bg-green-100 text-green-700',
  data_wipe_pending: 'bg-orange-100 text-orange-700',
  env_non_compliant: 'bg-red-100 text-red-700',
  missing_record: 'bg-gray-100 text-gray-600',
}

const ACTION_COLORS: Record<string, string> = {
  CREATE: 'bg-green-100 text-green-700',
  UPDATE: 'bg-blue-100 text-blue-700',
  DELETE: 'bg-red-100 text-red-700',
  EXPORT: 'bg-purple-100 text-purple-700',
}

export default function CompliancePage() {
  const [tab, setTab] = useState('frameworks')

  const [frameworks, setFrameworks] = useState<FrameworkSummary[]>([])
  const [disposals, setDisposals] = useState<DisposalCompliance[]>([])
  const [disposalStatus, setDisposalStatus] = useState('all')
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [auditPage, setAuditPage] = useState(1)
  const [auditTotal, setAuditTotal] = useState(0)

  const [isLoading, setIsLoading] = useState(false)
  const limit = 25

  const fetchFrameworks = useCallback(() => {
    setIsLoading(true)
    apiClient.get('/compliance/summary')
      .then(res => setFrameworks(res.data ?? []))
      .catch(() => toast.error('Gagal memuat framework compliance'))
      .finally(() => setIsLoading(false))
  }, [])

  const fetchDisposals = useCallback(() => {
    setIsLoading(true)
    const params = new URLSearchParams()
    if (disposalStatus !== 'all') params.set('status', disposalStatus)
    apiClient.get(`/compliance/disposal?${params}`)
      .then(res => setDisposals(res.data ?? []))
      .catch(() => toast.error('Gagal memuat disposal compliance'))
      .finally(() => setIsLoading(false))
  }, [disposalStatus])

  const fetchAuditLogs = useCallback((p: number) => {
    setIsLoading(true)
    apiClient.get(`/audit-logs?page=${p}&limit=${limit}`)
      .then(res => { setAuditLogs(res.data.data ?? res.data ?? []); setAuditTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat audit trail'))
      .finally(() => setIsLoading(false))
  }, [])

  useEffect(() => {
    if (tab === 'frameworks') fetchFrameworks()
    else if (tab === 'disposal') fetchDisposals()
    else if (tab === 'audit') fetchAuditLogs(auditPage)
  }, [tab, auditPage, fetchFrameworks, fetchDisposals, fetchAuditLogs])

  useEffect(() => {
    if (tab === 'disposal') fetchDisposals()
  }, [disposalStatus])

  const getCoverageColor = (pct?: number) => {
    if (pct == null) return 'text-gray-400'
    if (pct >= 80) return 'text-green-600 font-semibold'
    if (pct >= 50) return 'text-yellow-600 font-semibold'
    return 'text-red-600 font-semibold'
  }

  const auditTotalPages = Math.ceil(auditTotal / limit)

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ShieldCheck className="h-5 w-5 text-muted-foreground" />
          <h1 className="text-2xl font-semibold">Compliance & Audit</h1>
        </div>
        <Button variant="outline" onClick={() => downloadFile('/export/compliance.pdf', `compliance_report_${new Date().toISOString().slice(0,10)}.pdf`)}>
          <FileDown className="h-4 w-4 mr-1" /> Export PDF
        </Button>
      </div>

      <Tabs value={tab} onValueChange={t => { setTab(t) }}>
        <TabsList>
          <TabsTrigger value="frameworks"><ShieldCheck className="h-4 w-4 mr-1" />Framework Coverage</TabsTrigger>
          <TabsTrigger value="disposal"><Trash2 className="h-4 w-4 mr-1" />Disposal Compliance</TabsTrigger>
          <TabsTrigger value="audit"><FileText className="h-4 w-4 mr-1" />Audit Trail</TabsTrigger>
        </TabsList>

        {/* ── Framework Coverage ── */}
        <TabsContent value="frameworks" className="mt-4">
          <div className="flex justify-end mb-2">
            <Button variant="outline" size="sm" onClick={fetchFrameworks}><RefreshCw className="h-4 w-4 mr-1" />Refresh</Button>
          </div>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Framework</TableHead>
                  <TableHead>Kode</TableHead>
                  <TableHead className="text-right">Total Controls</TableHead>
                  <TableHead className="text-right">Covered</TableHead>
                  <TableHead className="text-right">Coverage %</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : frameworks.length === 0 ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
                ) : frameworks.map(f => (
                  <TableRow key={f.id}>
                    <TableCell className="font-medium">{f.name}</TableCell>
                    <TableCell><Badge variant="outline">{f.code}</Badge></TableCell>
                    <TableCell className="text-right">{f.total_controls}</TableCell>
                    <TableCell className="text-right">{f.covered_controls}</TableCell>
                    <TableCell className={`text-right ${getCoverageColor(f.coverage_pct)}`}>
                      {f.coverage_pct != null ? `${f.coverage_pct}%` : '-'}
                    </TableCell>
                    <TableCell>
                      {f.coverage_pct == null ? <Badge variant="secondary">N/A</Badge>
                        : f.coverage_pct >= 80 ? <Badge className="bg-green-100 text-green-700">Good</Badge>
                        : f.coverage_pct >= 50 ? <Badge className="bg-yellow-100 text-yellow-700">Partial</Badge>
                        : <Badge variant="destructive">Low</Badge>}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        {/* ── Disposal Compliance ── */}
        <TabsContent value="disposal" className="mt-4">
          <div className="flex items-center gap-2 mb-2">
            <Select value={disposalStatus} onValueChange={setDisposalStatus}>
              <SelectTrigger className="w-52"><SelectValue placeholder="Status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua Status</SelectItem>
                <SelectItem value="compliant">Compliant</SelectItem>
                <SelectItem value="data_wipe_pending">Data Wipe Pending</SelectItem>
                <SelectItem value="env_non_compliant">Env Non-Compliant</SelectItem>
                <SelectItem value="missing_record">Missing Record</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" size="icon" onClick={fetchDisposals}><RefreshCw className="h-4 w-4" /></Button>
          </div>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Asset</TableHead>
                  <TableHead>Tag</TableHead>
                  <TableHead>Metode Disposal</TableHead>
                  <TableHead>Data Wipe</TableHead>
                  <TableHead>Env OK</TableHead>
                  <TableHead>Sertifikat</TableHead>
                  <TableHead>Tanggal</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : disposals.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-8 text-muted-foreground">Tidak ada data disposal</TableCell></TableRow>
                ) : disposals.map((d, i) => (
                  <TableRow key={i}>
                    <TableCell className="font-medium">{d.asset_name}</TableCell>
                    <TableCell className="font-mono text-sm">{d.asset_tag}</TableCell>
                    <TableCell>{d.disposal_method ?? '-'}</TableCell>
                    <TableCell>{d.data_wipe_completed == null ? '-' : d.data_wipe_completed ? '✅' : '❌'}</TableCell>
                    <TableCell>{d.environmental_compliant == null ? '-' : d.environmental_compliant ? '✅' : '❌'}</TableCell>
                    <TableCell className="font-mono text-xs">{d.certificate_number ?? '-'}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{d.date_disposed ? new Date(d.date_disposed).toLocaleDateString('id-ID') : '-'}</TableCell>
                    <TableCell>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${DISPOSAL_STATUS_COLORS[d.compliance_status] ?? 'bg-gray-100 text-gray-600'}`}>
                        {d.compliance_status?.replace(/_/g, ' ')}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        {/* ── Audit Trail ── */}
        <TabsContent value="audit" className="mt-4">
          <div className="flex justify-end mb-2">
            <Button variant="outline" size="sm" onClick={() => fetchAuditLogs(auditPage)}><RefreshCw className="h-4 w-4 mr-1" />Refresh</Button>
          </div>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Entitas</TableHead>
                  <TableHead>Aksi</TableHead>
                  <TableHead>Waktu</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={4} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : auditLogs.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="text-center py-8 text-muted-foreground">Tidak ada data (perlu role super_admin)</TableCell></TableRow>
                ) : auditLogs.map(log => (
                  <TableRow key={log.id}>
                    <TableCell className="font-mono text-sm">{log.id}</TableCell>
                    <TableCell><Badge variant="outline">{log.entity_type} #{log.entity_id}</Badge></TableCell>
                    <TableCell>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${ACTION_COLORS[log.action] ?? 'bg-gray-100 text-gray-600'}`}>{log.action}</span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{new Date(log.changed_at).toLocaleString('id-ID')}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {auditTotalPages > 1 && (
            <div className="flex items-center justify-between mt-4">
              <span className="text-sm text-muted-foreground">Halaman {auditPage} dari {auditTotalPages}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" disabled={auditPage <= 1} onClick={() => setAuditPage(p => p - 1)}>Sebelumnya</Button>
                <Button variant="outline" size="sm" disabled={auditPage >= auditTotalPages} onClick={() => setAuditPage(p => p + 1)}>Selanjutnya</Button>
              </div>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}
