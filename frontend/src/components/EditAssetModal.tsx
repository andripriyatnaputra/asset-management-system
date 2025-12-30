import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import apiClient from '@/services/api'

import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Textarea } from './ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from './ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from './ui/select'

const NONE_VALUE = '__none__'
type DepMethod = 'straight_line' | 'declining_balance'

type AssetType = { id: number; name: string }
type Department = { id: number; name: string }
type Location = { id: number; site?: string; building?: string; room?: string }

const toISOorNull = (v: string | '') => (v ? new Date(v).toISOString() : null)
const toNumOrNull = (v: number | '' | null | undefined) =>
  typeof v === 'number' && !Number.isNaN(v) ? v : null

const locText = (l?: Partial<Location>) =>
  [l?.site, l?.building, l?.room].filter(Boolean).join(' • ')

type Props = {
  assetId: number | null
  isOpen?: boolean
  open?: boolean
  onClose?: () => void
  onOpenChange?: (v: boolean) => void
  onSuccess?: () => void
}

export default function EditAssetModal(props: Props) {
  const open = props.open ?? props.isOpen ?? false
  const onOpenChange = (v: boolean) => {
    props.onOpenChange?.(v)
    if (!v) props.onClose?.()
  }

  const [assetTypes, setAssetTypes] = useState<AssetType[]>([])
  const [departments, setDepartments] = useState<Department[]>([])
  const [locations, setLocations] = useState<Location[]>([])
  const [isLoading, setIsLoading] = useState(false)

  const [form, setForm] = useState({
    name: '',
    asset_tag: '',
    asset_type_id: '',
    status: 'in_stock',
    department_id: NONE_VALUE,
    location_id: NONE_VALUE,
    purchase_date: '',
    initial_price: '' as number | '',
    serial_number: '',
    vendor: '',
    warranty_expiry: '',
    useful_life_months: '' as number | '',
    depreciation_method: 'straight_line' as DepMethod,
    salvage_value: '' as number | '',
    notes: '',
  })

  // load detail & masters
  useEffect(() => {
    if (!open || !props.assetId) return
    let cancelled = false
    setIsLoading(true)

    const pickData = (res: any) =>
      Array.isArray(res?.data) ? res.data : res?.data?.data ?? []

    ;(async () => {
      try {
        const [detailRes, typesRes, deptRes, locRes] = await Promise.all([
          apiClient.get(`/assets/${props.assetId}`),
          apiClient.get('/asset-types'),
          apiClient.get('/departments'),
          apiClient.get('/locations').catch((err) => {
            if (err?.response?.status !== 404) throw err
            return { data: [] }
          }),
        ])
        if (cancelled) return

        const a = detailRes.data || {}

        setAssetTypes(pickData(typesRes))
        setDepartments(pickData(deptRes))
        setLocations(pickData(locRes))

        const toDateInput = (iso?: string | null) =>
          iso ? new Date(iso).toISOString().slice(0, 10) : ''

        setForm((s) => ({
          ...s,
          name: a.name ?? '',
          asset_tag: a.asset_tag ?? '',
          asset_type_id: a.asset_type_id ? String(a.asset_type_id) : '',
          status: (a.status?.toLowerCase().replace(/\s+/g, '_') as string) || 'in_stock',
          department_id:
            a.owner_department_id ? String(a.owner_department_id) : NONE_VALUE,
          location_id:
            a.current_location_id ? String(a.current_location_id) : NONE_VALUE,
          purchase_date: toDateInput(a.purchase_date),
          initial_price:
            typeof a.initial_price === 'number' ? a.initial_price : '',
          serial_number: a.serial_number ?? '',
          vendor: a.vendor ?? '',
          warranty_expiry: toDateInput(a.warranty_expiry),
          useful_life_months:
            typeof a.useful_life_months === 'number' ? a.useful_life_months : '',
          depreciation_method:
            (a.depreciation_method as DepMethod) || 'straight_line',
          salvage_value:
            typeof a.salvage_value === 'number' ? a.salvage_value : '',
          notes: '',
        }))
      } catch {
        if (!cancelled) toast.error('Gagal memuat detail aset.')
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [open, props.assetId])

  const disableSave = useMemo(() => {
    return !form.name || !form.asset_tag || !form.asset_type_id
  }, [form.name, form.asset_tag, form.asset_type_id])

  const handleSubmit = async () => {
    if (!props.assetId) return
    try {
      setIsLoading(true)
      await apiClient.put(`/assets/${props.assetId}`, {
        name: form.name,
        asset_type_id: Number(form.asset_type_id),
        status: form.status,
        department_id: form.department_id !== NONE_VALUE ? Number(form.department_id) : null,
        location_id: form.location_id !== NONE_VALUE ? Number(form.location_id) : null,
        purchase_date: toISOorNull(form.purchase_date),
        initial_price: toNumOrNull(form.initial_price),
        serial_number: form.serial_number || null,
        vendor: form.vendor || null,
        warranty_expiry: toISOorNull(form.warranty_expiry),
        notes: form.notes || null,

        useful_life_months: toNumOrNull(form.useful_life_months),
        depreciation_method: form.depreciation_method,
        salvage_value: toNumOrNull(form.salvage_value),
      })
      toast.success('Aset berhasil diperbarui.')
      props.onSuccess?.()
      onOpenChange(false)
    } catch {
      toast.error('Gagal memperbarui aset.')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Edit Aset</DialogTitle>
          <DialogDescription>Perbarui informasi aset di bawah ini.</DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <Label>Nama Aset</Label>
            <Input
              value={form.name}
              onChange={(e) => setForm((s) => ({ ...s, name: e.target.value }))}
              disabled={isLoading}
            />
          </div>
          <div>
            <Label>Asset Tag</Label>
            <Input value={form.asset_tag} disabled /> {/* biasanya tag tidak diubah */}
          </div>

          <div>
            <Label>Tipe Aset</Label>
            <Select
              value={form.asset_type_id}
              onValueChange={(v) => setForm((s) => ({ ...s, asset_type_id: v }))}
              disabled={isLoading}
            >
              <SelectTrigger><SelectValue placeholder="Pilih tipe aset" /></SelectTrigger>
              <SelectContent>
                {assetTypes.map((t) => (
                  <SelectItem key={t.id} value={String(t.id)}>{t.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Status</Label>
            <Select
              value={form.status}
              onValueChange={(v) => setForm((s) => ({ ...s, status: v }))}
              disabled={isLoading}
            >
              <SelectTrigger><SelectValue placeholder="Status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="in_stock">In Stock</SelectItem>
                <SelectItem value="assigned">Assigned</SelectItem>
                <SelectItem value="maintenance">Maintenance</SelectItem>
                <SelectItem value="retired">Retired</SelectItem>
                <SelectItem value="disposed">Disposed</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Owner Department</Label>
            <Select
              value={form.department_id}
              onValueChange={(v) => setForm((s) => ({ ...s, department_id: v }))}
              disabled={isLoading}
            >
              <SelectTrigger><SelectValue placeholder="Pilih department" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE_VALUE}>— Tidak ada —</SelectItem>
                {departments.map((d) => (
                  <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Current Location</Label>
            <Select
              value={form.location_id}
              onValueChange={(v) => setForm((s) => ({ ...s, location_id: v }))}
              disabled={isLoading}
            >
              <SelectTrigger><SelectValue placeholder="Pilih lokasi" /></SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE_VALUE}>— Tidak ada —</SelectItem>
                {locations.map((l) => (
                  <SelectItem key={l.id} value={String(l.id)}>{locText(l)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Tanggal Pembelian</Label>
            <Input
              type="date"
              value={form.purchase_date}
              onChange={(e) => setForm((s) => ({ ...s, purchase_date: e.target.value }))}
              disabled={isLoading}
            />
          </div>
          <div>
            <Label>Harga Perolehan</Label>
            <Input
              type="number"
              min={0}
              value={form.initial_price}
              onChange={(e) =>
                setForm((s) => ({ ...s, initial_price: e.target.value ? Number(e.target.value) : '' }))
              }
              disabled={isLoading}
            />
          </div>

          <div>
            <Label>Serial Number</Label>
            <Input
              placeholder="SN/IMEI/Service Tag"
              value={form.serial_number}
              onChange={(e) => setForm((s) => ({ ...s, serial_number: e.target.value }))}
              disabled={isLoading}
            />
          </div>
          <div>
            <Label>Vendor</Label>
            <Input
              value={form.vendor}
              onChange={(e) => setForm((s) => ({ ...s, vendor: e.target.value }))}
              disabled={isLoading}
            />
          </div>

          <div>
            <Label>Masa Garansi S/D</Label>
            <Input
              type="date"
              value={form.warranty_expiry}
              onChange={(e) => setForm((s) => ({ ...s, warranty_expiry: e.target.value }))}
              disabled={isLoading}
            />
          </div>
          <div className="md:col-span-2">
            <Label>Catatan Perubahan (opsional)</Label>
            <Textarea
              placeholder="Contoh: update status, pindah lokasi, koreksi data."
              value={form.notes}
              onChange={(e) => setForm((s) => ({ ...s, notes: e.target.value }))}
              disabled={isLoading}
            />
          </div>
        </div>

        {/* Advanced (Depreciation) */}
        <div className="mt-6 rounded-xl border p-4">
          <p className="text-sm font-medium mb-3">Advanced (Depresiasi) — opsional</p>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <Label>Umur Manfaat (bulan)</Label>
              <Input
                type="number"
                min={1}
                placeholder="contoh: 36"
                value={form.useful_life_months}
                onChange={(e) =>
                  setForm((s) => ({ ...s, useful_life_months: e.target.value ? Number(e.target.value) : '' }))
                }
                disabled={isLoading}
              />
            </div>
            <div>
              <Label>Metode Penyusutan</Label>
              <Select
                value={form.depreciation_method}
                onValueChange={(v) => setForm((s) => ({ ...s, depreciation_method: v as DepMethod }))}
                disabled={isLoading}
              >
                <SelectTrigger><SelectValue placeholder="Metode" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="straight_line">Garis Lurus</SelectItem>
                  <SelectItem value="declining_balance">Saldo Menurun</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Nilai Sisa (jika ada)</Label>
              <Input
                type="number"
                min={0}
                placeholder="contoh: 1000000"
                value={form.salvage_value}
                onChange={(e) =>
                  setForm((s) => ({ ...s, salvage_value: e.target.value ? Number(e.target.value) : '' }))
                }
                disabled={isLoading}
              />
            </div>
          </div>
        </div>

        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={disableSave || isLoading}>
            Simpan Perubahan
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
