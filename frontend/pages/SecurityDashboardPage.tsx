// File: src/pages/SecurityDashboardPage.tsx
import { useEffect, useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Download, Activity, ShieldCheck } from "lucide-react"
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts"
import { toast } from "sonner"
import { fetchSecurityAudits, fetchSecurityAuditMeta } from "@/services/securityService"

export default function SecurityDashboardPage() {
  const [data, setData] = useState<any[]>([])
  const [daily, setDaily] = useState<any[]>([])
  const [filters, setFilters] = useState({ user_id: "", action: "", start_date: "", end_date: "" })
  const [meta, setMeta] = useState<{ actors: {id:number; name:string}[]; actions: string[] }>({ actors: [], actions: [] })

  useEffect(() => {
    fetchSecurityAuditMeta()
      .then(setMeta)
      .catch(() => toast.error("Gagal memuat metadata audit."))
  }, [])

  const computeDaily = (logs: any[]) => {
    const grouped: Record<string, number> = {}
    logs.forEach((l) => {
      const d = l.created_at ? new Date(l.created_at).toISOString().slice(0,10) : "-"
      grouped[d] = (grouped[d] || 0) + 1
    })
    setDaily(Object.entries(grouped).map(([date, count]) => ({ date, count })))
  }

  const handleFilter = async () => {
    try {
      const logs = await fetchSecurityAudits(filters)
      setData(logs)
      computeDaily(logs)
    } catch {
      toast.error("Gagal memuat filter audit.")
    }
  }

  const exportCSV = () => {
    const header = ["ID","Entity","Action","ActorID","ActorName","Date","Path"]
    const rows = data.map((d:any)=>[
      d.id, d.entity_name, d.action, d.actor_id ?? "-", d.actor_name ?? "-", d.created_at ?? "-", d.request_path ?? "-"
    ])
    const csv = [header, ...rows].map(r=>r.join(",")).join("\n")
    const blob = new Blob([csv], { type: "text/csv" })
    const a = document.createElement("a")
    a.href = URL.createObjectURL(blob)
    a.download = `security_audit_${new Date().toISOString().slice(0,10)}.csv`
    a.click()
  }

  useEffect(() => {
    ;(async () => {
      try {
        const logs = await fetchSecurityAudits()
        setData(logs)
        computeDaily(logs)
      } catch {
        toast.error("Gagal memuat audit keamanan.")
      }
    })()
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <ShieldCheck className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold">Security Audit Dashboard</h1>
      </div>

      {/* Filter */}
      <div className="flex flex-wrap gap-2 items-end">
        {/* Actor (autosuggest ID – Name) */}
        <div>
          <Input
            list="actor-suggest"
            placeholder="User ID"
            value={filters.user_id}
            onChange={(e) => setFilters({ ...filters, user_id: e.target.value })}
            className="w-40"
          />
          <datalist id="actor-suggest">
            {meta.actors.map(a => (
              <option key={a.id} value={String(a.id)}>{`${a.id} - ${a.name || "Unknown"}`}</option>
            ))}
          </datalist>
        </div>

        {/* Action */}
        <div>
          <Input
            list="action-suggest"
            placeholder="Action (LOGIN, LOGOUT)"
            value={filters.action}
            onChange={(e) => setFilters({ ...filters, action: e.target.value })}
            className="w-44"
          />
          <datalist id="action-suggest">
            {meta.actions.map(a => (
              <option key={a} value={a.toUpperCase()} />
            ))}
          </datalist>
        </div>

        <Input type="date" value={filters.start_date}
               onChange={(e)=>setFilters({...filters, start_date: e.target.value})}/>
        <Input type="date" value={filters.end_date}
               onChange={(e)=>setFilters({...filters, end_date: e.target.value})}/>

        <Button onClick={handleFilter}>Terapkan</Button>
        <Button variant="outline" onClick={exportCSV}><Download className="h-4 w-4 mr-1" /> Export CSV</Button>
      </div>

      {/* Chart */}
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2">
          <Activity className="h-5 w-5 text-primary" /> Aktivitas Harian
        </CardTitle></CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={daily}>
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="count" stroke="#4f46e5" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader><CardTitle>Riwayat Audit</CardTitle></CardHeader>
        <CardContent>
          <div className="overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Entity</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Actor</TableHead>
                  <TableHead>Waktu</TableHead>
                  <TableHead>Path</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((r:any)=>(
                  <TableRow key={r.id}>
                    <TableCell>{r.id}</TableCell>
                    <TableCell>{r.entity_name}</TableCell>
                    <TableCell>{r.action}</TableCell>
                    <TableCell>
                      {r.actor_id ? `${r.actor_id} — ${r.actor_name ?? "Unknown"}` : "-"}
                    </TableCell>
                    <TableCell>{r.created_at ? new Date(r.created_at).toLocaleString("id-ID") : "-"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{r.request_path}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
