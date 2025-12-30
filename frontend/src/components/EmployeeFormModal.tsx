import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel
} from "@/components/ui/alert-dialog"
import { BookOpen } from "lucide-react"

type RoleType = "super_admin" | "asset_manager" | "it_support" | "finance" | "employee"

export interface EmployeeDTO {
  id?: number
  employee_nik: string
  name: string
  email: string
  department_id: number | null
  role: RoleType
}

interface Department { id: number; name: string }

interface EmployeeFormModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  employee: EmployeeDTO | null
}

const ROLE_OPTIONS: { value: RoleType; label: string }[] = [
  { value: "super_admin", label: "Super Admin" },
  { value: "asset_manager", label: "Asset Manager" },
  { value: "it_support", label: "IT Support" },
  { value: "finance", label: "Finance" },
  { value: "employee", label: "Employee" },
]

export default function EmployeeFormModal({
  isOpen, onClose, onSuccess, employee,
}: EmployeeFormModalProps) {
  const isEdit = !!employee
  const [departments, setDepartments] = useState<Department[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingDepts, setLoadingDepts] = useState(false)
  const [resetPassword, setResetPassword] = useState<string | null>(null)

  const [form, setForm] = useState({
    employee_nik: "",
    name: "",
    email: "",
    department_id: "" as string,
    role: "employee" as RoleType,
    password: "",
  })

  const [errors, setErrors] = useState<{ [k: string]: string }>({})

  // ---- Load Departments ----
  useEffect(() => {
    if (!isOpen) return
    setLoadingDepts(true)
    let active = true
    apiClient
      .get("/departments")
      .then((res) => {
        if (!active) return
        const data = Array.isArray(res.data) ? res.data : res.data?.data ?? []
        setDepartments(data)
      })
      .catch(() => toast.error("Gagal memuat daftar departemen"))
      .finally(() => setLoadingDepts(false))
    return () => {
      active = false
    }
  }, [isOpen])

  // ---- Prefill Edit Form ----
  useEffect(() => {
    if (!isOpen) return
    if (employee) {
      setForm({
        employee_nik: employee.employee_nik || "",
        name: employee.name || "",
        email: employee.email || "",
        department_id: employee.department_id != null ? String(employee.department_id) : "",
        role: employee.role || "employee",
        password: "",
      })
    } else {
      setForm({
        employee_nik: "",
        name: "",
        email: "",
        department_id: "",
        role: "employee",
        password: "",
      })
    }
  }, [isOpen, employee])

  // ---- Realtime Validation ----
  useEffect(() => {
    const newErr: { [k: string]: string } = {}

    if (!form.name.trim()) newErr.name = "Nama wajib diisi"
    if (!form.email.trim()) newErr.email = "Email wajib diisi"
    else if (!/\S+@\S+\.\S+/.test(form.email)) newErr.email = "Format email tidak valid"
    if (!isEdit && form.password.trim().length < 8)
      newErr.password = "Password minimal 8 karakter"
    setErrors(newErr)
  }, [form, isEdit])

  const disableSubmit = Object.keys(errors).length > 0 || loading

  const handleChange =
    (key: keyof typeof form) =>
    (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm((p) => ({ ...p, [key]: e.target.value }))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (disableSubmit || loading) return
    setLoading(true)
    try {
      if (isEdit && employee?.id) {
        const payload: any = {
          name: form.name.trim(),
          email: form.email.trim(),
          department_id: form.department_id ? Number(form.department_id) : null,
          role: form.role,
        }
        if (form.password.trim()) payload.password = form.password.trim()

        await apiClient.put(`/employees/${employee.id}`, payload)
        toast.success("Karyawan diperbarui.")
      } else {
        const payload = {
          employee_nik: form.employee_nik.trim(),
          name: form.name.trim(),
          email: form.email.trim(),
          department_id: form.department_id ? Number(form.department_id) : null,
          password: form.password.trim(),
          role: form.role,
        }
        await apiClient.post("/employees", payload)
        toast.success("Karyawan ditambahkan.")
      }
      onSuccess()
      onClose()
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal menyimpan data karyawan.")
    } finally {
      setLoading(false)
    }
  }

  const handleResetPassword = async () => {
    if (!employee?.id) return
    try {
      const res = await apiClient.post(`/employees/${employee.id}/reset-password`)
      setResetPassword(res.data.password)
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal reset password.")
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !loading && !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Karyawan" : "Tambah Karyawan"}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {!isEdit && (
            <div>
              <Label htmlFor="employee_nik">NIK Karyawan</Label>
              <Input
                id="employee_nik"
                value={form.employee_nik}
                onChange={handleChange("employee_nik")}
                placeholder="Kosongkan untuk generate otomatis (EMP-xxxx)"
                disabled={loading}
              />
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label htmlFor="name">Nama</Label>
              <Input
                id="name"
                value={form.name}
                onChange={handleChange("name")}
                disabled={loading}
                className={errors.name ? "border-red-500" : ""}
              />
              {errors.name && <p className="text-xs text-red-500 mt-1">{errors.name}</p>}
            </div>
            <div>
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={form.email}
                onChange={handleChange("email")}
                disabled={loading}
                className={errors.email ? "border-red-500" : ""}
              />
              {errors.email && <p className="text-xs text-red-500 mt-1">{errors.email}</p>}
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label>Departemen</Label>
              <Select
                value={form.department_id || "none"}
                onValueChange={(v) =>
                  setForm((p) => ({ ...p, department_id: v === "none" ? "" : v }))
                }
                disabled={loading || loadingDepts}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Pilih departemen" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">- Tanpa Departemen -</SelectItem>
                  {departments.map((d) => (
                    <SelectItem key={d.id} value={String(d.id)}>
                      {d.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Role</Label>
              <Select
                value={form.role}
                onValueChange={(v) => setForm((p) => ({ ...p, role: v as RoleType }))}
                disabled={loading}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Pilih role" />
                </SelectTrigger>
                <SelectContent>
                  {ROLE_OPTIONS.map((r) => (
                    <SelectItem key={r.value} value={r.value}>
                      {r.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div>
            <Label htmlFor="password">{isEdit ? "Password (opsional)" : "Password"}</Label>
            <div className="flex gap-2">
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={handleChange("password")}
                placeholder={isEdit ? "Kosongkan jika tidak diubah" : "Min. 8 karakter"}
                disabled={loading}
                className={errors.password ? "border-red-500" : ""}
              />
              {isEdit && (
                <Button
                  type="button"
                  variant="outline"
                  disabled={loading}
                  onClick={handleResetPassword}
                >
                  Reset
                </Button>
              )}
            </div>
            {errors.password && (
              <p className="text-xs text-red-500 mt-1">{errors.password}</p>
            )}
          </div>

          {isEdit && employee?.id && (
            <div className="pt-2">
              <Label>Pelatihan & Sertifikasi</Label>
              <Button
                variant="outline"
                size="sm"
                className="mt-1"
                onClick={() => window.open(`/employees/${employee.id}/trainings`, "_blank")}
              >
                <BookOpen className="mr-1 h-4 w-4" /> Lihat Riwayat Training
              </Button>
            </div>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={loading}>
              Batal
            </Button>
            <Button type="submit" disabled={disableSubmit}>
              {loading ? "Menyimpan..." : isEdit ? "Simpan Perubahan" : "Tambah Karyawan"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>

      {/* Modal Password Reset */}
      <AlertDialog
        open={!!resetPassword}
        onOpenChange={(open) => {
          if (!open) setResetPassword(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Password Baru</AlertDialogTitle>
            <AlertDialogDescription>
              Password karyawan berhasil direset. Silakan salin dan berikan ke user.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="p-3 bg-muted rounded-md font-mono text-sm text-center">
            {resetPassword}
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setResetPassword(null)}>Tutup</AlertDialogCancel>
            <Button
              onClick={() => {
                if (resetPassword) {
                  navigator.clipboard.writeText(resetPassword)
                  toast.success("Password disalin ke clipboard.")
                }
              }}
            >
              Salin Password
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Dialog>
  )
}
