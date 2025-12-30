import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"

interface Alert {
  id: number
  message: string
  severity: string
  category: string
  acknowledged: boolean
  created_at: string
}

export default function AlertsHistoryPage() {
  const [alerts, setAlerts] = useState<Alert[]>([])

  const load = async () => {
    const res = await apiClient.get("/alerts")
    setAlerts(res.data.alerts)
  }

  const handleAck = async (id: number) => {
    await apiClient.post(`/alerts/${id}/ack`)
    toast.success("Alert acknowledged")
    load()
  }

  useEffect(() => {
    load()
  }, [])

  return (
    <div className="p-6 space-y-4">
      <h1 className="text-xl font-semibold">Alert History</h1>
      <Card>
        <CardHeader><CardTitle>Recent Alerts</CardTitle></CardHeader>
        <CardContent>
          {alerts.length === 0 ? (
            <p className="text-muted-foreground">No alerts yet.</p>
          ) : (
            <div className="divide-y">
              {alerts.map(a => (
                <div key={a.id} className="py-3 flex justify-between items-center">
                  <div>
                    <p className="font-medium">{a.message}</p>
                    <p className="text-xs text-muted-foreground">{new Date(a.created_at).toLocaleString("id-ID")}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={
                        a.severity === "critical"
                          ? "destructive"
                          : a.severity === "warning"
                          ? "secondary"
                          : "outline"
                      }
                    >
                      {a.severity.toUpperCase()}
                    </Badge>
                    {!a.acknowledged && (
                      <Button size="sm" onClick={() => handleAck(a.id)}>
                        Acknowledge
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
