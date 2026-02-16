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
import {
  PencilLine, Trash2, Plus, Search, ArrowUpDown, ChevronLeft, ChevronRight
} from "lucide-react"
import apiClient from "@/services/api"
import LocationFormModal, { type Location } from "@/components/LocationFormModal"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const pageSizeOptions = [10, 25, 50]
const ROOT_VALUE = "__ROOT__"

export default function LocationsPage() {
  const [list, setList] = useState<Location[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)

  const [formOpen, setFormOpen] = useState(false)
  const [editItem, setEditItem] = useState<Location | null>(null)

  // ✅ mode modal: parent vs normal (child)
  const [modalMode, setModalMode] = useState<"normal" | "parent">("normal")

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
      setList([])
      toast.error("Gagal memuat data lokasi.")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchData() }, [])

  // ================================
  // FILTER + SORT (flat)
  // ================================
  const filtered = useMemo(() => {
    const q = search.toLowerCase()

    let result = list.filter(l =>
      (l.site ?? "").toLowerCase().includes(q) ||
      (l.building ?? "").toLowerCase().includes(q) ||
      (l.room ?? "").toLowerCase().includes(q) ||
      (l.parent_name ?? "").toLowerCase().includes(q)
    )

    result.sort((a, b) => {
      const keyA = a.parent_name || a.site
      const keyB = b.parent_name || b.site
      return sortAsc ? keyA.localeCompare(keyB) : keyB.localeCompare(keyA)
    })

    return result
  }, [list, search, sortAsc])

  // ================================
  // PAGINATION (flat agar stabil)
  // ================================
  const totalPages = Math.ceil(filtered.length / pageSize)
  const paginated = filtered.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  )

  // ================================
  // GROUPING Parent -> Items
  // ================================
  const grouped = useMemo(() => {
    const map = new Map<string, Location[]>()

    for (const item of paginated) {
      const key = item.parent_name || item.site
      const arr = map.get(key) || []
      arr.push(item)
      map.set(key, arr)
    }

    return Array.from(map.entries()).map(([parent, items]) => ({
      parent,
      items
    }))
  }, [paginated])

  // ================================
  // ACTIONS
  // ================================
  const openAddChild = () => {
    setModalMode("normal")
    setEditItem(null)
    setFormOpen(true)
  }

  const openAddParent = () => {
    setModalMode("parent")
    setEditItem(null)
    setFormOpen(true)
  }

  const openEdit = (loc: Location) => {
    setModalMode("normal")
    setEditItem(loc)
    setFormOpen(true)
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/locations/${deleteItem.id}`)
      toast.success(`Lokasi "${deleteItem.site}" dinonaktifkan.`)
      setConfirmOpen(false)
      fetchData()
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal menghapus lokasi.")
    }
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
              placeholder="Cari site / building / room / parent..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setCurrentPage(1) }}
              className="w-[260px]"
            />

            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSortAsc(!sortAsc)}
              title="Urutkan berdasarkan parent/site"
            >
              <ArrowUpDown size={18} />
            </Button>

            {/* ✅ Tambah Parent (root) */}
            <Button variant="outline" onClick={openAddParent}>
              + Parent
            </Button>

            {/* ✅ Tambah Child (normal) */}
            <Button onClick={openAddChild} className="gap-1">
              <Plus size={16} /> Tambah
            </Button>
          </div>
        </CardHeader>

        <CardContent>
          <div className="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Lokasi</TableHead>
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
                ) : grouped.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center h-24 text-muted-foreground">
                      Tidak ada lokasi.
                    </TableCell>
                  </TableRow>
                ) : (
                  grouped.map((group) => (
                    <>
                      {/* Parent Header */}
                      <TableRow key={`parent-${group.parent}`} className="bg-muted/50">
                        <TableCell colSpan={5} className="font-semibold">
                          {group.parent}
                        </TableCell>
                      </TableRow>

                      {/* Children/Items */}
                      {group.items.map((l) => (
                        <TableRow key={l.id} className="hover:bg-accent/40 transition-all">
                          <TableCell className="pl-6">
                            {l.parent_name ? "↳ " : ""}
                            {l.site}
                          </TableCell>
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
                      ))}
                    </>
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
                  disabled={currentPage === totalPages || totalPages === 0}
                  onClick={() => setCurrentPage(p => Math.min(totalPages || 1, p + 1))}
                >
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
        defaultParentValue={ROOT_VALUE}
        lockParent={modalMode === "parent"}
      />

      {/* Konfirmasi Delete */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Nonaktifkan Lokasi</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menonaktifkan lokasi "{deleteItem?.site}"?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>
              Nonaktifkan
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
