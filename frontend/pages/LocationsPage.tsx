import { useEffect, useMemo, useState } from "react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from "@/components/ui/table"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import { Tooltip, TooltipProvider, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip"
import { toast } from "sonner"
import { PencilLine, Trash2, Plus, Search, ArrowUpDown, ChevronLeft, ChevronRight } from "lucide-react"
import apiClient from "@/services/api"
import LocationFormModal, { type Location } from "@/components/LocationFormModal"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const pageSizeOptions = [10, 25, 50]

export default function LocationsPage() {
  const [list, setList] = useState<Location[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)
  const [formOpen, setFormOpen] = useState(false)
  const [editItem, setEditItem] = useState<Location | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<Location | null>(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get("/locations")

      const raw = res.data?.data ?? res.data
      const safeData: Location[] = Array.isArray(raw) ? raw : []

      setList(safeData)
    } catch {
      setList([]) // ⬅️ penting
      toast.error("Gagal memuat data lokasi.")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  const filtered = useMemo(() => {
    if (!Array.isArray(list)) return []

    const q = search.toLowerCase()
    let result = list.filter(l =>
      l.site?.toLowerCase().includes(q) ||
      (l.building ?? "").toLowerCase().includes(q) ||
      (l.room ?? "").toLowerCase().includes(q)
    )

    result.sort((a, b) =>
      sortAsc
        ? a.site.localeCompare(b.site)
        : b.site.localeCompare(a.site)
    )

    return result
  }, [list, search, sortAsc])


  const totalPages = Math.ceil(filtered.length / pageSize)
  const paginated = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/locations/${deleteItem.id}`)
      toast.success(`Lokasi "${deleteItem.site}" dihapus.`)
      setConfirmOpen(false)
      fetchData()
    } catch {
      toast.error("Gagal menghapus lokasi.")
    }
  }

  const openAdd = () => {
    setEditItem(null)
    setFormOpen(true)
  }

  const openEdit = (loc: Location) => {
    setEditItem(loc)
    setFormOpen(true)
  }

  const getStatusBadge = (status?: string) => {
    const isActive = status === "active"
    return (
      <span
        className={`px-2 py-1 rounded-full text-xs font-medium border ${
          isActive
            ? "bg-green-500/10 text-green-700 border-green-400"
            : "bg-gray-400/20 text-gray-500 border-gray-300"
        }`}
      >
        {isActive ? "Active" : "Inactive"}
      </span>
    )
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            🗺️ Manajemen Lokasi
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari site / building / room..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setCurrentPage(1) }}
              className="w-[240px]"
            />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSortAsc(!sortAsc)}
              title="Urutkan site"
            >
              <ArrowUpDown size={18} />
            </Button>
            <Button onClick={openAdd} className="gap-1">
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
                  <TableHead>Site</TableHead>
                  <TableHead>Building</TableHead>
                  <TableHead>Room</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center h-24 text-muted-foreground">
                      Memuat data...
                    </TableCell>
                  </TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center h-24 text-muted-foreground">
                      Tidak ada lokasi.
                    </TableCell>
                  </TableRow>
                ) : (
                  paginated.map(l => (
                    <TableRow key={l.id} className="hover:bg-accent/40 transition-all">
                      <TableCell>{l.site}</TableCell>
                      <TableCell>{l.building || "-"}</TableCell>
                      <TableCell>{l.room || "-"}</TableCell>
                      <TableCell>{getStatusBadge(l.status)}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="outline" onClick={() => openEdit(l)}>
                                  <PencilLine size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit</TooltipContent>
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
                              <TooltipContent>Hapus</TooltipContent>
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
          <div className="flex flex-col md:flex-row items-center justify-between mt-4 text-sm gap-3">
            <div className="flex items-center gap-2">
              <span>Baris per halaman:</span>
              <Select
              value={String(pageSize)}
              onValueChange={(v) => {
                setPageSize(Number(v))
                setCurrentPage(1)
              }}
            >
              <SelectTrigger
                className="
                  w-[90px] text-sm 
                  bg-background text-foreground 
                  border-border
                  focus:ring-2 focus:ring-ring focus:outline-none
                  hover:bg-accent/50 transition-colors
                "
              >
                <SelectValue placeholder="Size" />
              </SelectTrigger>
              <SelectContent
                className="
                  bg-popover text-popover-foreground
                  border border-border shadow-md
                "
              >
                {pageSizeOptions.map((size) => (
                  <SelectItem
                    key={size}
                    value={String(size)}
                    className="cursor-pointer text-sm focus:bg-accent focus:text-accent-foreground"
                  >
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            </div>
            <div className="flex items-center gap-4">
              <span>Halaman {currentPage} dari {totalPages || 1}</span>
              <div className="flex gap-2">
                <Button variant="outline" size="icon" disabled={currentPage === 1}
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}>
                  <ChevronLeft size={16} />
                </Button>
                <Button variant="outline" size="icon" disabled={currentPage === totalPages}
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}>
                  <ChevronRight size={16} />
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Modal Tambah/Edit */}
      <LocationFormModal
        isOpen={formOpen}
        onClose={() => setFormOpen(false)}
        onSuccess={fetchData}
        location={editItem}
      />

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Lokasi</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus lokasi "{deleteItem?.site}"?
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
