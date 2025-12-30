import { useEffect, useRef, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Tooltip, TooltipProvider, TooltipTrigger, TooltipContent
} from "@/components/ui/tooltip"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import EmployeeFormModal from "@/components/EmployeeFormModal"
import type { EmployeeDTO } from "@/components/EmployeeFormModal"
import { Plus, PencilLine, Trash2, Upload, Search, BookOpen, ChevronLeft, ChevronRight } from "lucide-react"
import { useNavigate } from "react-router-dom"

type RoleType = "super_admin" | "asset_manager" | "it_support" | "finance" | "employee"

interface EmployeeRow {
  id: number
  employee_nik: string
  name: string
  email: string
  department_id: number | null
  department_name?: string
  role: RoleType

  last_login_at?: string | null
  active_assets?: number
  delegation_count?: number
  employee_health_score?: number

  created_at?: string | null
  updated_at?: string | null
}


const ROLE_LABELS: Record<RoleType, string> = {
  super_admin: "Super Admin",
  asset_manager: "Asset Manager",
  it_support: "IT Support",
  finance: "Finance",
  employee: "Employee",
}

const pageSizeOptions = [10, 25, 50]

export default function EmployeesPage() {
  const [rows, setRows] = useState<EmployeeRow[]>([])
  const [q, setQ] = useState("")
  const [loading, setLoading] = useState(false)
  const [openForm, setOpenForm] = useState(false)
  const [editing, setEditing] = useState<EmployeeDTO | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalPages, setTotalPages] = useState(1)
  const [totalRecords, setTotalRecords] = useState(0)
  const [sortKey, setSortKey] = useState("name")
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc")
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<EmployeeRow | null>(null)
  const fileRef = useRef<HTMLInputElement | null>(null)
  const [importing, setImporting] = useState(false)
  const navigate = useNavigate()

  const fetchEmployees = async (searchQ = q) => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      params.set("page", String(page))
      params.set("limit", String(pageSize))
      params.set("sort_by", sortKey)
      params.set("sort_dir", sortDir)
      if (searchQ.trim()) params.set("q", searchQ.trim())

      const res = await apiClient.get(`/employees?${params.toString()}`)
      const list = res.data?.data ?? res.data ?? []
      const pagination = res.data?.pagination ?? {}
      setRows(Array.isArray(list) ? list : [])
      setTotalPages(pagination.total_pages ?? 1)
      setTotalRecords(pagination.total_records ?? list.length)
    } catch {
      toast.error("Gagal memuat data karyawan.")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchEmployees() }, [page, pageSize, sortKey, sortDir])

  const toggleSort = (key: string) => {
    if (key === sortKey) setSortDir(d => (d === "asc" ? "desc" : "asc"))
    else { setSortKey(key); setSortDir("asc") }
  }

  // CRUD
  const handleAdd = () => { setEditing(null); setOpenForm(true) }
  const handleEdit = (r: EmployeeRow) => {
    setEditing({
      id: r.id,
      employee_nik: r.employee_nik,
      name: r.name,
      email: r.email,
      department_id: r.department_id,
      role: r.role,
    })
    setOpenForm(true)
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/employees/${deleteItem.id}`)
      toast.success(`Karyawan "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchEmployees()
    } catch (e: any) {
      toast.error(e?.response?.data?.error || "Gagal menghapus karyawan.")
    }
  }

  // Import CSV
  const onPickFile: React.ChangeEventHandler<HTMLInputElement> = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    setImporting(true)
    try {
      const fd = new FormData()
      fd.append("file", file)
      await apiClient.post("/employees/import", fd, {
        headers: { "Content-Type": "multipart/form-data" },
      })
      toast.success("Import karyawan berhasil.")
      await fetchEmployees()
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal import CSV.")
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ""
    }
  }

  const roleBadgeVariant = (role: RoleType) => {
    switch (role) {
      case "super_admin": return "destructive"
      case "finance": return "secondary"
      case "asset_manager": return "default"
      case "it_support": return "outline"
      default: return "default"
    }
  }

  const formatDate = (val?: string | null) => {
    if (!val) return "-"
    const d = new Date(val)
    return isNaN(d.getTime()) ? "-" : d.toLocaleString("id-ID", {
      dateStyle: "short",
      timeStyle: "short",
    })
  }

  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <div>
            <CardTitle className="text-lg font-semibold">👥 Manajemen Karyawan</CardTitle>
            <p className="text-sm text-muted-foreground mt-1">
              Kelola data karyawan, role, departemen, dan training.
            </p>
          </div>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <input ref={fileRef} type="file" accept=".csv" className="hidden" onChange={onPickFile}/>
            <Button variant="outline" onClick={() => fileRef.current?.click()} disabled={importing}>
              <Upload size={16} className="mr-2" /> {importing ? "Mengunggah..." : "Import CSV"}
            </Button>
            <Button onClick={handleAdd} className="gap-1">
              <Plus size={16} /> Tambah
            </Button>
          </div>
        </CardHeader>

        <CardContent>
          {/* Search + page size */}
          <div className="flex flex-col md:flex-row items-center justify-between gap-3 mb-4">
            <div className="relative w-full md:w-1/2">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground"/>
              <Input
                placeholder="Cari nama, NIK, email, atau departemen..."
                value={q}
                onChange={(e) => { setQ(e.target.value); setPage(1); fetchEmployees(e.target.value) }}
                className="pl-9"
              />
            </div>
            <div className="flex items-center gap-2">
              <span>Baris per halaman:</span>
              <Select
                value={String(pageSize)}
                onValueChange={(v) => { setPageSize(Number(v)); setPage(1) }}
              >
                <SelectTrigger className="w-[90px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {pageSizeOptions.map(size => (
                    <SelectItem key={size} value={String(size)}>{size}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Table */}
          <div className="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead onClick={() => toggleSort("employee_nik")} className="cursor-pointer">NIK</TableHead>
                  <TableHead onClick={() => toggleSort("name")} className="cursor-pointer">Nama</TableHead>
                  <TableHead onClick={() => toggleSort("email")} className="cursor-pointer">Email</TableHead>
                  <TableHead onClick={() => toggleSort("department_name")} className="cursor-pointer">Departemen</TableHead>
                  <TableHead onClick={() => toggleSort("role")} className="cursor-pointer">Role</TableHead>
                  <TableHead className="text-center">Aset Aktif</TableHead>
                  <TableHead className="text-center">Delegasi</TableHead>
                  <TableHead className="text-center">Skor Aktivitas</TableHead>
                  <TableHead>Dibuat</TableHead>
                  <TableHead>Diperbarui</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow><TableCell colSpan={8} className="text-center h-24 text-muted-foreground">Memuat data...</TableCell></TableRow>
                ) : rows.length === 0 ? (
                  <TableRow><TableCell colSpan={8} className="text-center h-24 text-muted-foreground">Tidak ada data.</TableCell></TableRow>
                ) : (
                  rows.map((r) => (
                    <TableRow key={r.id} className="hover:bg-accent/40 transition-all">
                      <TableCell className="font-mono text-sm">{r.employee_nik}</TableCell>
                      <TableCell>{r.name}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">{r.email}</TableCell>
                      <TableCell>{r.department_name || "-"}</TableCell>
                      <TableCell><Badge variant={roleBadgeVariant(r.role)}>{ROLE_LABELS[r.role]}</Badge></TableCell>
                      <TableCell className="text-center">{r.active_assets}</TableCell>
                      <TableCell className="text-center">{r.delegation_count}</TableCell>
                      <TableCell className="text-center">
                        {(() => {
                          const score = r.employee_health_score ?? 0
                          return (
                            <Badge
                              variant={score > 90 ? "default" : score > 70 ? "secondary" : "destructive"}
                              className={
                                score > 90
                                  ? "bg-green-500/10 text-green-700 border-green-400"
                                  : score > 70
                                  ? "bg-yellow-500/10 text-yellow-700 border-yellow-400"
                                  : "bg-red-500/10 text-red-700 border-red-400"
                              }
                            >
                              {score.toFixed(1)}%
                            </Badge>
                          )
                        })()}
                      </TableCell>
                      <TableCell>{formatDate(r.created_at)}</TableCell>
                      <TableCell>{formatDate(r.updated_at)}</TableCell>
                      <TableCell className="text-right">
                        <TooltipProvider>
                          <div className="flex justify-end gap-2">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="outline" size="icon" onClick={() => handleEdit(r)}>
                                  <PencilLine size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Karyawan</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="outline" size="icon" onClick={() => navigate(`/employees/${r.id}/trainings`)}>
                                  <BookOpen size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Pelatihan</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="destructive" size="icon" onClick={() => { setDeleteItem(r); setConfirmOpen(true) }}>
                                  <Trash2 size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Karyawan</TooltipContent>
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
          {!loading && totalRecords > 0 && (
            <div className="flex flex-col md:flex-row items-center justify-between mt-4 text-sm gap-3">
              <p>{(page - 1) * pageSize + 1}–{Math.min(page * pageSize, totalRecords)} dari {totalRecords}</p>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="icon" disabled={page <= 1}
                  onClick={() => setPage(p => Math.max(1, p - 1))}>
                  <ChevronLeft size={16} />
                </Button>
                <span>Halaman {page} dari {totalPages}</span>
                <Button variant="outline" size="icon" disabled={page >= totalPages}
                  onClick={() => setPage(p => Math.min(totalPages, p + 1))}>
                  <ChevronRight size={16} />
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Modal Form */}
      <EmployeeFormModal
        isOpen={openForm}
        onClose={() => setOpenForm(false)}
        onSuccess={fetchEmployees}
        employee={editing}
      />

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Karyawan</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus karyawan "<strong>{deleteItem?.name}</strong>"?
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
