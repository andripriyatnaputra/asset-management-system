import { useEffect, useMemo, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import {
  Card, CardContent, CardHeader, CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  PieChart, Pie, Cell, Tooltip, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, LineChart, Line,
} from "recharts"

type AssetRow = {
  id: number
  name: string
  asset_tag: string
  department_name?: string | null
  owner_department_name?: string | null
  compliance_flag?: boolean | null
  compliance_note?: string | null
  updated_at?: string | null
}

const STATUS_COLORS = {
  Compliant: "bg-green-100 text-green-700",
  "Non-Compliant": "bg-red-100 text-red-700",
  Pending: "bg-yellow-100 text-yellow-700",
} as const

export default function ComplianceDashboardPage() {
  const [rows, setRows] = useState<AssetRow[]>([])
  const [summary, setSummary] = useState<any>({})
  const [loading, setLoading] = useState(false)
  const [q, setQ] = useState("")
  const [status, setStatus] = useState("ALL")
  const [insight, setInsight] = useState<string>("")

  // 🔹 Ambil data compliance summary dari backend
  async function fetchAssets() {
    try {
      setLoading(true)
      // backend sudah di‐prefix "/api/v1" lewat VITE_API_BASE_URL
      const { data } = await apiClient.get("/assets/compliance-summary")

      const list = data?.data ?? []
      const summary = data?.summary ?? {}

      // --- Aggregasi by Department ---
      const deptAgg: Record<string, { compliant: number; non_compliant: number }> = {}
      list.forEach((a: any) => {
        const dept = a.owner_department_name || a.department_name || "Unassigned"
        if (!deptAgg[dept]) deptAgg[dept] = { compliant: 0, non_compliant: 0 }
        if (a.compliance_flag === true) deptAgg[dept].compliant++
        else if (a.compliance_flag === false) deptAgg[dept].non_compliant++
      })
      const byDepartment = Object.entries(deptAgg).map(([name, v]) => ({
        name,
        compliant: v.compliant,
        non_compliant: v.non_compliant,
      }))

      // --- Aggregasi tren by tanggal ---
      const trendAgg: Record<string, { compliant: number; non_compliant: number }> = {}
      list.forEach((a: any) => {
        const dateKey = a.updated_at ? a.updated_at.split("T")[0] : "Unknown"
        if (!trendAgg[dateKey]) trendAgg[dateKey] = { compliant: 0, non_compliant: 0 }
        if (a.compliance_flag === true) trendAgg[dateKey].compliant++
        else if (a.compliance_flag === false) trendAgg[dateKey].non_compliant++
      })
      const trend = Object.entries(trendAgg)
        .map(([date, v]) => ({ date, ...v }))
        .sort((a, b) => a.date.localeCompare(b.date))

      setRows(list)
      setSummary({ ...summary, byDepartment, trend })
    } catch {
      toast.error("Gagal memuat data compliance")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAssets()
  }, [])

  // --- Mapping status ---
  const mappedRows = useMemo(() => {
    return rows.map((it) => ({
      ...it,
      status: it.compliance_flag
        ? "Compliant"
        : it.compliance_flag === false
        ? "Non-Compliant"
        : "Pending",
    }))
  }, [rows])

  // --- Filter pencarian & status ---
  const filteredRows = useMemo(() => {
    return mappedRows.filter((r) => {
      const matchQ = q
        ? (r.name ?? "").toLowerCase().includes(q.toLowerCase()) ||
          (r.asset_tag ?? "").toLowerCase().includes(q.toLowerCase()) ||
          (r.department_name ?? "").toLowerCase().includes(q.toLowerCase())
        : true
      const matchStatus = status === "ALL" ? true : r.status === status
      return matchQ && matchStatus
    })
  }, [mappedRows, q, status])

  // --- Hitung summary cepat ---
  const quickSummary = useMemo(() => {
    const total = rows.length
    const compliant = rows.filter((r) => r.compliance_flag === true).length
    const non_compliant = rows.filter((r) => r.compliance_flag === false).length
    const pending = rows.filter((r) => r.compliance_flag == null).length
    return { total, compliant, non_compliant, pending }
  }, [rows])

  // --- Verifikasi compliance ---
  async function handleVerify(assetId: number) {
    try {
      await apiClient.post(`/assets/${assetId}/verify-compliance`)
      toast.success("Verifikasi compliance berhasil")
      fetchAssets()
    } catch {
      toast.error("Gagal memverifikasi compliance")
    }
  }

  // --- Export CSV ---
  async function handleExport() {
    try {
      toast.info("Menyiapkan file laporan...")
      const res = await apiClient.get("/assets/compliance-export", {
        responseType: "blob",
        headers: { Accept: "text/csv" },
      })

      const blob = new Blob([res.data], { type: "text/csv;charset=utf-8" })
      const url = window.URL.createObjectURL(blob)

      const a = document.createElement("a")
      a.href = url
      a.download = `compliance_report_${new Date().toISOString().split("T")[0]}.csv`
      document.body.appendChild(a)
      a.click()
      a.remove()
      window.URL.revokeObjectURL(url)

      toast.success("Export CSV berhasil")
    } catch (err: any) {
      console.error("Export error:", err)
      toast.error("Gagal mengekspor CSV")
    }
  }

  // --- Pie chart data ---
  const pieData = [
    { name: "Compliant", value: quickSummary.compliant },
    { name: "Non-Compliant", value: quickSummary.non_compliant },
    { name: "Pending", value: quickSummary.pending },
  ]
  const COLORS = ["#16a34a", "#dc2626", "#f59e0b"]

  // --- Predictive Compliance Insight ---
  useEffect(() => {
    if (!summary?.trend) return
    const trend = summary.trend
    if (trend.length < 3) {
      setInsight("Belum cukup data untuk analisis tren.")
      return
    }
    const last3 = trend.slice(-3)
    const avgPrev = (last3[0].compliant + last3[1].compliant) / 2
    const delta = last3[2].compliant - avgPrev
    const rate = (delta / (avgPrev || 1)) * 100
    if (rate < -10) {
      setInsight(`🚨 Tingkat kepatuhan menurun ${rate.toFixed(1)}% dalam 3 hari terakhir.`)
    } else if (rate > 10) {
      setInsight(`✅ Kepatuhan meningkat ${rate.toFixed(1)}% dalam 3 hari terakhir.`)
    } else {
      setInsight(`ℹ️ Tingkat kepatuhan relatif stabil minggu ini.`)
    }
  }, [summary])

  // --- Render UI ---
  return (
    <div className="space-y-6 py-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Compliance Dashboard</h1>
          <p className="text-muted-foreground">
            Ringkasan status kepatuhan aset dan linkage governance
          </p>
        </div>
        <Button variant="outline" onClick={handleExport}>
          Export CSV
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          ["🟢 Compliant", quickSummary.compliant],
          ["🔴 Non-Compliant", quickSummary.non_compliant],
          ["⏳ Pending", quickSummary.pending],
          ["📦 Total Assets", quickSummary.total],
        ].map(([label, val], i) => (
          <Card key={i}>
            <CardHeader>
              <CardTitle>{label}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{val ?? "-"}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Predictive Insight */}
      {insight && (
        <Card className="border-l-4 border-blue-500 bg-blue-50">
          <CardContent className="py-3">
            <p className="text-sm text-blue-800 font-medium">{insight}</p>
          </CardContent>
        </Card>
      )}

      {/* Chart Section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Pie Chart */}
        <Card className="col-span-1">
          <CardHeader><CardTitle>Distribusi Status</CardTitle></CardHeader>
          <CardContent className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={pieData} dataKey="value" nameKey="name" label>
                  {pieData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Compliance by Department */}
        <Card className="col-span-1 lg:col-span-2">
          <CardHeader><CardTitle>Compliance by Department</CardTitle></CardHeader>
          <CardContent className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={summary?.byDepartment ?? []}>
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="compliant" fill="#16a34a" name="Compliant" />
                <Bar dataKey="non_compliant" fill="#dc2626" name="Non-Compliant" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Compliance Trend */}
        <Card className="col-span-1 lg:col-span-3">
          <CardHeader><CardTitle>Compliance Trend Over Time</CardTitle></CardHeader>
          <CardContent className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={summary?.trend ?? []}>
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip />
                <Line type="monotone" dataKey="compliant" stroke="#16a34a" />
                <Line type="monotone" dataKey="non_compliant" stroke="#dc2626" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* Table Section */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col md:flex-row gap-2 md:items-center md:justify-between">
            <div className="flex gap-2">
              <Input
                placeholder="Cari aset (nama/tag/department)"
                value={q}
                onChange={(e) => setQ(e.target.value)}
              />
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger className="w-48"><SelectValue placeholder="Status" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">All</SelectItem>
                  <SelectItem value="Compliant">Compliant</SelectItem>
                  <SelectItem value="Non-Compliant">Non-Compliant</SelectItem>
                  <SelectItem value="Pending">Pending</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="text-sm text-muted-foreground">
              {filteredRows.length} items
            </div>
          </div>

          <div className="mt-4 border rounded-md">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Asset</TableHead>
                  <TableHead>Tag</TableHead>
                  <TableHead>Department</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Note</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading && (
                  <TableRow><TableCell colSpan={7}>Loading...</TableCell></TableRow>
                )}
                {!loading &&
                  filteredRows.map((r) => (
                    <TableRow key={r.id}>
                      <TableCell>{r.name}</TableCell>
                      <TableCell>{r.asset_tag}</TableCell>
                      <TableCell>{r.department_name ?? "-"}</TableCell>
                      <TableCell>
                        <span
                          className={`px-2 py-1 rounded text-xs ${
                            (STATUS_COLORS as any)[r.status] ?? ""
                          }`}
                        >
                          {r.status}
                        </span>
                      </TableCell>
                      <TableCell className="max-w-[200px] truncate text-sm text-muted-foreground">
                        {r.compliance_note ?? "-"}
                      </TableCell>
                      <TableCell>
                        {r.updated_at
                          ? new Date(r.updated_at).toLocaleString("id-ID")
                          : "-"}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button size="sm" variant="secondary" onClick={() => handleVerify(r.id)}>
                          Verify
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                {!loading && filteredRows.length === 0 && (
                  <TableRow><TableCell colSpan={7}>Tidak ada data</TableCell></TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
