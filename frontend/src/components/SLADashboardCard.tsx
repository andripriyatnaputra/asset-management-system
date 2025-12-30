import { useEffect, useState } from "react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import apiClient from "@/services/api"

export default function SLADashboardCard() {
  const [data, setData] = useState({ open_tickets: 0, breached_tickets: 0, resolved_tickets: 0 })

  const fetchData = async () => {
    try {
      const res = await apiClient.get("/dashboard/sla")
      setData(res.data)
    } catch {
      console.error("Failed to fetch SLA stats.")
    }
  }

  useEffect(() => { fetchData() }, [])

  return (
    <Card className="shadow-sm">
      <CardHeader>
        <CardTitle>SLA Performance Overview</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-3 text-center">
        <div>
          <p className="text-3xl font-bold text-blue-600">{data.open_tickets}</p>
          <p className="text-sm text-muted-foreground">Open Tickets</p>
        </div>
        <div>
          <p className="text-3xl font-bold text-red-600">{data.breached_tickets}</p>
          <p className="text-sm text-muted-foreground">SLA Breached</p>
        </div>
        <div>
          <p className="text-3xl font-bold text-green-600">{data.resolved_tickets}</p>
          <p className="text-sm text-muted-foreground">Resolved</p>
        </div>
      </CardContent>
    </Card>
  )
}
