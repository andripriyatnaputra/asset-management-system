import { useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import apiClient from "@/services/api"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { ArrowLeft, RefreshCcw, Monitor, KeyRound, Wallet } from "lucide-react"

export default function DepartmentSummaryPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [summary, setSummary] = useState<{
    department_id: number
    total_assets: number
    total_licenses: number
    total_budget: number
  } | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchSummary = async () => {
    if (!id) return
    setLoading(true)
    try {
      const res = await apiClient.get(`/departments/${id}/summary`)
      setSummary(res.data)
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal memuat ringkasan departemen.")
      setSummary(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchSummary() }, [id])

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-semibold flex items-center gap-2">
            🏢 Ringkasan Departemen #{id}
          </h1>
          <p className="text-sm text-muted-foreground">
            Data agregat aset, lisensi, dan anggaran departemen ini.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => navigate(-1)}>
            <ArrowLeft className="mr-2 h-4 w-4" /> Kembali
          </Button>
          <Button variant="outline" onClick={fetchSummary}>
            <RefreshCcw className="mr-2 h-4 w-4" /> Refresh
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="text-center text-muted-foreground py-20">
          Memuat data ringkasan...
        </div>
      ) : !summary ? (
        <div className="text-center text-destructive py-20">
          Gagal memuat data departemen.
        </div>
      ) : (
        <div className="grid sm:grid-cols-3 gap-6">
          {/* Card: Total Aset */}
          <Card className="border hover:shadow-md transition-all">
            <CardHeader className="flex items-center justify-between pb-2">
              <CardTitle className="text-base flex items-center gap-2">
                <Monitor size={18} className="text-primary" /> Total Aset
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-4xl font-bold text-primary">
                {summary.total_assets}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                Aset terdaftar di departemen ini
              </p>
            </CardContent>
          </Card>

          {/* Card: Total Lisensi */}
          <Card className="border hover:shadow-md transition-all">
            <CardHeader className="flex items-center justify-between pb-2">
              <CardTitle className="text-base flex items-center gap-2">
                <KeyRound size={18} className="text-blue-500" /> Total Lisensi
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-4xl font-bold text-blue-500">
                {summary.total_licenses}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                Lisensi perangkat lunak aktif
              </p>
            </CardContent>
          </Card>

          {/* Card: Total Anggaran */}
          <Card className="border hover:shadow-md transition-all">
            <CardHeader className="flex items-center justify-between pb-2">
              <CardTitle className="text-base flex items-center gap-2">
                <Wallet size={18} className="text-green-600" /> Total Anggaran
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-4xl font-bold text-green-600">
                Rp {Number(summary.total_budget || 0).toLocaleString("id-ID")}
              </p>
              <p className="text-sm text-muted-foreground mt-1">
                Anggaran aktif departemen ini
              </p>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
