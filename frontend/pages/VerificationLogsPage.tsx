import { useEffect, useState, useMemo } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import {
  Card, CardContent, CardHeader, CardTitle, CardDescription,
} from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

type AuditLog = {
  id: number
  actor_name?: string
  entity_name?: string
  entity_id?: number
  action?: string
  changes?: any
  created_at?: string
  ip_address?: string
  request_path?: string
}

export default function VerificationLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState("")
  const [dateFrom, setDateFrom] = useState("")
  const [dateTo, setDateTo] = useState("")

  const filtered = useMemo(() => {
    return logs.filter((l) => {
      const matchSearch = search
        ? (l.actor_name ?? "").toLowerCase().includes(search.toLowerCase()) ||
          (l.entity_name ?? "").toLowerCase().includes(search.toLowerCase())
        : true
      return matchSearch
    })
  }, [logs, search])

  async function fetchLogs() {
    try {
      setLoading(true)
      const params = new URLSearchParams()
      if (dateFrom) params.append("from", dateFrom)
      if (dateTo) params.append("to", dateTo)
      const { data } = await apiClient.get(`/audit-logs?${params.toString()}`)
      setLogs(Array.isArray(data) ? data : data?.audit_logs ?? [])
    } catch (err: any) {
      console.error(err)
      toast.error("Gagal memuat data log verifikasi")
    } finally {
      setLoading(false)
    }
  }

  async function handleExport() {
    try {
      const res = await apiClient.get("/api/v1/audit-logs/export", { responseType: "blob" })
      const blob = new Blob([res.data], { type: "text/csv;charset=utf-8" })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = "verification_logs.csv"
      a.click()
      window.URL.revokeObjectURL(url)
    } catch {
      toast.error("Gagal mengekspor log")
    }
  }

  useEffect(() => {
    fetchLogs()
  }, [])

  return (
    <div className="space-y-6 py-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Verification Logs</h1>
          <p className="text-muted-foreground">
            Riwayat aktivitas verifikasi compliance aset
          </p>
        </div>
        <Button variant="outline" onClick={handleExport}>
          Export CSV
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6 space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div>
              <Label>Pencarian</Label>
              <Input
                placeholder="Cari nama aset / aktor"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div>
              <Label>Dari tanggal</Label>
              <Input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
              />
            </div>
            <div>
              <Label>Sampai tanggal</Label>
              <Input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
              />
            </div>
          </div>
          <div className="flex justify-end">
            <Button onClick={fetchLogs}>Terapkan Filter</Button>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <CardTitle>Riwayat Verifikasi</CardTitle>
          <CardDescription>Daftar semua tindakan compliance verification</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Memuat data...</p>
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>Aktor</TableHead>
                    <TableHead>Entity</TableHead>
                    <TableHead>Aksi</TableHead>
                    <TableHead>Tanggal</TableHead>
                    <TableHead>IP</TableHead>
                    <TableHead>Path</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filtered.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7}>Tidak ada data</TableCell>
                    </TableRow>
                  ) : (
                    filtered.map((l) => (
                      <TableRow key={l.id}>
                        <TableCell>{l.id}</TableCell>
                        <TableCell>{l.actor_name ?? "-"}</TableCell>
                        <TableCell>
                          {l.entity_name} #{l.entity_id}
                        </TableCell>
                        <TableCell>{l.action}</TableCell>
                        <TableCell>
                          {l.created_at
                            ? new Date(l.created_at).toLocaleString("id-ID")
                            : "-"}
                        </TableCell>
                        <TableCell>{l.ip_address ?? "-"}</TableCell>
                        <TableCell className="text-xs">{l.request_path}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
