import { useEffect, useMemo, useState } from "react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from "@/components/ui/table"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from "@/components/ui/dialog"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader,
  AlertDialogTitle, AlertDialogDescription, AlertDialogFooter,
  AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { toast } from "sonner"
import { PencilLine, Trash2, Plus, Search, ArrowUpDown, ChevronLeft, ChevronRight } from "lucide-react"
import apiClient from "@/services/api"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

type AssetType = {
  id: number
  name: string
}

const pageSizeOptions = [10, 25, 50]

export default function AssetTypesPage() {
  const [list, setList] = useState<AssetType[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)
  const [, setPage] = useState(1)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [formName, setFormName] = useState("")
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteItem, setDeleteItem] = useState<AssetType | null>(null)

  const toSafeArray = (raw: any): any[] => {
    if (Array.isArray(raw)) return raw
    if (Array.isArray(raw?.data)) return raw.data
    if (raw?.data === null) return []
    if (raw === null) return []
    return []
  }

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get("/asset-types")

      const data = toSafeArray(res.data)
      setList(data)
    } catch (err) {
      console.error("Failed to load asset types:", err)
      toast.error("Gagal memuat tipe aset.")
      setList([])
    } finally {
      setLoading(false)
    }
  }


  useEffect(() => { fetchData() }, [])

  // --- Filter + Sort + Pagination
  const filtered = useMemo(() => {
    const q = search.toLowerCase()
    let result = list.filter(a => a.name.toLowerCase().includes(q))
    result.sort((a, b) => sortAsc
      ? a.name.localeCompare(b.name)
      : b.name.localeCompare(a.name))
    return result
  }, [list, search, sortAsc])

  const totalPages = Math.ceil(filtered.length / pageSize)
  const paginated = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  // --- CRUD actions
  const handleSave = async () => {
    const name = formName.trim()
    if (!name) return toast.error("Nama tipe aset wajib diisi.")
    try {
      if (editId) {
        await apiClient.put(`/asset-types/${editId}`, { name })
        toast.success("Tipe aset diperbarui.")
      } else {
        await apiClient.post("/asset-types", { name })
        toast.success("Tipe aset ditambahkan.")
      }
      setDialogOpen(false)
      setFormName("")
      setEditId(null)
      fetchData()
    } catch {
      toast.error("Gagal menyimpan tipe aset.")
    }
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/asset-types/${deleteItem.id}`)
      toast.success(`Tipe aset "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchData()
    } catch {
      toast.error("Gagal menghapus tipe aset.")
    }
  }

  const openAdd = () => {
    setFormName("")
    setEditId(null)
    setDialogOpen(true)
  }

  const openEdit = (a: AssetType) => {
    setEditId(a.id)
    setFormName(a.name)
    setDialogOpen(true)
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            🧱 Manajemen Tipe Aset
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari nama tipe aset..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setCurrentPage(1) }}
              className="w-[220px]"
            />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSortAsc(!sortAsc)}
              title="Urutkan nama"
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
                  <TableHead>ID</TableHead>
                  <TableHead>Nama</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center h-24 text-muted-foreground">
                      Memuat data...
                    </TableCell>
                  </TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center h-24 text-muted-foreground">
                      Tidak ada tipe aset.
                    </TableCell>
                  </TableRow>
                ) : (
                  paginated.map(a => (
                    <TableRow key={a.id} className="hover:bg-accent/40 transition-all">
                      <TableCell className="font-mono">{a.id}</TableCell>
                      <TableCell>{a.name}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="outline" onClick={() => openEdit(a)}>
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
                                  onClick={() => { setDeleteItem(a); setConfirmOpen(true) }}
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
    setPage(1)
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
                <Button
                  variant="outline"
                  size="icon"
                  disabled={currentPage === 1}
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                >
                  <ChevronLeft size={16} />
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  disabled={currentPage === totalPages}
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                >
                  <ChevronRight size={16} />
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Dialog Tambah/Edit */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Tipe Aset" : "Tambah Tipe Aset"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Input
              placeholder="Nama tipe aset"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
            />
          </div>
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setDialogOpen(false)}>Batal</Button>
            <Button onClick={handleSave}>Simpan</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Tipe Aset</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus "{deleteItem?.name}"?
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
