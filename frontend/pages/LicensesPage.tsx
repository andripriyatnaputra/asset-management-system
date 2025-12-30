import { useEffect, useState, useMemo } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import {
  Card, CardHeader, CardTitle, CardContent
} from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import {
  Tooltip, TooltipProvider, TooltipTrigger, TooltipContent
} from "@/components/ui/tooltip"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  Plus, Search, Edit3, Trash2, KeyRound, ChevronLeft, ChevronRight
} from "lucide-react"
import LicenseFormModal from "@/components/LicenseFormModal"

type License = {
  id: number
  name: string
  vendor?: string | null
  license_key?: string | null
  license_type?: string | null
  license_model?: string | null
  total_seats?: number | null
  used_seats?: number | null
  expiration_date?: string | null
  status?: string | null
  cost?: number | null
  contract_id?: number | null
  purchase_date?: string | null
}

const pageSizeOptions = [10, 25, 50]

export default function LicensesPage() {
  const [list, setList] = useState<License[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [openForm, setOpenForm] = useState(false)
  const [editItem, setEditItem] = useState<License | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<License | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  // sorting
  const [sortKey, setSortKey] = useState<"name" | "vendor" | "expiration_date" | "status" | "used_seats">("name")
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc")

  const handleSort = (key: typeof sortKey) => {
    if (sortKey === key) setSortDir(d => (d === "asc" ? "desc" : "asc"))
    else { setSortKey(key); setSortDir("asc") }
    setPage(1)
  }

  const sortArrow = (col: typeof sortKey) =>
    sortKey === col ? (sortDir === "asc" ? " ↑" : " ↓") : ""

  const fetchLicenses = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get("/licenses")
      const data = res.data?.data ?? res.data

      setList(
        (Array.isArray(data) ? data : []).map((d: any): License => ({
          id: d.id,
          name: d.name,
          vendor: d.vendor,
          license_key: d.license_key,
          license_type: d.license_type,
          license_model: d.license_model,
          total_seats: d.total_seats,
          used_seats: d.used_seats,
          expiration_date: d.expiration_date,
          status: d.status,
          cost: d.cost,
          contract_id: d.contract_id,
          purchase_date: d.purchase_date,
        }))
      )
    } catch {
      toast.error("Gagal memuat data lisensi.")
      setList([]) // fallback aman
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchLicenses() }, [])

  const filtered: License[] = Array.isArray(list)
    ? list.filter(l =>
        (l.name ?? "").toLowerCase().includes(search.toLowerCase()) ||
        (l.vendor ?? "").toLowerCase().includes(search.toLowerCase())
      )
    : []

  const sorted = useMemo(() => {
    const arr = [...filtered]
    arr.sort((a, b) => {
      const dir = sortDir === "asc" ? 1 : -1
      const val = (k: typeof sortKey, x: License) => {
        if (k === "used_seats") return x.used_seats ?? 0
        if (k === "expiration_date") return x.expiration_date ? new Date(x.expiration_date).getTime() : 0
        if (k === "status") return (x.status ?? "").toLowerCase()
        if (k === "vendor") return (x.vendor ?? "").toLowerCase()
        return (x.name ?? "").toLowerCase()
      }
      const va = val(sortKey, a), vb = val(sortKey, b)
      if (va < vb) return -1 * dir
      if (va > vb) return  1 * dir
      return 0
    })
    return arr
  }, [filtered, sortKey, sortDir])

  const totalPages = Math.ceil(sorted.length / pageSize) || 1
  const paginated = sorted.slice((page - 1) * pageSize, page * pageSize)

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/licenses/${deleteItem.id}`)
      toast.success(`Lisensi "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchLicenses()
    } catch {
      toast.error("Gagal menghapus lisensi.")
    }
  }

  const getStatusBadge = (status?: string | null) => {
    const s = (status || "").toLowerCase()
    if (s === "compliant") return <Badge className="bg-green-500/10 text-green-700 border-green-400">Compliant</Badge>
    if (s === "non-compliant") return <Badge className="bg-red-500/10 text-red-700 border-red-400">Non-Compliant</Badge>
    if (s === "unknown") return <Badge variant="outline">Unknown</Badge>
    return <Badge variant="outline">-</Badge>
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        {/* Header */}
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            <KeyRound size={18} /> Manajemen Lisensi
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari lisensi / vendor..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-[240px]"
            />
            <Button
              onClick={() => {
                setEditItem(null)
                setOpenForm(true)
              }}
              className="gap-1"
            >
              <Plus size={16} /> Tambah
            </Button>
          </div>
        </CardHeader>

        <CardContent>
          {/* Table */}
          <div className="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>

                  <TableHead
                    onClick={() => handleSort("name")}
                    className="cursor-pointer select-none"
                  >
                    Nama{sortArrow("name")}
                  </TableHead>

                  <TableHead
                    onClick={() => handleSort("vendor")}
                    className="cursor-pointer select-none"
                  >
                    Vendor{sortArrow("vendor")}
                  </TableHead>

                  <TableHead>Key</TableHead>

                  <TableHead
                    onClick={() => handleSort("used_seats")}
                    className="cursor-pointer select-none"
                  >
                    Seats{sortArrow("used_seats")}
                  </TableHead>

                  <TableHead
                    onClick={() => handleSort("expiration_date")}
                    className="cursor-pointer select-none"
                  >
                    Kedaluwarsa{sortArrow("expiration_date")}
                  </TableHead>

                  <TableHead
                    onClick={() => handleSort("status")}
                    className="cursor-pointer select-none"
                  >
                    Status{sortArrow("status")}
                  </TableHead>

                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-6">Memuat data...</TableCell></TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-center py-6">Tidak ada lisensi.</TableCell></TableRow>
                ) : (
                  paginated.map((l) => (
                    <TableRow key={l.id} className="hover:bg-accent/40 transition-all">
                      <TableCell>{l.id}</TableCell>
                      <TableCell>{l.name}</TableCell>
                      <TableCell>{l.vendor || "-"}</TableCell>
                      <TableCell className="font-mono text-xs">{l.license_key || "-"}</TableCell>
                      <TableCell>{l.used_seats ?? 0}/{l.total_seats ?? 0}</TableCell>
                      <TableCell>{l.expiration_date ? new Date(l.expiration_date).toLocaleDateString() : "-"}</TableCell>
                      <TableCell>{getStatusBadge(l.status)}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="outline"
                                  onClick={() => { setEditItem(l); setOpenForm(true) }}
                                >
                                  <Edit3 size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Lisensi</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="destructive"
                                  onClick={() => { setDeleteItem(l); setConfirmOpen(true) }}
                                >
                                  <Trash2 size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Lisensi</TooltipContent>
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

          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 text-sm">
            <div className="flex items-center gap-2">
              <span>Baris per halaman:</span>
              <select
                value={pageSize}
                onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1) }}
                className="border rounded px-2 py-1 text-sm"
              >
                {pageSizeOptions.map(size => (
                  <option key={size} value={size}>{size}</option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <span>Halaman {page} dari {totalPages}</span>
              <Button variant="outline" size="icon" disabled={page === 1} onClick={() => setPage(p => Math.max(1, p - 1))}>
                <ChevronLeft size={16} />
              </Button>
              <Button variant="outline" size="icon" disabled={page === totalPages} onClick={() => setPage(p => Math.min(totalPages, p + 1))}>
                <ChevronRight size={16} />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Modals */}
      <LicenseFormModal
        isOpen={openForm}
        onClose={() => setOpenForm(false)}
        onSuccess={fetchLicenses}
        license={editItem}
      />

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Lisensi</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus lisensi "<strong>{deleteItem?.name}</strong>"?
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
