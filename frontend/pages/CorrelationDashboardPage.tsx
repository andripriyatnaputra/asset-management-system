import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { ResponsiveContainer, ScatterChart, Scatter, XAxis, YAxis, ZAxis, Tooltip, Cell } from "recharts"

export default function CorrelationDashboardPage() {
  const [data, setData] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    apiClient.get("/dashboard/correlation")
      .then(res => setData(res.data.data || []))
      .catch(() => toast.error("Gagal memuat data korelasi"))
      .finally(() => setIsLoading(false))
  }, [])

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader><CardTitle>Asset–Alert–Ticket Correlation Map</CardTitle></CardHeader>
        <CardContent className="h-[350px]">
          {isLoading ? (
            <div className="h-full animate-pulse rounded bg-muted" />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <ScatterChart>
                <XAxis dataKey="alert_ratio" name="Alert Ratio (%)" />
                <YAxis dataKey="ticket_ratio" name="Ticket Ratio (%)" />
                <ZAxis dataKey="avg_health" range={[100, 600]} />
                <Tooltip cursor={{ strokeDasharray: "3 3" }} />
                <Scatter data={data}>
                  {data.map((d, i) => {
                    const risk = Math.max(0, 100 - d.avg_health)
                    const red = Math.min(255, risk * 2.55)
                    const green = 255 - red
                    return <Cell key={i} fill={`rgb(${red},${green},60)`} />
                  })}
                </Scatter>
              </ScatterChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
