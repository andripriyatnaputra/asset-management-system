import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useWebSocket } from "@/hooks/useWebSocket"
import { toast } from "sonner"

// Bentuk alert yang dikirim dari backend
interface Alert {
  id?: number
  message: string
  severity: "info" | "warning" | "critical" | string
  category?: string
  created_at?: string
  timestamp?: string
}

export default function AlertsCenterPage() {
  const { alerts: liveAlerts } = useWebSocket()
  const [historyAlerts, setHistoryAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)

  // Ambil riwayat alert dari REST API saat halaman dibuka
  useEffect(() => {
    const loadHistory = async () => {
      try {
        const res = await apiClient.get("/alerts")
        setHistoryAlerts(res.data.alerts ?? [])
      } catch (err: any) {
        toast.error("Gagal memuat data alert.")
        console.error("GetAllAlerts error:", err)
      } finally {
        setLoading(false)
      }
    }
    loadHistory()
  }, [])

  // Gabungkan alert lama + realtime (hindari duplikat)
  const mergedAlerts = [
    ...liveAlerts.filter((la) => !historyAlerts.some((ha) => ha.message === la.message)),
    ...historyAlerts,
  ]

  const renderTime = (a: Alert) => {
    const t = a.timestamp || a.created_at
    if (!t) return "-"
    try {
      const d = new Date(t)
      return d.toLocaleString("id-ID", { hour12: false })
    } catch {
      return t
    }
  }

  const badgeVariant = (sev: string) => {
    switch (sev) {
      case "critical":
        return "destructive"
      case "warning":
        return "secondary"
      case "info":
        return "outline"
      default:
        return "outline"
    }
  }

  return (
    <div className="p-6 space-y-4">
      <h1 className="text-xl font-semibold">Live Alerts Center</h1>

      <Card>
        <CardHeader>
          <CardTitle>
            Realtime &amp; Historical Alerts{" "}
            {!loading && `(${mergedAlerts.length})`}
          </CardTitle>
        </CardHeader>

        <CardContent className="space-y-2 max-h-[600px] overflow-y-auto">
          {loading ? (
            <p className="text-muted-foreground italic">Memuat data alert...</p>
          ) : mergedAlerts.length === 0 ? (
            <p className="text-muted-foreground">Belum ada alert baru.</p>
          ) : (
            mergedAlerts.map((a, i) => (
            <div
              key={`${("id" in a ? a.id : i)}-${a.message}`}
              className="flex items-start justify-between border-b pb-2 last:border-none"
            >
                <div className="pr-4">
                  <p className="font-medium">{a.message}</p>
                  <p className="text-xs text-muted-foreground">
                    {renderTime(a)}
                  </p>
                </div>
                <Badge variant={badgeVariant(a.severity)}>
                  {a.severity.toUpperCase()}
                </Badge>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
