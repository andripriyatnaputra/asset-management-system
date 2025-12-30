import { useEffect, useState } from "react"
import { toast } from "sonner"
import apiClient from "@/services/api"
import {
  Card, CardContent, CardHeader, CardTitle,
} from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import AssetTypeChart from "@/components/AssetTypeChart"
import EmployeeDeptChart from "@/components/EmployeeDeptChart"
import {
  PieChart as PieIcon, Users, Box, Activity, Clock, GitBranch,
} from "lucide-react"
import SLADashboardCard from "@/components/SLADashboardCard"
import GovernanceComplianceChart from "@/components/GovernanceComplianceChart"
import AlertSummaryCard from "@/components/AlertSummaryCard"
import { useWebSocket } from "@/hooks/useWebSocket"
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer,
  CartesianGrid, LineChart, ScatterChart, ZAxis, Scatter, Cell,
  ComposedChart, Line, Legend,
} from "recharts"

interface StatCard { title: string; value: number }
interface RecentActivity { asset_name: string; employee_name: string; assigned_at: string }
interface ChartData { name: string; value: number }

interface AssetMetric {
  name: string
  avg_health: number
  avg_governance: number
  compliance_rate: number
  total_assets: number
}

interface DashboardStats {
  stat_cards: StatCard[]
  recent_activity: RecentActivity[]
  assets_by_type: ChartData[]
  employees_by_dept: ChartData[]
  asset_metrics_by_dept: AssetMetric[]
  compliance?: ChartData[]
  // predictive_risk ada di API, tapi tidak dipakai di UI saat ini -> bisa any[]
  predictive_risk?: any[]
}

interface KGSummary {
  orphaned_assets: number
  assets_with_breached_tickets: number
  high_risk_contracts: number
  total_nodes: number
  total_edges: number
}

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [slaStats, setSlaStats] = useState<any | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [trendData, setTrendData] = useState<any[]>([])
  const [forecast, setForecast] = useState<any | null>(null)
  const [alertStats, setAlertStats] = useState<any[]>([])
  const [alertTrend, setAlertTrend] = useState<any[]>([])
  const [heatmapData, setHeatmapData] = useState<any[]>([])
  const [recommendations, setRecommendations] = useState<any[]>([])
  const [forecastData, setForecastData] = useState<any[]>([])
  const [, setCompliance] = useState<any[]>([])
  const [kgSummary, setKgSummary] = useState<KGSummary | null>(null)
  const { connected, lastMessage } = useWebSocket() // ✅ universal realtime hook

  const isNEA = (v: unknown): v is any[] => Array.isArray(v) && v.length > 0
  const asArray = (v: unknown): any[] => (Array.isArray(v) ? v : [])


  // 🧭 Tangani pesan realtime dari WebSocket
  useEffect(() => {
    if (!lastMessage) return

    switch (lastMessage.type) {
      case "alert":
        toast.warning(
          lastMessage.data?.message || "⚠️ Alert baru terdeteksi",
          { position: "top-right" }
        )
        refreshAlertData()
        break

      case "audit":
        console.log("📜 Audit event:", lastMessage)
        break

      case "ticket":
        toast.info("🎫 Pembaruan tiket diterima")
        break

      default:
        console.log("🔔 WS event:", lastMessage)
    }
  }, [lastMessage])

  // 🔹 Load data utama dashboard
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [main, sla, kg] = await Promise.all([
          apiClient.get("/dashboard/stats"),
          apiClient.get("/dashboard/sla"),
          apiClient.get("/dashboard/kg-summary"),
        ])
        setStats(main.data)
        setSlaStats(sla.data)
        setCompliance(main.data?.compliance || [])
        setKgSummary(kg.data || null)

        // data tambahan untuk role non-employee
        const token = localStorage.getItem("authToken")
        const payload = token ? JSON.parse(atob(token.split(".")[1])) : null
        const role = payload?.role

        if (["super_admin", "asset_manager", "finance", "it_support", "manager"].includes(role)) {
          const [alerts, trends, heatmap, forecastRes, recs] = await Promise.all([
            apiClient.get("/dashboard/alert-stats"),
            apiClient.get("/dashboard/alert-trends"),
            apiClient.get("/dashboard/health-heatmap"),
            apiClient.get("/dashboard/predictive-forecast"),
            apiClient.get("/dashboard/recommendations"),
          ])
          setAlertStats(alerts.data.data || [])
          setAlertTrend(trends.data.data || [])
          setHeatmapData(heatmap.data.data || [])
          setForecastData(forecastRes.data.data || [])
          setRecommendations(recs.data.recommendations || [])
        } else {
          setAlertStats([])
          setAlertTrend([])
          setHeatmapData([])
          setForecastData([])
          setRecommendations([])
        }
      } catch (err) {
        console.error(err)
        toast.error("Gagal memuat data dashboard.")
      } finally {
        setIsLoading(false)
      }
    }
    fetchData()
  }, [])

  // 🔹 Load tren & forecast tambahan
  useEffect(() => {
    const fetchTrendAndForecast = async () => {
      try {
        const [trendRes, forecastRes] = await Promise.all([
          apiClient.get("/dashboard/trend-health-sla"),
          apiClient.get("/dashboard/forecast-health-sla"),
        ])
        setTrendData(trendRes.data.trend || [])
        setForecast(forecastRes.data ?? null)
      } catch (err: any) {
        console.warn("Tidak ada akses ke data tren / prediksi:", err?.response?.status)
        setTrendData([])
        setForecast(null)
      }
    }
    fetchTrendAndForecast()
  }, [])

  // 🔄 Refresh ringkasan alert saat ada event baru
  const refreshAlertData = async () => {
    try {
      const [alerts, trends] = await Promise.all([
        apiClient.get("/dashboard/alert-stats"),
        apiClient.get("/dashboard/alert-trends"),
      ])
      setAlertStats(alerts.data.data || [])
      setAlertTrend(trends.data.data || [])
    } catch (err) {
      console.warn("Gagal refresh data alert:", err)
    }
  }

  const iconSet = [Box, Activity, PieIcon, Users]

  return (
    <div className="space-y-6 p-4 md:p-6">
      {/* ✅ QUICK ACTIONS */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-xl font-semibold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          WebSocket: {connected ? "🟢 Connected" : "🔴 Disconnected"}
        </p>
      </div>

      {/* ✅ STAT CARDS */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => (
              <Card key={i}>
                <CardHeader className="pb-2">
                  <div className="h-4 w-24 animate-pulse rounded bg-muted" />
                </CardHeader>
                <CardContent>
                  <div className="h-8 w-16 animate-pulse rounded bg-muted" />
                </CardContent>
              </Card>
            ))
          : stats?.stat_cards?.map((card, idx) => {
              const Icon = iconSet[idx % iconSet.length]
              return (
                <Card
                  key={idx}
                  className="bg-[radial-gradient(800px_300px_at_0%_-50%,hsl(var(--primary)/0.04),transparent_70%)]"
                >
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle className="text-sm font-medium">
                      {card.title}
                    </CardTitle>
                    <Icon className="h-5 w-5 text-muted-foreground" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-3xl font-semibold">
                      {card.value}
                    </div>
                  </CardContent>
                </Card>
              )
            })}
      </div>

      {/* ✅ SLA OVERVIEW */}
      <div className="grid md:grid-cols-2 gap-4">
        <Card className="shadow-sm border">
          <CardHeader className="flex flex-row justify-between items-center">
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <Clock className="w-4 h-4 text-primary" /> SLA Overview
            </CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="h-24 animate-pulse rounded bg-muted" />
            ) : slaStats ? (
              <div className="grid grid-cols-3 text-center gap-y-4">
                <div>
                  <p className="text-3xl font-bold text-blue-600">{slaStats.open_tickets}</p>
                  <p className="text-sm text-muted-foreground">Open</p>
                </div>
                <div>
                  <p className="text-3xl font-bold text-destructive">{slaStats.breached_tickets}</p>
                  <p className="text-sm text-muted-foreground">SLA Breached</p>
                </div>
                <div>
                  <p className="text-3xl font-bold text-green-600">{slaStats.resolved_tickets}</p>
                  <p className="text-sm text-muted-foreground">Resolved</p>
                </div>
                <div>
                  <p className="font-semibold">
                    {slaStats.sla_compliance_rate?.toFixed(1) ?? "--"}%
                  </p>
                  <p className="text-muted-foreground">Compliance</p>
                </div>
                <div>
                  <p className="font-semibold">{slaStats.avg_mttr_minutes ?? "--"}</p>
                  <p className="text-muted-foreground"> (min)</p>
                </div>
                <div>
                  <p className="font-semibold">{slaStats.avg_mtta_minutes ?? "--"}</p>
                  <p className="text-muted-foreground">Avg MTTA (min)</p>
                </div>
              </div>
            ) : (
              <p className="text-muted-foreground">Tidak ada data SLA.</p>
            )}
          </CardContent>
        </Card>

        <SLADashboardCard />
      </div>

      {/* ✅ CHART SECTION */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* ✅ Governance Compliance Overview */}
        <Card className="min-h-[360px]">
          <CardHeader>
            <CardTitle>Kepatuhan Governance Aset</CardTitle>
          </CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <SkeletonChart />
            ) : (
              <GovernanceComplianceChart
                data={
                  Array.isArray(stats?.compliance) && stats.compliance.length > 0
                    ? stats.compliance
                    : []
                }
              />
            )}
          </CardContent>
        </Card>

        {/* ✅ Active Alert Summary */}
        <Card className="min-h-[360px]">
          <CardHeader>
            <CardTitle>Ringkasan Alert Aktif</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <SkeletonChart />
            ) : (
              <AlertSummaryCard
                data={stats?.stat_cards?.filter((s) => s.title.startsWith("Alert")) || []}
              />
            )}
          </CardContent>
        </Card>

        {/* 🔍 Knowledge Graph Governance Summary */}
        <Card className="min-h-[360px]">
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <GitBranch className="h-4 w-4 text-primary" />
              Knowledge Graph Governance
            </CardTitle>
          </CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <SkeletonChart />
            ) : !kgSummary ? (
              <EmptyState message="Knowledge Graph belum memiliki data yang cukup atau belum di-build." />
            ) : (
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div className="col-span-2">
                  <p className="text-xs text-muted-foreground">Topology</p>
                  <p className="text-lg font-semibold">
                    {kgSummary.total_nodes} nodes • {kgSummary.total_edges} edges
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Orphaned Assets</p>
                  <p className="text-2xl font-bold text-destructive">
                    {kgSummary.orphaned_assets}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Assets with Breached Tickets</p>
                  <p className="text-2xl font-bold text-amber-600">
                    {kgSummary.assets_with_breached_tickets}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">High-Exposure Contracts (≥5 assets)</p>
                  <p className="text-xl font-semibold text-blue-600">
                    {kgSummary.high_risk_contracts}
                  </p>
                </div>
                <p className="col-span-2 text-[11px] text-muted-foreground">
                  Data dihitung dari kg_nodes / kg_edges • dipakai untuk analitik blast radius & risiko governance.
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* ✅ Aset per Tipe */}
        <Card className="min-h-[360px]">
          <CardHeader><CardTitle>Aset per Tipe</CardTitle></CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <SkeletonChart />
            ) : isNEA(stats?.assets_by_type) ? (
              <AssetTypeChart data={stats!.assets_by_type || []} />
            ) : (
              <EmptyState message="Belum ada data aset per tipe." />
            )}
          </CardContent>
        </Card>

        {/* ✅ Karyawan per Departemen */}
        <Card className="min-h-[360px]">
          <CardHeader><CardTitle>Karyawan per Departemen</CardTitle></CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <SkeletonChart />
            ) : isNEA(stats?.employees_by_dept) ? (
              <EmployeeDeptChart data={stats!.employees_by_dept || []} />
            ) : (
              <EmptyState message="Belum ada data karyawan per departemen." />
            )}
          </CardContent>
        </Card>

        {/* ✅ Average Asset Health per Department (pakai AssetMetricsByDept) */}
        <Card className="col-span-2">
          <CardHeader><CardTitle>Rata-rata Kesehatan Aset per Departemen</CardTitle></CardHeader>
          <CardContent className="h-[280px]">
            {isLoading ? (
              <SkeletonChart />
            ) : isNEA(stats?.asset_metrics_by_dept) ? (
              <ResponsiveContainer width="100%" height={250}>
                <BarChart
                  data={asArray(stats?.asset_metrics_by_dept)}
                  margin={{ top: 10, right: 10, left: 0, bottom: 20 }}
                >
                  <XAxis dataKey="name" />
                  <YAxis domain={[0, 100]} />
                  <Tooltip formatter={(v) => `${v}%`} />
                  <Bar dataKey="avg_health" radius={[6, 6, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <EmptyState message="Belum ada data kesehatan aset per departemen." />
            )}
          </CardContent>
        </Card>

        {/* ✅ Health vs SLA Trend */}
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>Tren Korelasi Health Score & SLA Breach</CardTitle>
          </CardHeader>
          <CardContent className="h-[300px]">
            {isNEA(trendData) ? (
              <ResponsiveContainer width="100%" height={280}>
                <ComposedChart data={trendData} margin={{ top: 10, right: 10, left: 0, bottom: 20 }}>
                  <XAxis dataKey="period" />
                  <YAxis yAxisId="left" orientation="left" domain={[0, 100]} />
                  <YAxis yAxisId="right" orientation="right" />
                  <Tooltip />
                  <Legend />
                  <Bar
                    yAxisId="right"
                    dataKey="sla_breach_count"
                    barSize={20}
                    name="SLA Breach"
                  />
                  <Line
                    yAxisId="left"
                    type="monotone"
                    dataKey="avg_health"
                    strokeWidth={3}
                    dot={{ r: 4 }}
                    name="Avg Health"
                  />
                </ComposedChart>
              </ResponsiveContainer>
            ) : (
              <EmptyState message="Belum ada data tren SLA vs Health." />
            )}
          </CardContent>
        </Card>

        {/* 🔮 Predictive Forecast Card */}
        <Card className="col-span-2 border shadow-sm">
          <CardHeader>
            <CardTitle>Prediksi Bulan Berikutnya (AI Forecast)</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-center">
            <div>
              <p className="text-3xl font-bold text-green-600">
                {forecast ? `${forecast.predicted_health_next_month ?? "--"}%` : "--"}
              </p>
              <p className="text-sm text-muted-foreground">Perkiraan Rata-rata Health</p>
            </div>
            <div>
              <p className="text-3xl font-bold text-destructive">
                {forecast ? forecast.predicted_sla_breach_next_month ?? "--" : "--"}
              </p>
              <p className="text-sm text-muted-foreground">Perkiraan Jumlah SLA Breach</p>
            </div>
            <p className="col-span-2 text-xs text-muted-foreground">
              Sampel {forecast?.sample_size || 0} bulan • Prediksi berbasis tren historis linier
            </p>
          </CardContent>
        </Card>

        {/* ✅ Aggregated Alert Stats */}
        <Card className="col-span-2">
          <CardHeader><CardTitle>Tren Jumlah Alert per Severity</CardTitle></CardHeader>
          <CardContent className="h-[250px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={alertStats}>
                <XAxis dataKey="severity" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* ✅ Alert Trend per Hari */}
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>Tren Alert per Hari</CardTitle>
          </CardHeader>
          <CardContent className="h-[250px]">
            {isLoading ? (
              <div className="h-full w-full animate-pulse rounded bg-muted" />
            ) : isNEA(alertTrend) ? (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={alertTrend || []}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="day" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Line type="monotone" dataKey="count" name="Jumlah Alert" />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-muted-foreground">Belum ada data tren alert.</p>
            )}
          </CardContent>
        </Card>

        {/* ✅ Department Risk Heatmap */}
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>Department Risk Heatmap</CardTitle>
          </CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <div className="h-full w-full animate-pulse rounded bg-muted" />
            ) : isNEA(heatmapData) ? (
              <ResponsiveContainer width="100%" height="100%">
                <ScatterChart margin={{ top: 20, right: 20, bottom: 10, left: 0 }}>
                  <XAxis type="category" dataKey="department" name="Department" />
                  <YAxis type="number" dataKey="avg_health" name="Avg Health" domain={[0, 100]} />
                  <ZAxis type="number" dataKey="alert_count" range={[100, 600]} />
                  <Tooltip cursor={{ strokeDasharray: "3 3" }} />
                  <Scatter data={heatmapData}>
                    {heatmapData.map((entry, index) => {
                      const ratio = entry.avg_health / 100
                      const red = Math.round(255 * (1 - ratio))
                      const green = Math.round(255 * ratio)
                      return <Cell key={`cell-${index}`} fill={`rgb(${red},${green},60)`} />
                    })}
                  </Scatter>
                </ScatterChart>
              </ResponsiveContainer>
            ) : (
              <p className="text-muted-foreground">Tidak ada data heatmap departemen.</p>
            )}
          </CardContent>
        </Card>

        {/* ✅ Prediksi Kesehatan Aset (30 Hari) */}
        <Card className="col-span-2">
          <CardHeader><CardTitle>Prediksi Kesehatan Aset (30 Hari)</CardTitle></CardHeader>
          <CardContent className="h-[300px]">
            {isLoading ? (
              <div className="h-full animate-pulse rounded bg-muted" />
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={forecastData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="department" />
                  <YAxis domain={[0, 100]} />
                  <Tooltip />
                  <Legend />
                  <Line type="monotone" dataKey="avg_health" name="Current" />
                  <Line type="monotone" dataKey="forecast_next_7" name="+7 Days" />
                  <Line type="monotone" dataKey="forecast_next_30" name="+30 Days" />
                </LineChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* ✅ Rekomendasi Preventif */}
        <Card className="col-span-2">
          <CardHeader><CardTitle>Rekomendasi Preventif AI</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {recommendations.map((r, idx) => (
              <div key={idx} className="border rounded-md p-3">
                <p className="font-semibold text-primary">{r.department}</p>
                <p className="text-sm text-muted-foreground">{r.reason}</p>
                <p className="mt-1 text-sm">{r.action}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      {/* ✅ RECENT ACTIVITY */}
      <section>
        <h2 className="mb-3 text-xl font-semibold">Aktivitas Terakhir</h2>
        <Card>
          <CardContent className="p-0">
            {isLoading ? (
              <SkeletonTable />
            ) : isNEA(stats?.recent_activity) ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Nama Aset</TableHead>
                    <TableHead>Diberikan Kepada</TableHead>
                    <TableHead>Tanggal</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {asArray(stats?.recent_activity).map((a: RecentActivity, i: number) => (
                    <TableRow key={`${a.asset_name}-${i}`}>
                      <TableCell className="font-medium">{a.asset_name}</TableCell>
                      <TableCell>{a.employee_name}</TableCell>
                      <TableCell>
                        {a.assigned_at
                          ? new Date(a.assigned_at).toLocaleDateString("id-ID")
                          : "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <div className="p-6">
                <EmptyState message="Belum ada aktivitas terbaru." />
              </div>
            )}
          </CardContent>
        </Card>
      </section>

    </div>
  )
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex h-full min-h-[160px] w-full flex-col items-center justify-center gap-2 text-center">
      <div className="h-10 w-10 rounded-full bg-muted" />
      <p className="max-w-sm text-sm text-muted-foreground">{message}</p>
    </div>
  )
}

function SkeletonChart() {
  return <div className="h-full w-full animate-pulse rounded bg-muted" />
}

function SkeletonTable() {
  return (
    <div className="p-4">
      <div className="mb-2 h-5 w-64 animate-pulse rounded bg-muted" />
      <div className="h-24 w-full animate-pulse rounded bg-muted" />
    </div>
  )
}
