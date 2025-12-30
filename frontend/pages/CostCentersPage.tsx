// File: src/pages/CostCentersPage.tsx
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
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import { Tooltip, TooltipProvider, TooltipTrigger, TooltipContent } from "@/components/ui/tooltip"
import { toast } from "sonner"
import { PencilLine, Trash2, Search, Plus, ArrowUpDown, ChevronLeft, ChevronRight } from "lucide-react"
import apiClient from "@/services/api"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

type CostCenter = {
  id: number
  code: string
  name: string
}

const pageSizeOptions = [10, 25, 50]

export default function CostCentersPage() {
  const [list, setList] = useState<CostCenter[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)

  // Dialogs
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)

  // Form state
  const [form, setForm] = useState({ code: "", name: "" })
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteItem, setDeleteItem] = useState<CostCenter | null>(null)

  // Load data
  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get("/cost-centers")
      const data = res.data?.data ?? res.data
      // pastikan selalu array
      setList(Array.isArray(data) ? data : [])
    } catch {
      toast.error("Gagal memuat cost centers.")
      setList([]) // fallback aman
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  // Filter + Sort + Pagination
  const filtered = useMemo(() => {
    const q = search.toLowerCase()
    const base = Array.isArray(list) ? list : []   // guard clause
    let result = base.filter(
      c => (c.name ?? "").toLowerCase().includes(q) ||
          (c.code ?? "").toLowerCase().includes(q)
    )
    result.sort((a, b) => sortAsc
      ? a.name.localeCompare(b.name)
      : b.name.localeCompare(a.name))
    return result
  }, [list, search, sortAsc])

  const totalPages = Math.ceil(filtered.length / pageSize)
  const paginated = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  // CRUD Handlers
  const handleSave = async () => {
    const { code, name } = form
    if (!code.trim() || !name.trim()) return toast.error("Kode dan nama wajib diisi.")

    try {
      if (editId) {
        await apiClient.put(`/cost-centers/${editId}`, { code, name })
        toast.success("Cost center diperbarui.")
      } else {
        await apiClient.post("/cost-centers", { code, name })
        toast.success("Cost center ditambahkan.")
      }
      setDialogOpen(false)
      setForm({ code: "", name: "" })
      setEditId(null)
      fetchData()
    } catch {
      toast.error("Gagal menyimpan cost center.")
    }
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/cost-centers/${deleteItem.id}`)
      toast.success(`Cost center "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchData()
    } catch {
      toast.error("Gagal menghapus cost center.")
    }
  }

  const openAdd = () => {
    setForm({ code: "", name: "" })
    setEditId(null)
    setDialogOpen(true)
  }

  const openEdit = (cc: CostCenter) => {
    setEditId(cc.id)
    setForm({ code: cc.code, name: cc.name })
    setDialogOpen(true)
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            💼 Manajemen Cost Center
          </CardTitle>

          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari kode atau nama cost center..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setCurrentPage(1) }}
              className="w-[240px]"
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
                  <TableHead>Kode</TableHead>
                  <TableHead>Nama</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={4} className="text-center h-24 text-muted-foreground">Memuat data...</TableCell></TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow><TableCell colSpan={4} className="text-center h-24 text-muted-foreground">Tidak ada cost center.</TableCell></TableRow>
                ) : (
                  paginated.map(cc => (
                    <TableRow key={cc.id} className="hover:bg-accent/40 transition-all">
                      <TableCell className="font-mono">{cc.id}</TableCell>
                      <TableCell className="font-medium">{cc.code}</TableCell>
                      <TableCell>{cc.name}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="outline" onClick={() => openEdit(cc)}>
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
                                  onClick={() => { setDeleteItem(cc); setConfirmOpen(true) }}
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
                <SelectTrigger className="w-[90px] text-sm">
                  <SelectValue placeholder="Size" />
                </SelectTrigger>
                <SelectContent>
                  {pageSizeOptions.map((size) => (
                    <SelectItem key={size} value={String(size)}>
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

      {/* Dialog Tambah/Edit */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Cost Center" : "Tambah Cost Center"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Input
              placeholder="Kode Cost Center"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
            />
            <Input
              placeholder="Nama Cost Center"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>Batal</Button>
            <Button onClick={handleSave}>Simpan</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Cost Center</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus cost center "{deleteItem?.name}"?
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
