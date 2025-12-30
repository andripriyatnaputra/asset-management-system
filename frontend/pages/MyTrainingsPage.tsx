import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

interface Training {
  id: number
  training_name: string
  certificate_url?: string | null
  completed_at?: string | null
  created_at?: string | null
}

export default function MyTrainingsPage() {
  const [trainings, setTrainings] = useState<Training[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    apiClient
      .get("/employees/me/trainings")
        .then(res => setTrainings(Array.isArray(res.data) ? res.data : []))
        .catch(() => toast.error("Gagal memuat data pelatihan Anda."))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    apiClient.get("/employees/me/trainings")
        .then(res => {
        console.log("DEBUG trainings response:", res.data)
        setTrainings(res.data)
        })
        .catch(err => {
        console.error("DEBUG trainings error:", err)
        })
    }, [])

  return (
    <div className="space-y-6 p-4 md:p-6">
      <h1 className="text-2xl font-semibold">Pelatihan Saya</h1>
      <Card>
        <CardHeader>
          <CardTitle>Daftar Pelatihan yang Pernah Diikuti</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama Pelatihan</TableHead>
                <TableHead>Tanggal Selesai</TableHead>
                <TableHead>Sertifikat</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center h-24">
                    Memuat…
                  </TableCell>
                </TableRow>
              ) : (trainings ?? []).length > 0 ? (
                trainings.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell>{t.training_name}</TableCell>
                    <TableCell>
                      {t.completed_at
                        ? new Date(t.completed_at).toLocaleDateString("id-ID")
                        : "-"}
                    </TableCell>
                    <TableCell>
                      {t.certificate_url ? (
                        <a
                          href={t.certificate_url}
                          target="_blank"
                          rel="noreferrer"
                          className="text-blue-600 underline"
                        >
                          Lihat
                        </a>
                      ) : (
                        "-"
                      )}
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={3} className="text-center h-24 text-muted-foreground">
                    Belum ada data pelatihan.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
