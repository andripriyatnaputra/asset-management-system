import { useEffect, useState, useCallback } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { RefreshCw, Plug, BarChart3 } from 'lucide-react'

interface VendorPerformance {
  id: number
  vendor_name: string
  service_name: string
  period: string
  sla_target: number
  sla_achieved: number
  incidents_reported: number
  created_at: string
}

interface ServiceAvailability {
  id: number
  service_code: string
  service_name: string
  period: string
  total_minutes: number
  downtime_minutes: number
  availability_pct: number
  created_at: string
}

export default function IntegrationsPage() {
  const [tab, setTab] = useState('vendor')

  const [vendors, setVendors] = useState<VendorPerformance[]>([])
  const [vendorTotal, setVendorTotal] = useState(0)
  const [vendorPage, setVendorPage] = useState(1)

  const [availability, setAvailability] = useState<ServiceAvailability[]>([])
  const [availTotal, setAvailTotal] = useState(0)
  const [availPage, setAvailPage] = useState(1)

  const [isLoading, setIsLoading] = useState(false)
  const [serviceCode, setServiceCode] = useState('')
  const [period, setPeriod] = useState('')
  const limit = 20

  const fetchVendors = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (period.trim()) params.set('period', period.trim())
    apiClient.get(`/vendors/performance?${params}`)
      .then(res => { setVendors(res.data.data ?? []); setVendorTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat data vendor performance'))
      .finally(() => setIsLoading(false))
  }, [period])

  const fetchAvailability = useCallback((p: number) => {
    setIsLoading(true)
    const params = new URLSearchParams({ page: String(p), limit: String(limit) })
    if (serviceCode.trim()) params.set('service_code', serviceCode.trim())
    apiClient.get(`/services/availability?${params}`)
      .then(res => { setAvailability(res.data.data ?? []); setAvailTotal(res.data.total ?? 0) })
      .catch(() => toast.error('Gagal memuat data availability'))
      .finally(() => setIsLoading(false))
  }, [serviceCode])

  useEffect(() => {
    if (tab === 'vendor') fetchVendors(vendorPage)
    else fetchAvailability(availPage)
  }, [tab, vendorPage, availPage, fetchVendors, fetchAvailability])

  useEffect(() => {
    setVendorPage(1); setAvailPage(1)
    if (tab === 'vendor') fetchVendors(1)
    else fetchAvailability(1)
  }, [period, serviceCode])

  const getSlaColor = (achieved: number, target: number) => {
    const ratio = achieved / target
    if (ratio >= 1) return 'text-green-600 font-semibold'
    if (ratio >= 0.95) return 'text-yellow-600 font-medium'
    return 'text-red-600 font-semibold'
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
      <div className="flex items-center gap-2">
        <Plug className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-2xl font-semibold">Integrasi & Monitoring</h1>
      </div>

      <Tabs value={tab} onValueChange={t => { setTab(t); setServiceCode(''); setPeriod('') }}>
        <TabsList>
          <TabsTrigger value="vendor"><BarChart3 className="h-4 w-4 mr-1" />Vendor Performance <Badge variant="secondary" className="ml-1">{vendorTotal}</Badge></TabsTrigger>
          <TabsTrigger value="availability"><Plug className="h-4 w-4 mr-1" />Service Availability <Badge variant="secondary" className="ml-1">{availTotal}</Badge></TabsTrigger>
        </TabsList>

        <div className="flex flex-wrap gap-2 mt-4">
          {tab === 'vendor' ? (
            <Input placeholder="Filter periode (e.g. 2024-01)..." value={period} onChange={e => setPeriod(e.target.value)} className="w-64" />
          ) : (
            <Input placeholder="Filter service code..." value={serviceCode} onChange={e => setServiceCode(e.target.value)} className="w-64" />
          )}
          <Button variant="outline" size="icon" onClick={() => tab === 'vendor' ? fetchVendors(vendorPage) : fetchAvailability(availPage)}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>

        <TabsContent value="vendor" className="mt-4">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Vendor</TableHead>
                  <TableHead>Layanan</TableHead>
                  <TableHead>Periode</TableHead>
                  <TableHead>Target SLA (%)</TableHead>
                  <TableHead>Pencapaian SLA (%)</TableHead>
                  <TableHead>Insiden</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : vendors.length === 0 ? (
                  <TableRow><TableCell colSpan={6} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
                ) : vendors.map(v => (
                  <TableRow key={v.id}>
                    <TableCell className="font-medium">{v.vendor_name}</TableCell>
                    <TableCell>{v.service_name}</TableCell>
                    <TableCell className="font-mono text-sm">{v.period}</TableCell>
                    <TableCell>{v.sla_target}%</TableCell>
                    <TableCell className={getSlaColor(v.sla_achieved, v.sla_target)}>{v.sla_achieved}%</TableCell>
                    <TableCell>
                      <Badge variant={v.incidents_reported > 5 ? 'destructive' : v.incidents_reported > 0 ? 'secondary' : 'default'}>
                        {v.incidents_reported}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {renderPagination(vendorPage, vendorTotal, setVendorPage)}
        </TabsContent>

        <TabsContent value="availability" className="mt-4">
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kode Layanan</TableHead>
                  <TableHead>Nama Layanan</TableHead>
                  <TableHead>Periode</TableHead>
                  <TableHead>Downtime (mnt)</TableHead>
                  <TableHead>Availability (%)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={5} className="text-center py-8 text-muted-foreground">Memuat...</TableCell></TableRow>
                ) : availability.length === 0 ? (
                  <TableRow><TableCell colSpan={5} className="text-center py-8 text-muted-foreground">Tidak ada data</TableCell></TableRow>
                ) : availability.map(a => (
                  <TableRow key={a.id}>
                    <TableCell className="font-mono text-sm">{a.service_code}</TableCell>
                    <TableCell className="font-medium">{a.service_name}</TableCell>
                    <TableCell className="font-mono text-sm">{a.period}</TableCell>
                    <TableCell>{a.downtime_minutes}</TableCell>
                    <TableCell className={getSlaColor(a.availability_pct, 99.9)}>{a.availability_pct.toFixed(3)}%</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {renderPagination(availPage, availTotal, setAvailPage)}
        </TabsContent>
      </Tabs>
    </div>
  )
}
