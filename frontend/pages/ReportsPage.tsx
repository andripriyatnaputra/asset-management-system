import { useState, useEffect } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'

import type { Department } from '@/types'
import type { Employee } from '@/types'

interface AssetReportRow {
  asset_name: string
  asset_tag: string
  asset_type: string
  employee_name: string
  employee_nik: string
  assigned_at: string | null
}

interface TicketReportRow {
  asset_type: string
  ticket_count: number
}

export default function ReportsPage() {
  const [departments, setDepartments] = useState<Department[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])

  const [selectedDept, setSelectedDept] = useState<string>('')
  const [deptReportData, setDeptReportData] = useState<AssetReportRow[]>([])
  const [isDeptLoading, setIsDeptLoading] = useState(false)

  const [selectedEmployee, setSelectedEmployee] = useState<string>('')
  const [employeeReportData, setEmployeeReportData] = useState<AssetReportRow[]>([])
  const [isEmployeeLoading, setIsEmployeeLoading] = useState(false)

  const [ticketReport, setTicketReport] = useState<TicketReportRow[]>([])
  const [isTicketLoading, setIsTicketLoading] = useState(true)

  // ============================================================
  // SAFE PAYLOAD HELPER (fallback untuk berbagai bentuk response)
  // ============================================================
  function toArray<T>(resData: unknown): T[] {
    if (!resData) return []

    // if array straight
    if (Array.isArray(resData)) return resData

    // if object containing { data: [...] }
    if (
      typeof resData === "object" &&
      resData !== null &&
      "data" in resData &&
      Array.isArray((resData as any).data)
    ) {
      return (resData as any).data
    }

    return []
  }


  // ============================================================
  // Bootstrap: load dropdown + load tiket chart + interval refresh
  // ============================================================
  useEffect(() => {
    const loadDropdowns = async () => {
      try {
        const [deptRes, empRes] = await Promise.all([
          apiClient.get('/departments'),
          apiClient.get('/employees?limit=500'),
        ])
        setDepartments(toArray<Department>(deptRes.data))
        setEmployees(toArray<Employee>(empRes.data))
      } catch {
        toast.error('Gagal memuat data referensi.')
      }
    }

    const loadTicketReport = async () => {
      try {
        const res = await apiClient.get('/reports/tickets-by-asset-type')
        setTicketReport(toArray<TicketReportRow>(res.data))
      } catch {
        toast.error('Gagal memuat laporan tiket.')
      } finally {
        setIsTicketLoading(false)
      }
    }

    loadDropdowns()
    loadTicketReport()

    // Auto-refresh setiap 30 detik
    const interval = setInterval(() => {
      apiClient.get('/reports/tickets-by-asset-type')
        .then(res => setTicketReport(toArray<TicketReportRow>(res.data)))
        .catch(() => {})
    }, 30000)

    return () => clearInterval(interval)
  }, [])

  // ============================================================
  // Generate laporan departemen
  // ============================================================
  const handleGenerateDeptReport = async () => {
    if (!selectedDept) {
      toast.error('Silakan pilih departemen terlebih dahulu.')
      return
    }
    try {
      setIsDeptLoading(true)
      const res = await apiClient.get(`/reports/assets-by-department?department_id=${selectedDept}`)
      const rows = toArray<AssetReportRow>(res.data)
      setDeptReportData(rows)

      if (rows.length === 0) {
        toast.info('Tidak ada data aset untuk departemen ini.')
      } else {
        toast.success('Laporan per departemen berhasil dibuat!')
      }
    } catch {
      toast.error('Gagal membuat laporan per departemen.')
    } finally {
      setIsDeptLoading(false)
    }
  }

  // ============================================================
  // Generate laporan karyawan
  // ============================================================
  const handleGenerateEmployeeReport = async () => {
    if (!selectedEmployee) {
      toast.error('Silakan pilih karyawan terlebih dahulu.')
      return
    }
    try {
      setIsEmployeeLoading(true)
      const res = await apiClient.get(`/reports/assets-by-employee?employee_id=${selectedEmployee}`)
      const rows = toArray<AssetReportRow>(res.data)
      setEmployeeReportData(rows)

      if (rows.length === 0) {
        toast.info('Tidak ada data aset untuk karyawan ini.')
      } else {
        toast.success('Laporan per karyawan berhasil dibuat!')
      }
    } catch {
      toast.error('Gagal membuat laporan per karyawan.')
    } finally {
      setIsEmployeeLoading(false)
    }
  }

  // ============================================================
  // Ekspor CSV laporan departemen
  // ============================================================
  const handleExportCSV = async () => {
    if (!selectedDept) {
      toast.error('Silakan pilih departemen terlebih dahulu.')
      return
    }

    if (deptReportData.length === 0) {
      toast.error('Tidak bisa ekspor CSV karena data laporan kosong.')
      return
    }

    const promise = apiClient.get(
      `/reports/assets-by-department?department_id=${selectedDept}&export=csv`,
      { responseType: 'blob' }
    )

    toast.promise(promise, {
      loading: 'Mengekspor CSV...',
      success: (response: { data: Blob }) => {
        const blob =
          response?.data instanceof Blob
            ? response.data
            : new Blob([response?.data ?? ''])

        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.setAttribute('download', `report-aset-departemen-${selectedDept}.csv`)
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)

        return 'Ekspor berhasil!'
      },
      error: 'Gagal mengekspor CSV.',
    })
  }


  return (
    <div className="container mx-auto space-y-6 py-8">
      <h1 className="text-3xl font-bold">Pelaporan</h1>

      {/* Laporan per Departemen */}
      <Card>
        <CardHeader>
          <CardTitle>Laporan Aset per Departemen</CardTitle>
          <CardDescription>Pilih departemen untuk melihat daftar aset aktif dan ekspor ke CSV.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex items-center gap-2">
            <Select value={selectedDept} onValueChange={setSelectedDept}>
              <SelectTrigger className="w-[280px]">
                <SelectValue placeholder="Pilih Departemen..." />
              </SelectTrigger>
              <SelectContent>
                {departments.map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button onClick={handleGenerateDeptReport} disabled={isDeptLoading}>
              {isDeptLoading ? 'Memuat...' : 'Tampilkan Laporan'}
            </Button>
            {deptReportData.length > 0 && (
              <Button variant="outline" onClick={handleExportCSV}>
                Ekspor ke CSV
              </Button>
            )}
          </div>

          {isDeptLoading && <p className="text-sm text-muted-foreground">Memuat data…</p>}

          {!isDeptLoading && deptReportData.length > 0 && (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nama Karyawan</TableHead>
                    <TableHead>NIK</TableHead>
                    <TableHead>Nama Aset</TableHead>
                    <TableHead>Tag Aset</TableHead>
                    <TableHead>Tipe Aset</TableHead>
                    <TableHead>Tanggal Diberikan</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deptReportData.map((row, i) => (
                    <TableRow key={i}>
                      <TableCell>{row.employee_name}</TableCell>
                      <TableCell>{row.employee_nik}</TableCell>
                      <TableCell className="font-medium">{row.asset_name}</TableCell>
                      <TableCell className="font-mono">{row.asset_tag}</TableCell>
                      <TableCell>{row.asset_type}</TableCell>
                      <TableCell>
                        {row.assigned_at
                          ? new Date(row.assigned_at).toLocaleDateString('id-ID')
                          : '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Laporan per Karyawan */}
      <Card>
        <CardHeader>
          <CardTitle>Laporan Aset per Karyawan</CardTitle>
          <CardDescription>Pilih karyawan untuk melihat daftar aset yang dipegang.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="mb-4 flex items-center gap-2">
            <Select value={selectedEmployee} onValueChange={setSelectedEmployee}>
              <SelectTrigger className="w-[280px]">
                <SelectValue placeholder="Pilih Karyawan..." />
              </SelectTrigger>
              <SelectContent>
                {employees.map((emp) => (
                  <SelectItem key={emp.id} value={String(emp.id)}>
                    {emp.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button onClick={handleGenerateEmployeeReport} disabled={isEmployeeLoading}>
              {isEmployeeLoading ? 'Memuat...' : 'Tampilkan Laporan'}
            </Button>
          </div>

          {isEmployeeLoading && <p className="text-sm text-muted-foreground">Memuat data…</p>}

          {!isEmployeeLoading && employeeReportData.length > 0 && (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nama Aset</TableHead>
                    <TableHead>Tag Aset</TableHead>
                    <TableHead>Tipe Aset</TableHead>
                    <TableHead>Tanggal Diberikan</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {employeeReportData.map((row, i) => (
                    <TableRow key={i}>
                      <TableCell className="font-medium">{row.asset_name}</TableCell>
                      <TableCell className="font-mono">{row.asset_tag}</TableCell>
                      <TableCell>{row.asset_type}</TableCell>
                      <TableCell>
                        {row.assigned_at
                          ? new Date(row.assigned_at).toLocaleDateString('id-ID')
                          : '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Laporan Tiket per Tipe Aset */}
      <Card>
        <CardHeader>
          <CardTitle>Laporan Tiket per Tipe Aset</CardTitle>
          <CardDescription>Jumlah tiket berdasarkan tipe aset.</CardDescription>
        </CardHeader>
        <CardContent className="h-[350px]">
          {isTicketLoading ? (
            <div className="h-full w-full animate-pulse rounded bg-muted" />
          ) : ticketReport.length === 0 ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              Tidak ada data tiket.
            </div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={ticketReport} layout="vertical">
                <XAxis type="number" />
                <YAxis type="category" dataKey="asset_type" width={120} />
                <Tooltip cursor={{ fill: 'rgba(241,245,249,0.5)' }} />
                {/* Pakai warna chart pertama agar konsisten dengan tema */}
                <Bar dataKey="ticket_count" fill="hsl(var(--chart-1))" barSize={30} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
