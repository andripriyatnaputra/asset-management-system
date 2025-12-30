// File: src/pages/ContractsPage.tsx
import { useEffect, useMemo, useRef, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow
} from "@/components/ui/table"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel,
  AlertDialogContent, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle
} from "@/components/ui/alert-dialog"
import {
  Tooltip, TooltipProvider, TooltipTrigger, TooltipContent
} from "@/components/ui/tooltip"
import { Badge } from "@/components/ui/badge"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import {
  Plus, Search, Edit3, Trash2, FileText, ChevronLeft, ChevronRight, ArrowUpDown
} from "lucide-react"
import ContractFormModal from "@/components/ContractFormModal"
import ContractLicensesModal from "@/components/ContractLicensesModal"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface Contract {
  id: number
  contract_number: string
  vendor?: string
  contract_type?: string
  start_date?: string
  end_date?: string
  total_value?: number
  currency?: string
  status?: string
}

type SortKey = "id" | "contract_number" | "vendor" | "start_date"
type SortDir = "asc" | "desc"

const pageSizeOptions = [10, 25, 50]

export default function ContractsPage() {
  const [contracts, setContracts] = useState<Contract[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [sortKey, setSortKey] = useState<SortKey>("id")
  const [sortDir, setSortDir] = useState<SortDir>("asc")
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editing, setEditing] = useState<Contract | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [showLicenses, setShowLicenses] = useState(false)
  const [selectedContract, setSelectedContract] = useState<number | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Fetch contracts
  const fetchContracts = async (q = "") => {
    setIsLoading(true)
    try {
      const res = await apiClient.get("/contracts", { params: { q } })
      const data = Array.isArray(res.data)
        ? res.data
        : Array.isArray(res.data?.data)
        ? res.data.data
        : []
      setContracts(data)
    } catch (err) {
      console.error("Failed to load contracts:", err)
      toast.error("Gagal memuat data kontrak.")
      setContracts([])
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => { fetchContracts() }, [])

  // 🔍 Debounced search
  const onChangeSearch = (val: string) => {
    setSearch(val)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => { setPage(1); fetchContracts(val) }, 400)
  }

  // 🔁 Sorting
  const toggleSort = (key: SortKey) => {
    if (key === sortKey) setSortDir(d => (d === "asc" ? "desc" : "asc"))
    else { setSortKey(key); setSortDir("asc") }
  }

  // 🧮 Sorting + Pagination
  const sorted = useMemo(() => {
    const list = [...contracts].sort((a, b) => {
      let av: any, bv: any
      switch (sortKey) {
        case "contract_number": av = a.contract_number; bv = b.contract_number; break
        case "vendor": av = (a.vendor || "").toLowerCase(); bv = (b.vendor || "").toLowerCase(); break
        case "start_date": av = new Date(a.start_date || 0).getTime(); bv = new Date(b.start_date || 0).getTime(); break
        default: av = a.id; bv = b.id
      }
      if (av < bv) return sortDir === "asc" ? -1 : 1
      if (av > bv) return sortDir === "asc" ? 1 : -1
      return 0
    })
    return list
  }, [contracts, sortKey, sortDir])

  const totalPages = Math.ceil(sorted.length / pageSize)
  const paginated = sorted.slice((page - 1) * pageSize, page * pageSize)

  // 🧱 CRUD ops
  const openAdd = () => { setEditing(null); setIsModalOpen(true) }
  const openEdit = (c: Contract) => { setEditing(c); setIsModalOpen(true) }
  const closeModal = () => { setIsModalOpen(false); setEditing(null) }

  const handleDelete = async () => {
    if (deleteId == null) return
    const id = deleteId
    setDeleteId(null)
    toast.promise(apiClient.delete(`/contracts/${id}`), {
      loading: "Menghapus kontrak…",
      success: () => { fetchContracts(); return "Kontrak dihapus." },
      error: (err) => err?.response?.data?.error || "Gagal menghapus kontrak.",
    })
  }

  const getStatusBadge = (status?: string) => {
    const s = (status || "").toLowerCase()
    if (s === "active") return <Badge className="bg-green-500/10 text-green-700 border-green-400">Aktif</Badge>
    if (s === "expired") return <Badge className="bg-red-500/10 text-red-700 border-red-400">Kedaluwarsa</Badge>
    if (s === "pending") return <Badge className="bg-yellow-500/10 text-yellow-700 border-yellow-400">Menunggu</Badge>
    return <Badge variant="outline">-</Badge>
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        {/* Header */}
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            <FileText size={18} /> Manajemen Kontrak
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari nomor / vendor…"
              value={search}
              onChange={(e) => onChangeSearch(e.target.value)}
              className="w-[240px]"
            />
            <Button
              variant="ghost"
              size="sm"
              onClick={() => toggleSort("contract_number")}
              title="Urutkan"
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
                  <TableHead onClick={() => toggleSort("id")} className="cursor-pointer select-none">
                    ID {sortKey === "id" ? (sortDir === "asc" ? "↑" : "↓") : ""}
                  </TableHead>
                  <TableHead onClick={() => toggleSort("contract_number")} className="cursor-pointer select-none">
                    Nomor Kontrak {sortKey === "contract_number" ? (sortDir === "asc" ? "↑" : "↓") : ""}
                  </TableHead>
                  <TableHead onClick={() => toggleSort("vendor")} className="cursor-pointer select-none">
                    Vendor {sortKey === "vendor" ? (sortDir === "asc" ? "↑" : "↓") : ""}
                  </TableHead>
                  <TableHead>Tipe</TableHead>
                  <TableHead onClick={() => toggleSort("start_date")} className="cursor-pointer select-none">
                    Periode {sortKey === "start_date" ? (sortDir === "asc" ? "↑" : "↓") : ""}
                  </TableHead>
                  <TableHead>Nilai</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {isLoading ? (
                  <TableRow><TableCell colSpan={8} className="text-center h-24 text-muted-foreground">Memuat data…</TableCell></TableRow>
                ) : paginated.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-center h-24 text-muted-foreground">Tidak ada kontrak.</TableCell></TableRow>
                ) : (
                  paginated.map((c) => (
                    <TableRow key={c.id} className="hover:bg-accent/40 transition-all">
                      <TableCell>{c.id}</TableCell>
                      <TableCell>{c.contract_number}</TableCell>
                      <TableCell>{c.vendor || "-"}</TableCell>
                      <TableCell>{c.contract_type || "-"}</TableCell>
                      <TableCell>
                        {c.start_date ? new Date(c.start_date).toLocaleDateString("id-ID") : "-"} 
                        {" - "}
                        {c.end_date ? new Date(c.end_date).toLocaleDateString("id-ID") : "-"}
                      </TableCell>
                      <TableCell>
                        {typeof c.total_value === "number"
                          ? c.total_value.toLocaleString("id-ID", { style: "currency", currency: c.currency || "IDR" })
                          : "-"}
                      </TableCell>
                      <TableCell>{getStatusBadge(c.status)}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="secondary"
                                  size="sm"
                                  onClick={() => { setSelectedContract(c.id); setShowLicenses(true) }}
                                >
                                  Lihat Lisensi
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Daftar lisensi kontrak</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="outline" size="sm" onClick={() => openEdit(c)}>
                                  <Edit3 size={16} className="mr-1" /> Edit
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Kontrak</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="destructive" size="sm" onClick={() => setDeleteId(c.id)}>
                                  <Trash2 size={16} className="mr-1" /> Hapus
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Kontrak</TooltipContent>
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
                    <SelectItem key={size} value={String(size)}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <span>Halaman {page} dari {totalPages || 1}</span>
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
      <ContractFormModal
        isOpen={isModalOpen}
        onClose={closeModal}
        onSuccess={() => { closeModal(); fetchContracts() }}
        contract={editing}
      />

      <ContractLicensesModal
        isOpen={showLicenses}
        onClose={() => setShowLicenses(false)}
        contractId={selectedContract}
      />

      {/* Dialog Hapus */}
      <AlertDialog open={deleteId !== null} onOpenChange={(open) => { if (!open) setDeleteId(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Kontrak</AlertDialogTitle>
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
