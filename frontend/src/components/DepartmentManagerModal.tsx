// File: src/components/DepartmentManagerModal.tsx
import { useEffect, useMemo, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
} from "@/components/ui/dialog"
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Table, TableBody, TableCell, TableRow } from "@/components/ui/table"

type Department = {
  id: number
  name: string
  manager_id?: number | null
  manager_name?: string | null
  cost_center_id?: number | null
  cost_center_name?: string | null
}

type Employee = { id: number; name: string; email: string }
type CostCenter = { id: number; code: string; name: string }

function toArray<T>(resData: any): T[] {
  if (Array.isArray(resData)) return resData
  if (Array.isArray(resData?.data)) return resData.data
  return []
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onChanged?: () => void
}

export default function DepartmentManagerModal({ isOpen, onClose, onChanged }: Props) {
  const [departments, setDepartments] = useState<Department[]>([])
  const [employees, setEmployees] = useState<Employee[]>([])
  const [costCenters, setCostCenters] = useState<CostCenter[]>([])

  const [search, setSearch] = useState("")
  const [newDeptName, setNewDeptName] = useState("")
  const [newManagerId, setNewManagerId] = useState<string>("")
  const [newCostCenterId, setNewCostCenterId] = useState<string>("")

  const [editingId, setEditingId] = useState<number | null>(null)
  const [editingName, setEditingName] = useState("")
  const [editingManagerId, setEditingManagerId] = useState<string>("")
  const [editingCostCenterId, setEditingCostCenterId] = useState<string>("")

  const [confirmId, setConfirmId] = useState<number | null>(null)

  // ---- Fetch data ----
  const fetchDepartments = async () => {
    try {
      const res = await apiClient.get("/departments")
      setDepartments(toArray<Department>(res.data))
    } catch {
      toast.error("Gagal memuat daftar departemen.")
    }
  }

  const fetchEmployees = async () => {
    try {
      const res = await apiClient.get("/employees")
      setEmployees(toArray<Employee>(res.data))
    } catch {
      toast.error("Gagal memuat daftar karyawan.")
    }
  }

  const fetchCostCenters = async () => {
    try {
      const res = await apiClient.get("/cost-centers")
      setCostCenters(toArray<CostCenter>(res.data))
    } catch {
      toast.error("Gagal memuat daftar cost center.")
    }
  }

  useEffect(() => {
    if (isOpen) {
      setSearch("")
      setNewDeptName("")
      setNewManagerId("")
      setNewCostCenterId("")
      setEditingId(null)
      setEditingName("")
      setEditingManagerId("")
      setEditingCostCenterId("")
      setConfirmId(null)
      fetchDepartments()
      fetchEmployees()
      fetchCostCenters()
    }
  }, [isOpen])

  // ---- Filter ----
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return departments
    return departments.filter(d =>
      String(d.id).includes(q) ||
      d.name.toLowerCase().includes(q) ||
      d.manager_name?.toLowerCase().includes(q) ||
      d.cost_center_name?.toLowerCase().includes(q)
    )
  }, [departments, search])

  // ---- Add ----
  const handleAdd = async () => {
    const name = newDeptName.trim()
    if (!name) return toast.error("Nama departemen wajib diisi.")
    const payload = {
      name,
      manager_id: newManagerId ? Number(newManagerId) : null,
      cost_center_id: newCostCenterId ? Number(newCostCenterId) : null,
    }
    const promise = apiClient.post("/departments", payload)
    toast.promise(promise, {
      loading: "Menambahkan...",
      success: () => {
        setNewDeptName("")
        setNewManagerId("")
        setNewCostCenterId("")
        fetchDepartments()
        onChanged?.()
        return "Departemen ditambahkan."
      },
      error: (err) => err?.response?.data?.error || "Gagal menambahkan departemen.",
    })
  }

  // ---- Edit ----
  const startEdit = (d: Department) => {
    setEditingId(d.id)
    setEditingName(d.name)
    setEditingManagerId(d.manager_id ? String(d.manager_id) : "")
    setEditingCostCenterId(d.cost_center_id ? String(d.cost_center_id) : "")
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditingName("")
    setEditingManagerId("")
    setEditingCostCenterId("")
  }

  const submitEdit = async () => {
    if (editingId == null) return
    const name = editingName.trim()
    if (!name) return toast.error("Nama departemen wajib diisi.")
    const payload = {
      name,
      manager_id: editingManagerId ? Number(editingManagerId) : null,
      cost_center_id: editingCostCenterId ? Number(editingCostCenterId) : null,
    }
    const promise = apiClient.put(`/departments/${editingId}`, payload)
    toast.promise(promise, {
      loading: "Menyimpan...",
      success: () => {
        cancelEdit()
        fetchDepartments()
        onChanged?.()
        return "Departemen diperbarui!"
      },
      error: (err) => err?.response?.data?.error || "Gagal memperbarui departemen.",
    })
  }

  // ---- Delete ----
  const confirmDelete = (id: number) => setConfirmId(id)
  const doDelete = async () => {
    if (confirmId == null) return
    const id = confirmId
    setConfirmId(null)
    const promise = apiClient.delete(`/departments/${id}`)
    toast.promise(promise, {
      loading: "Menghapus...",
      success: () => {
        fetchDepartments()
        onChanged?.()
        return "Departemen dihapus."
      },
      error: (err) => err?.response?.data?.error || "Gagal menghapus departemen.",
    })
  }

  // ---- UI ----
  return (
    <>
      <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
        <DialogContent className="sm:max-w-[720px]">
          <DialogHeader>
            <DialogTitle>Manajemen Departemen</DialogTitle>
            <DialogDescription>
              Tambah, ubah nama/manager/cost center, atau hapus departemen.
            </DialogDescription>
          </DialogHeader>

          {/* Add */}
          <div className="grid sm:grid-cols-4 gap-2">
            <Input
              placeholder="Nama departemen..."
              value={newDeptName}
              onChange={(e) => setNewDeptName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleAdd() }}
            />
            <Select
              value={newManagerId || "none"}
              onValueChange={(v) => setNewManagerId(v === "none" ? "" : v)}
            >
              <SelectTrigger><SelectValue placeholder="Pilih Manager" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="none">- Tanpa Manager -</SelectItem>
                {employees.map(emp => (
                  <SelectItem key={emp.id} value={String(emp.id)}>{emp.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={newCostCenterId || ""}
              onValueChange={(v) => setNewCostCenterId(v)}
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
            <Button onClick={handleAdd}>Tambah</Button>
          </div>

          {/* Search */}
          <Input
            placeholder="Cari nama / manager / cost center…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="mt-3"
          />

          {/* List */}
          <div className="mt-4 max-h-80 overflow-y-auto rounded-md border">
            <Table>
              <TableBody>
                {filtered.map((d) => {
                  const editing = editingId === d.id
                  return (
                    <TableRow key={d.id}>
                      <TableCell className="w-16 text-xs text-muted-foreground font-mono">#{d.id}</TableCell>
                      <TableCell>
                        {editing ? (
                          <div className="grid grid-cols-3 gap-2">
                            <Input
                              value={editingName}
                              onChange={(e) => setEditingName(e.target.value)}
                              placeholder="Nama departemen"
                              autoFocus
                            />
                            <Select
                              value={editingManagerId || "none"}
                              onValueChange={(v) => setEditingManagerId(v === "none" ? "" : v)}
                            >
                              <SelectTrigger><SelectValue placeholder="Pilih Manager" /></SelectTrigger>
                              <SelectContent>
                                <SelectItem value="none">- Tanpa Manager -</SelectItem>
                                {employees.map(emp => (
                                  <SelectItem key={emp.id} value={String(emp.id)}>{emp.name}</SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <Select
                              value={editingCostCenterId || ""}
                              onValueChange={(v) => setEditingCostCenterId(v)}
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
                        ) : (
                          <div>
                            <div className="font-medium">{d.name}</div>
                            <div className="text-xs text-muted-foreground">
                              {d.manager_name || "-"} · {d.cost_center_name || "-"}
                            </div>
                          </div>
                        )}
                      </TableCell>
                      <TableCell className="w-[210px] text-right space-x-2">
                        {editing ? (
                          <>
                            <Button size="sm" variant="outline" onClick={cancelEdit}>Batal</Button>
                            <Button size="sm" onClick={submitEdit}>Simpan</Button>
                          </>
                        ) : (
                          <>
                            <Button size="sm" variant="outline" onClick={() => startEdit(d)}>Edit</Button>
                            <Button size="sm" variant="destructive" onClick={() => confirmDelete(d.id)}>Hapus</Button>
                          </>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
                {filtered.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} className="h-20 text-center text-muted-foreground">
                      Tidak ada data departemen.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirm Delete */}
      <AlertDialog open={confirmId !== null} onOpenChange={(open) => { if (!open) setConfirmId(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Departemen</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini tidak dapat dibatalkan. Yakin menghapus?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={doDelete}>Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
