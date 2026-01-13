// File: src/pages/DepartmentsPage.tsx
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

type Department = {
  id: number
  name: string
  manager_name?: string | null
  cost_center_name?: string | null
  cost_center_id?: number | null
}

type CostCenter = {
  id: number
  code: string
  name: string
}

const pageSizeOptions = [10, 25, 50]

export default function DepartmentsPage() {
  const [list, setList] = useState<Department[]>([])
  const [costCenters, setCostCenters] = useState<CostCenter[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)
  const [, setPage] = useState(1)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [form, setForm] = useState({ name: "", manager_name: "", cost_center_id: "" })
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteItem, setDeleteItem] = useState<Department | null>(null)

  // Fetch data
  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await apiClient.get("/departments")

      const raw = res.data?.data ?? res.data

      const data: Department[] = Array.isArray(raw)
        ? raw
        : Array.isArray(raw?.data)
          ? raw.data
          : []

      setList(data)
    } catch {
      setList([]) // ← penting
      toast.error("Gagal memuat departemen.")
    } finally {
      setLoading(false)
    }
  }


  const fetchCostCenters = async () => {
    try {
      const res = await apiClient.get("/cost-centers")
      setCostCenters(res.data?.data ?? [])
    } catch {
      toast.error("Gagal memuat cost center.")
    }
  }

  useEffect(() => {
    fetchData()
    fetchCostCenters()
  }, [])

  // Filter + Sort + Pagination
  const filtered = useMemo(() => {
    const q = search.toLowerCase()
    let result = list.filter(
      d =>
        d.name.toLowerCase().includes(q) ||
        d.manager_name?.toLowerCase().includes(q) ||
        d.cost_center_name?.toLowerCase().includes(q)
    )
    result.sort((a, b) =>
      sortAsc ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name)
    )
    return result
  }, [list, search, sortAsc])

  const totalPages = Math.ceil(filtered.length / pageSize)
  const paginated = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize)

  // CRUD Handlers
  const handleSave = async () => {
    const { name, manager_name, cost_center_id } = form
    if (!name.trim()) return toast.error("Nama departemen wajib diisi.")
    try {
      const payload = {
        name,
        manager_name: manager_name?.trim() || null,
        cost_center_id: cost_center_id ? Number(cost_center_id) : null,
      }

      if (editId) {
        await apiClient.put(`/departments/${editId}`, payload)
        toast.success("Departemen diperbarui.")
      } else {
        await apiClient.post("/departments", payload)
        toast.success("Departemen ditambahkan.")
      }
      setDialogOpen(false)
      setForm({ name: "", manager_name: "", cost_center_id: "" })
      setEditId(null)
      fetchData()
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal menyimpan data departemen.")
    }
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/departments/${deleteItem.id}`)
      toast.success(`Departemen "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchData()
    } catch {
      toast.error("Gagal menghapus departemen.")
    }
  }

  const openAdd = () => {
    setForm({ name: "", manager_name: "", cost_center_id: "" })
    setEditId(null)
    setDialogOpen(true)
  }

  const openEdit = (d: Department) => {
    setEditId(d.id)
    setForm({
      name: d.name,
      manager_name: d.manager_name || "",
      cost_center_id: d.cost_center_id ? String(d.cost_center_id) : "",
    })
    setDialogOpen(true)
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            🏢 Manajemen Departemen
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari nama / manager / cost center..."
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
                  <TableHead>Nama</TableHead>
                  <TableHead>Manager</TableHead>
                  <TableHead>Cost Center</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={5} className="text-center h-24 text-muted-foreground">Memuat data...</TableCell></TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow><TableCell colSpan={5} className="text-center h-24 text-muted-foreground">Tidak ada departemen.</TableCell></TableRow>
                ) : (
                  paginated.map(d => (
                    <TableRow key={d.id} className="hover:bg-accent/40 transition-all">
                      <TableCell className="font-mono">{d.id}</TableCell>
                      <TableCell>{d.name}</TableCell>
                      <TableCell>{d.manager_name || "-"}</TableCell>
                      <TableCell>{d.cost_center_name || "-"}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="outline" onClick={() => openEdit(d)}>
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
                                  onClick={() => { setDeleteItem(d); setConfirmOpen(true) }}
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
                <SelectTrigger className="w-[90px] text-sm">
                  <SelectValue placeholder="Size" />
                </SelectTrigger>
                <SelectContent>
                  {pageSizeOptions.map((size) => (
                    <SelectItem key={size} value={String(size)}>{size}</SelectItem>
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
            <DialogTitle>{editId ? "Edit Departemen" : "Tambah Departemen"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2">
            <Input
              placeholder="Nama Departemen"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
            <Input
              placeholder="Nama Manager (opsional)"
              value={form.manager_name}
              onChange={(e) => setForm({ ...form, manager_name: e.target.value })}
            />
            <Select
              value={form.cost_center_id || ""}
              onValueChange={(v) => setForm({ ...form, cost_center_id: v })}
            >
              <SelectTrigger><SelectValue placeholder="Pilih Cost Center" /></SelectTrigger>
              <SelectContent>
                {costCenters.map(cc => (
                  <SelectItem key={cc.id} value={String(cc.id)}>
                    {cc.code} — {cc.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
            <AlertDialogTitle>Hapus Departemen</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus departemen "{deleteItem?.name}"?
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
