import { useEffect, useState } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { AlertTriangle, Activity } from "lucide-react"
import { fetchSecurityAnomalies } from "@/services/anomalyService"
import { toast } from "sonner"
import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, Tooltip } from "recharts"

interface AnomalyEntry {
  actor_id: number | null
  action: string
  total: number
  date: string
  score: number
}

export default function SecurityAnomalyPage() {
  const [data, setData] = useState<AnomalyEntry[]>([])
  const [highRisk, setHighRisk] = useState<AnomalyEntry[]>([])

  useEffect(() => {
    ;(async () => {
        try {
        const logs: AnomalyEntry[] = await fetchSecurityAnomalies()
        setData(logs)
        setHighRisk(logs.filter((l: AnomalyEntry) => l.score > 3))
        } catch {
        toast.error("Gagal memuat data anomali keamanan.")
        }
    })()
    }, [])

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <AlertTriangle className="h-6 w-6 text-red-500" />
        <h1 className="text-2xl font-bold">Security Anomaly Detection</h1>
      </div>

      {/* Grafik tren aktivitas */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-primary" /> Aktivitas Login/Logout 30 Hari Terakhir
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={data}>
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="total" stroke="#4f46e5" strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Daftar anomali tinggi */}
      <Card>
        <CardHeader>
          <CardTitle>Potensi Anomali</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Tanggal</TableHead>
                <TableHead>Jumlah</TableHead>
                <TableHead>Skor Anomali</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {highRisk.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    Tidak ada aktivitas mencurigakan terdeteksi.
                  </TableCell>
                </TableRow>
              ) : (
                highRisk.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{r.actor_id || "-"}</TableCell>
                    <TableCell>{r.action}</TableCell>
                    <TableCell>{r.date}</TableCell>
                    <TableCell>{r.total}</TableCell>
                    <TableCell className="font-semibold text-red-600">
                      {r.score.toFixed(2)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
