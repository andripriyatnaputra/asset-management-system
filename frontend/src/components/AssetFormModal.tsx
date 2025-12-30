import { useEffect, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { toast } from "sonner"
import apiClient from "@/services/api"

type Option = { id: number; name: string }
type Department = { id: number; name: string }
type Location = { id: number; site: string; building?: string | null; room?: string | null }
type CostCenter = { id: number; code?: string | null; name: string }
type Budget = { id: number; name: string; cost_center_id?: number | null; department_id?: number | null }
type Contract = { id: number; name?: string | null }

interface AssetFormModalProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  assetId?: string | null
  onSaved?: () => void
  assetTypes: Option[]
  departments: Department[]
  locations: Location[]
}

interface AssetForm {
  // identification
  name: string
  asset_tag: string
  asset_type_id: string | null
  department_id: string | null
  location_id: string | null
  status: string

  // finance & procurement
  purchase_date: string | null
  purchase_cost: string | null
  initial_price: string | null
  depreciation_method: string
  salvage_value: string | null
  useful_life_months: string | null
  serial_number: string
  vendor: string
  warranty_expiry: string | null

  // governance & misc
  notes: string
  ownership_type: string
  acquisition_type: string
  asset_condition: string
  budget_id: string | null
  cost_center_id: string | null
  contract_id: string | null
  lifecycle_stage: string
  asset_criticality: string | null
}

const toISO = (d?: string | null) => (d ? new Date(d).toISOString() : null)

export default function AssetFormModal({
  open,
  onOpenChange,
  assetId,
  onSaved,
  assetTypes,
  departments,
  locations,
}: AssetFormModalProps) {
  const [saving, setSaving] = useState(false)

  const [budgets, setBudgets] = useState<Budget[]>([])
  const [contracts, setContracts] = useState<Contract[]>([])
  const [costCenters, setCostCenters] = useState<CostCenter[]>([])
  const [autoCostCenter, setAutoCostCenter] = useState(false)

  const [form, setForm] = useState<AssetForm>({
    name: "",
    asset_tag: "",
    asset_type_id: null,
    department_id: null,
    location_id: null,
    status: "in_stock",
    purchase_date: null,
    purchase_cost: null,
    initial_price: null,
    depreciation_method: "straight_line",
    salvage_value: null,
    useful_life_months: "36",
    serial_number: "",
    vendor: "",
    warranty_expiry: null,
    notes: "",
    ownership_type: "company_owned",
    acquisition_type: "purchase",
    asset_condition: "good",
    budget_id: null,
    cost_center_id: null,
    contract_id: null,
    lifecycle_stage: "in_use",
    asset_criticality: null,
  })

  const formatCurrency = (value: string | null) => {
    if (!value) return "";
    const cleaned = value.replace(/\D/g, ""); // hanya angka
    return cleaned.replace(/\B(?=(\d{3})+(?!\d))/g, "."); // format ribuan
  };

  const parseCurrency = (value: string | null) => {
    if (!value) return null;
    return Number(value.replace(/\./g, "")); // hapus titik sebelum dikirim ke backend
  };

  const safeArray = (v: any): any[] => {
    if (Array.isArray(v)) return v
    if (v && typeof v === "object") {
      // coba ambil array pertama yang valid
      for (const k of Object.keys(v)) {
        if (Array.isArray(v[k])) return v[k]
      }
    }
    return []
  }

  // ---------- Load reference data ----------
  useEffect(() => {
    if (!open) return
    apiClient.get("/budgets")
      .then(res => setBudgets(safeArray(res.data)))
      .catch(() => setBudgets([]))
    apiClient.get("/cost-centers")
      .then(res => setCostCenters(safeArray(res.data)))
      .catch(() => setCostCenters([]))
    Promise.all([apiClient.get("/contracts"), apiClient.get("/licenses")])
      .then(([c]) => {
        setContracts(safeArray(c.data))
      })
      .catch(() => {
        setContracts([])
      })
  }, [open])

  const loadAsset = async (id: string) => {
    try {
      const res = await apiClient.get(`/assets/${id}`)
      const a = res.data?.asset ?? {}
      if (a.status === "disposed") {
        toast.error("Aset sudah disposed, tidak dapat diedit.")
        onOpenChange(false)
        return
      }
      setForm({
        ...form,
        name: a.name ?? "",
        asset_tag: a.asset_tag ?? "",
        asset_type_id: a.asset_type_id ? String(a.asset_type_id) : null,
        department_id: a.department_id ? String(a.department_id) : null,
        location_id: a.location_id ? String(a.location_id) : null,
        status: a.status ?? "in_stock",
        purchase_date: a.purchase_date ? a.purchase_date.split("T")[0] : null,
        purchase_cost: a.purchase_cost ? String(a.purchase_cost) : null,
        initial_price: a.initial_price ? String(a.initial_price) : null,
        depreciation_method: a.depreciation_method ?? "straight_line",
        salvage_value: a.salvage_value ? String(a.salvage_value) : null,
        useful_life_months: a.useful_life_months ? String(a.useful_life_months) : "36",
        serial_number: a.serial_number ?? "",
        vendor: a.vendor ?? "",
        warranty_expiry: a.warranty_expiry ? a.warranty_expiry.split("T")[0] : null,
        notes: a.notes ?? "",
        ownership_type: a.ownership_type ?? "company_owned",
        acquisition_type: a.acquisition_type ?? "purchase",
        asset_condition: a.asset_condition ?? "good",
        budget_id: a.budget_id ? String(a.budget_id) : null,
        cost_center_id: a.cost_center_id ? String(a.cost_center_id) : null,
        contract_id: a.contract_id ? String(a.contract_id) : null,
        lifecycle_stage: a.lifecycle_stage ?? "in_use",
        asset_criticality: a.asset_criticality ?? null,
      })
      setAutoCostCenter(Boolean(a.budget_id && a.cost_center_id))
    } catch {
      toast.error("Gagal memuat data aset.")
    }
  }

  useEffect(() => {
    if (open && assetId) loadAsset(assetId)
  }, [open, assetId])

  // ---------- Budget → auto Cost Center ----------
  const handleBudgetChange = async (v: string) => {
    setForm((prev) => ({ ...prev, budget_id: v || null }))
    if (!v) {
      setAutoCostCenter(false)
      setForm((prev) => ({ ...prev, cost_center_id: null }))
      return
    }
    try {
      const res = await apiClient.get(`/budgets/${v}`)
      const b = res.data?.budget ?? {}
      if (b?.cost_center_id) {
        setForm((prev) => ({ ...prev, cost_center_id: String(b.cost_center_id) }))
        setAutoCostCenter(true)
        toast.info("Cost Center diisi otomatis dari Budget terpilih")
      } else setAutoCostCenter(false)
    } catch {
      setAutoCostCenter(false)
    }
  }

  // ---------- Submit ----------
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (saving) return

    if (!form.initial_price || Number(form.initial_price) <= 0) {
      toast.error("Nilai perolehan (Initial Price) harus diisi dan > 0.")
      return
    }
    if (form.purchase_cost && Number(form.purchase_cost) < 0) {
      toast.error("Harga pembelian tidak boleh negatif.")
      return
    }

    setSaving(true)
    const payload = {
      name: form.name,
      asset_tag: form.asset_tag,
      asset_type_id: form.asset_type_id ? Number(form.asset_type_id) : null,
      department_id: form.department_id ? Number(form.department_id) : null,
      location_id: form.location_id ? Number(form.location_id) : null,
      status: form.status,
      purchase_date: toISO(form.purchase_date),
      purchase_cost: parseCurrency(form.purchase_cost),
      initial_price: parseCurrency(form.initial_price),
      salvage_value: parseCurrency(form.salvage_value),
      depreciation_method: form.depreciation_method,
      useful_life_months: form.useful_life_months ? Number(form.useful_life_months) : 36,
      serial_number: form.serial_number || null,
      vendor: form.vendor || null,
      warranty_expiry: toISO(form.warranty_expiry),
      notes: form.notes || null,
      ownership_type: form.ownership_type,
      acquisition_type: form.acquisition_type,
      asset_condition: form.asset_condition,
      budget_id: form.budget_id ? Number(form.budget_id) : null,
      cost_center_id: form.cost_center_id ? Number(form.cost_center_id) : null,
      contract_id: form.contract_id ? Number(form.contract_id) : null,
      lifecycle_stage: form.lifecycle_stage,
      asset_criticality: form.asset_criticality,
      currency: "IDR",
    }

    try {
      const promise = assetId
        ? apiClient.put(`/assets/${assetId}`, payload)
        : apiClient.post("/assets", payload)

      await toast.promise(promise, {
        loading: "Menyimpan aset...",
        success: () => {
          onSaved?.()
          onOpenChange(false)
          return "Aset berhasil disimpan."
        },
        error: (err: any) =>
          err?.response?.data?.error || "Gagal menyimpan data aset.",
      })
    } finally {
      setSaving(false)
    }
  }

  // ---------- Render ----------
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl">
        <DialogHeader>
          <DialogTitle>{assetId ? "Edit Aset" : "Tambah Aset Baru"}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 py-4 px-2">
          {/* --- Data Dasar --- */}
          <section>
            <h3 className="text-sm font-semibold text-muted-foreground mb-2">Data Dasar</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <div><Label>Nama Aset</Label><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div>
              <div><Label>Asset Tag</Label><Input value={form.asset_tag} onChange={(e) => setForm({ ...form, asset_tag: e.target.value })} required /></div>
              <div>
                <Label>Tipe Aset</Label>
                <Select value={form.asset_type_id ?? ""} onValueChange={(v) => setForm({ ...form, asset_type_id: v || null })}>
                  <SelectTrigger><SelectValue placeholder="Pilih tipe aset" /></SelectTrigger>
                  <SelectContent>{assetTypes.map((t) => <SelectItem key={t.id} value={String(t.id)}>{t.name}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              <div>
                <Label>Departemen</Label>
                <Select value={form.department_id ?? ""} onValueChange={(v) => setForm({ ...form, department_id: v || null })}>
                  <SelectTrigger><SelectValue placeholder="Pilih departemen" /></SelectTrigger>
                  <SelectContent>{departments.map((d) => <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              <div>
                <Label>Lokasi</Label>
                <Select value={form.location_id ?? ""} onValueChange={(v) => setForm({ ...form, location_id: v || null })}>
                  <SelectTrigger><SelectValue placeholder="Pilih lokasi" /></SelectTrigger>
                  <SelectContent>
                    {locations.map((l) => (
                      <SelectItem key={l.id} value={String(l.id)}>
                        {l.site}{l.building ? ` - ${l.building}` : ""}{l.room ? ` - ${l.room}` : ""}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </section>

          {/* --- Keuangan --- */}
          <section>
            <h3 className="text-sm font-semibold text-muted-foreground mb-2">Informasi Keuangan</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <div><Label>Tanggal Pembelian</Label><Input type="date" value={form.purchase_date ?? ""} onChange={(e) => setForm({ ...form, purchase_date: e.target.value || null })} /></div>
              <div>
                <Label>Harga Pembelian (Rp)</Label>
                <Input
                  value={form.purchase_cost ?? ""}
                  onChange={(e) =>
                    setForm({ ...form, purchase_cost: formatCurrency(e.target.value) })
                  }
                />
              </div>
              <div>
              <Label>Nilai Perolehan (Rp)</Label>
              <Input
                value={form.initial_price ?? ""}
                onChange={(e) =>
                  setForm({ ...form, initial_price: formatCurrency(e.target.value) })
                }
              />
            </div>
            <div>
              <Label>Nilai Residu (Rp)</Label>
              <Input
                value={form.salvage_value ?? ""}
                onChange={(e) =>
                  setForm({ ...form, salvage_value: formatCurrency(e.target.value) })
                }
              />
            </div>
              <div>
                <Label>Metode Depresiasi</Label>
                <Select value={form.depreciation_method} onValueChange={(v) => setForm({ ...form, depreciation_method: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="straight_line">Straight Line</SelectItem>
                    <SelectItem value="double_declining">Double Declining</SelectItem>
                    <SelectItem value="sum_of_years">Sum of Years</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <Label>Umur Manfaat (bulan)</Label>
                <Input type="number" value={form.useful_life_months ?? ""} onChange={(e) => setForm({ ...form, useful_life_months: e.target.value || null })} />
              </div>

              <div><Label>Vendor</Label><Input value={form.vendor} onChange={(e) => setForm({ ...form, vendor: e.target.value })} /></div>
              <div><Label>Garansi Berakhir</Label><Input type="date" value={form.warranty_expiry ?? ""} onChange={(e) => setForm({ ...form, warranty_expiry: e.target.value || null })} /></div>
            </div>
          </section>

          {/* --- Governance & Compliance --- */}
          <section>
            <h3 className="text-sm font-semibold text-muted-foreground mb-2">Governance & Compliance</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <div><Label>Budget</Label><Select value={form.budget_id ?? ""} onValueChange={handleBudgetChange}><SelectTrigger><SelectValue placeholder="Pilih budget" /></SelectTrigger><SelectContent>{budgets.map((b) => <SelectItem key={b.id} value={String(b.id)}>{b.name}</SelectItem>)}</SelectContent></Select></div>
              <div>
                <Label>Cost Center</Label>
                <Select value={form.cost_center_id ?? ""} onValueChange={(v) => setForm({ ...form, cost_center_id: v || null })} disabled={autoCostCenter}>
                  <SelectTrigger><SelectValue placeholder={autoCostCenter ? "Otomatis dari Budget" : "Pilih cost center"} /></SelectTrigger>
                  <SelectContent>{costCenters.map((cc) => <SelectItem key={cc.id} value={String(cc.id)}>{cc.code ? `${cc.code} — ${cc.name}` : cc.name}</SelectItem>)}</SelectContent>
                </Select>
                {autoCostCenter && <p className="text-xs text-muted-foreground mt-1">Cost Center diisi otomatis dari Budget.</p>}
              </div>
              <div><Label>Contract</Label><Select value={form.contract_id ?? ""} onValueChange={(v) => setForm({ ...form, contract_id: v || null })}><SelectTrigger><SelectValue placeholder="Pilih kontrak" /></SelectTrigger><SelectContent>{contracts.map((c) => <SelectItem key={c.id} value={String(c.id)}>{c.name || `Contract #${c.id}`}</SelectItem>)}</SelectContent></Select></div>
              <div><Label>Lifecycle Stage</Label><Select value={form.lifecycle_stage} onValueChange={(v) => setForm({ ...form, lifecycle_stage: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="in_use">In Use</SelectItem><SelectItem value="maintenance">Maintenance</SelectItem><SelectItem value="retired">Retired</SelectItem><SelectItem value="disposed">Disposed</SelectItem></SelectContent></Select></div>
              <div><Label>Asset Criticality</Label><Select value={form.asset_criticality ?? ""} onValueChange={(v) => setForm({ ...form, asset_criticality: v || null })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="low">Low</SelectItem><SelectItem value="medium">Medium</SelectItem><SelectItem value="high">High</SelectItem><SelectItem value="critical">Critical</SelectItem></SelectContent></Select></div>
            </div>
          </section>

          {/* --- Teknis --- */}
          <section>
            <h3 className="text-sm font-semibold text-muted-foreground mb-2">Teknis</h3>
            <div className="grid md:grid-cols-3 gap-4">
              <div><Label>Serial Number</Label><Input value={form.serial_number} onChange={(e) => setForm({ ...form, serial_number: e.target.value })} /></div>
              <div><Label>Asset Condition</Label><Select value={form.asset_condition} onValueChange={(v) => setForm({ ...form, asset_condition: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="excellent">Excellent</SelectItem><SelectItem value="good">Good</SelectItem><SelectItem value="fair">Fair</SelectItem><SelectItem value="poor">Poor</SelectItem></SelectContent></Select></div>
              <div><Label>Ownership Type</Label><Select value={form.ownership_type} onValueChange={(v) => setForm({ ...form, ownership_type: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="company_owned">Company Owned</SelectItem><SelectItem value="leased">Leased</SelectItem><SelectItem value="loaned">Loaned</SelectItem></SelectContent></Select></div>
              <div><Label>Acquisition Type</Label><Select value={form.acquisition_type} onValueChange={(v) => setForm({ ...form, acquisition_type: v })}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="purchase">Purchase</SelectItem><SelectItem value="transfer">Transfer</SelectItem><SelectItem value="donation">Donation</SelectItem></SelectContent></Select></div>
            </div>
          </section>

          <div><Label>Catatan</Label><Textarea value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Batal</Button>
            <Button type="submit" disabled={saving}>{saving ? "Menyimpan..." : "Simpan"}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
