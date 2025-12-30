import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import apiClient from "@/services/api"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import { Tooltip, TooltipProvider, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip"
import { toast } from "sonner"
import { Plus, Trash2, ExternalLink } from "lucide-react"

type Training = {
  id: number
  training_name: string
  certificate_url?: string | null
  completed_at?: string | null
}

export default function EmployeeTrainingPage() {
  const { id } = useParams()
  const [list, setList] = useState<Training[]>([])
  const [trainingName, setTrainingName] = useState("")
  const [certURL, setCertURL] = useState("")
  const [loading, setLoading] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<Training | null>(null)

  const fetchTrainings = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get(`/employees/${id}/trainings`)
      const data = Array.isArray(res.data) ? res.data : res.data?.data ?? []
      setList(data)
    } catch {
      toast.error("Gagal memuat data training.")
      setList([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchTrainings() }, [id])

  const handleAdd = async () => {
    if (!trainingName.trim()) return toast.error("Nama training wajib diisi.")
    try {
      await apiClient.post(`/employees/${id}/trainings`, {
        training_name: trainingName.trim(),
        certificate_url: certURL.trim() || null,
        completed_at: new Date().toISOString().split("T")[0],
      })
      toast.success("Training ditambahkan.")
      setTrainingName("")
      setCertURL("")
      fetchTrainings()
    } catch {
      toast.error("Gagal menambah training.")
    }
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/employees/${id}/trainings/${deleteItem.id}`)
      toast.success(`Training "${deleteItem.training_name}" dihapus.`)
      setConfirmOpen(false)
      fetchTrainings()
    } catch {
      toast.error("Gagal menghapus training.")
    }
  }

  return (
    <div className="p-6 space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>📘 Pelatihan Karyawan #{id}</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Form tambah training */}
          <div className="grid md:grid-cols-3 gap-3 mb-6">
            <Input
              placeholder="Nama training..."
              value={trainingName}
              onChange={(e) => setTrainingName(e.target.value)}
            />
            <Input
              placeholder="URL Sertifikat (opsional)"
              value={certURL}
              onChange={(e) => setCertURL(e.target.value)}
            />
            <Button onClick={handleAdd} className="w-full md:w-auto">
              <Plus size={16} className="mr-1" /> Tambah
            </Button>
          </div>

          {/* Table */}
          <div className="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nama Training</TableHead>
                  <TableHead>Sertifikat</TableHead>
                  <TableHead>Tanggal</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center h-24 text-muted-foreground">
                      Memuat data...
                    </TableCell>
                  </TableRow>
                ) : list.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center h-24 text-muted-foreground">
                      Belum ada data training.
                    </TableCell>
                  </TableRow>
                ) : (
                  list.map((t) => (
                    <TableRow key={t.id} className="hover:bg-accent/40 transition-all">
                      <TableCell>{t.training_name}</TableCell>
                      <TableCell>
                        {t.certificate_url ? (
                          <a
                            href={t.certificate_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-primary text-sm underline flex items-center gap-1"
                          >
                            <ExternalLink size={14} /> Lihat
                          </a>
                        ) : (
                          <span className="text-muted-foreground text-sm italic">-</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {t.completed_at
                          ? new Date(t.completed_at).toLocaleDateString()
                          : "-"}
                      </TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="destructive"
                                  onClick={() => { setDeleteItem(t); setConfirmOpen(true) }}
                                >
                                  <Trash2 size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Training</TooltipContent>
                            </Tooltip>
                          </div>
                        </TooltipProvider>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Training</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus training "<strong>{deleteItem?.training_name}</strong>"?
              <br />Tindakan ini tidak dapat dibatalkan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
