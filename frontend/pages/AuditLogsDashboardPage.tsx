import { useEffect, useState, useMemo } from "react"
import apiClient from "@/services/api"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { RefreshCw } from "lucide-react"

interface AuditLog {
  id: number
  actor_id: number
  entity_name: string
  entity_id: number
  action: string
  changes: string
  ip_address: string
  user_agent: string
  request_path: string
  created_at: string
}

export default function AuditLogsDashboardPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [search, setSearch] = useState("")
  const [isLoading, setIsLoading] = useState(true)

  const fetchLogs = async () => {
    try {
      setIsLoading(true)
      const res = await apiClient.get("/dashboard/audit-logs")
      setLogs(res.data.logs || [])
    } catch {
      toast.error("Gagal memuat data audit logs.")
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs()
  }, [])

  // 🔍 Filter client-side
  const filtered = useMemo(() => {
    if (!search.trim()) return logs
    return logs.filter((l) =>
      [l.entity_name, l.action, l.request_path]
        .join(" ")
        .toLowerCase()
        .includes(search.toLowerCase())
    )
  }, [logs, search])

  const renderBadge = (action: string) => {
    const a = action.toUpperCase()
    if (a.includes("DELETE")) return <Badge variant="destructive">DELETE</Badge>
    if (a.includes("CREATE")) return <Badge className="bg-green-100 text-green-700">CREATE</Badge>
    if (a.includes("UPDATE")) return <Badge variant="secondary">UPDATE</Badge>
    if (a.includes("ALERT")) return <Badge className="bg-red-100 text-red-700">ALERT</Badge>
    if (a.includes("LOGIN")) return <Badge className="bg-blue-100 text-blue-700">LOGIN</Badge>
    return <Badge variant="outline">{a}</Badge>
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Audit & Alert Dashboard</h1>
        <div className="flex items-center gap-2">
          <Input
            placeholder="Cari entity / action / path..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-64"
          />
          <Button onClick={fetchLogs} variant="outline" disabled={isLoading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      <Card className="shadow-sm">
        <CardHeader>
          <CardTitle>200 Aktivitas Terakhir</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          {isLoading ? (
            <div className="h-32 w-full animate-pulse rounded bg-muted" />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-14">ID</TableHead>
                  <TableHead>Entity</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Actor</TableHead>
                  <TableHead>Path</TableHead>
                  <TableHead>Timestamp</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.length ? (
                  filtered.map((l) => (
                    <TableRow key={l.id}>
                      <TableCell>{l.id}</TableCell>
                      <TableCell>{l.entity_name}</TableCell>
                      <TableCell>{renderBadge(l.action)}</TableCell>
                      <TableCell>{l.actor_id === 0 ? "System" : l.actor_id}</TableCell>
                      <TableCell className="max-w-[280px] truncate">{l.request_path}</TableCell>
                      <TableCell>
                        {new Date(l.created_at).toLocaleString("id-ID")}
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground py-6">
                      Tidak ada data audit yang cocok.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
