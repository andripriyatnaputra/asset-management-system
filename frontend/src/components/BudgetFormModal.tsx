import { useState, useEffect } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { CostCenter, Department } from "@/types"

// ===============================
// SAFE ARRAY HELPER
// ===============================
function toArray<T>(v: any): T[] {
  if (!v) return []
  if (Array.isArray(v)) return v
  if (Array.isArray(v?.data)) return v.data
  return []
}

interface BudgetFormModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  budget: any | null
}

export default function BudgetFormModal({ isOpen, onClose, onSuccess, budget }: BudgetFormModalProps) {
  const isEdit = !!budget

  const [departments, setDepartments] = useState<any[]>([])
  const [costCenters, setCostCenters] = useState<any[]>([])

  const [form, setForm] = useState({
    name: "",
    department_id: "",
    start_date: "",
    end_date: "",
    total_amount: "",
    category: "CAPEX",
    cost_center_id: "",
  })

  const [saving, setSaving] = useState(false)

  useEffect(() => {
    Promise.all([
      apiClient.get("/departments").catch(() => ({ data: null })),
      apiClient.get("/cost-centers").catch(() => ({ data: null })),
    ]).then(([d, cc]) => {
      setDepartments(toArray(d.data))
      setCostCenters(toArray(cc.data))
    })

    if (isEdit && budget) {
      setForm({
        name: budget.name ?? "",
        department_id: budget.department_id ? String(budget.department_id) : "",
        start_date: budget.start_date?.split("T")[0] ?? "",
        end_date: budget.end_date?.split("T")[0] ?? "",
        total_amount: String(budget.total_amount ?? 0),
        category: budget.category ?? "CAPEX",
        cost_center_id: budget.cost_center_id ? String(budget.cost_center_id) : "",
      })
    } else {
      setForm({
        name: "",
        department_id: "",
        start_date: "",
        end_date: "",
        total_amount: "",
        category: "CAPEX",
        cost_center_id: "",
      })
    }
  }, [isOpen])

  const handleSubmit = async () => {
    if (saving) return
    if (!form.name || !form.start_date || !form.end_date) {
      toast.error("Nama, tanggal mulai, dan tanggal selesai wajib diisi.")
      return
    }

    const payload = {
      name: form.name.trim(),
      department_id: form.department_id ? Number(form.department_id) : null,
      start_date: new Date(form.start_date).toISOString(),
      end_date: new Date(form.end_date).toISOString(),
      total_amount: Number(form.total_amount) || 0,
      category: form.category,
      cost_center_id: form.cost_center_id ? Number(form.cost_center_id) : null,
    }

    setSaving(true)
    const promise = isEdit
      ? apiClient.put(`/budgets/${budget.id}`, payload)
      : apiClient.post("/budgets", payload)

    toast.promise(promise, {
      loading: "Menyimpan anggaran...",
      success: "Anggaran berhasil disimpan.",
      error: (err) => err?.response?.data?.error || "Gagal menyimpan anggaran.",
    })

    try {
      await promise
      onSuccess()
      onClose()
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Anggaran" : "Tambah Anggaran"}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4 py-4">

          <div>
            <Label>Nama Anggaran</Label>
            <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} />
          </div>

          <div>
            <Label>Departemen</Label>
            <Select value={form.department_id} onValueChange={v => setForm({ ...form, department_id: v })}>
              <SelectTrigger><SelectValue placeholder="Pilih Departemen" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__general__">(Umum)</SelectItem>
                {toArray<Department>(departments).map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div><Label>Tanggal Mulai</Label><Input type="date" value={form.start_date} onChange={e => setForm({ ...form, start_date: e.target.value })} /></div>
            <div><Label>Tanggal Selesai</Label><Input type="date" value={form.end_date} onChange={e => setForm({ ...form, end_date: e.target.value })} /></div>
          </div>

          <div>
            <Label>Jumlah Total (IDR)</Label>
            <Input type="number" value={form.total_amount} onChange={e => setForm({ ...form, total_amount: e.target.value })} />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <Label>Kategori</Label>
              <Select value={form.category} onValueChange={v => setForm({ ...form, category: v })}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="CAPEX">CAPEX (Investasi)</SelectItem>
                  <SelectItem value="OPEX">OPEX (Operasional)</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label>Cost Center</Label>
              <Select value={form.cost_center_id} onValueChange={v => setForm({ ...form, cost_center_id: v })}>
                <SelectTrigger><SelectValue placeholder="Pilih Cost Center" /></SelectTrigger>
                <SelectContent>
                {toArray<CostCenter>(costCenters).length === 0 ? (
                  <SelectItem value="__no_data__" disabled>
                    Belum ada data Cost Center
                  </SelectItem>
                ) : (
                  toArray<CostCenter>(costCenters).map((cc) => (
                    <SelectItem key={cc.id} value={String(cc.id)}>
                      {cc.code} — {cc.name}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
              </Select>
            </div>
          </div>

        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit} disabled={saving}>
            {saving ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
