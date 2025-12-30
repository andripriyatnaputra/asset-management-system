import { useEffect, useState } from "react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "@/components/ui/table"
import { fetchSecurityRisks } from "@/services/riskAdvisorService"
import { ShieldAlert, Zap } from "lucide-react"
import { toast } from "sonner"

interface RiskEntry {
  actor_id: number | null
  action: string
  score: number
  severity: string
  message: string
}

export default function SecurityRiskAdvisorPage() {
  const [risks, setRisks] = useState<RiskEntry[]>([])

  useEffect(() => {
    ;(async () => {
      try {
        const data = await fetchSecurityRisks()
        setRisks(data)
      } catch {
        toast.error("Gagal memuat rekomendasi risiko.")
      }
    })()
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <ShieldAlert className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold">AI Risk Advisor</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-yellow-500" /> Rekomendasi Keamanan Otomatis
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Severity</TableHead>
                <TableHead>Score</TableHead>
                <TableHead>Rekomendasi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {risks.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    Tidak ada risiko signifikan terdeteksi.
                  </TableCell>
                </TableRow>
              ) : (
                risks.map((r, i) => (
                  <TableRow key={i}>
                    <TableCell>{r.actor_id || "-"}</TableCell>
                    <TableCell>{r.action}</TableCell>
                    <TableCell
                      className={
                        r.severity === "high"
                          ? "text-red-600 font-semibold"
                          : r.severity === "medium"
                          ? "text-yellow-600"
                          : "text-green-600"
                      }
                    >
                      {r.severity.toUpperCase()}
                    </TableCell>
                    <TableCell>{r.score.toFixed(0)}%</TableCell>
                    <TableCell>{r.message}</TableCell>
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
