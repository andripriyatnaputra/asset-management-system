import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

interface ContractOption { id: number; contract_number: string }

interface LicenseFormModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  license?: any | null
}

export default function LicenseFormModal({ isOpen, onClose, onSuccess, license }: LicenseFormModalProps) {
  const isEditMode = !!license
  const [contracts, setContracts] = useState<ContractOption[]>([])
  const [form, setForm] = useState({
    name: '',
    license_key: '',
    vendor: '',
    license_type: '',
    license_model: '',
    total_seats: 1,
    cost: 0,
    contract_id: '',
    purchase_date: '',
    expiration_date: '',
    compliance_status: 'unknown', // hanya untuk tampilan
  })

  const resetForm = () => {
    setForm({
      name: '',
      license_key: '',
      vendor: '',
      license_type: '',
      license_model: '',
      total_seats: 1,
      cost: 0,
      contract_id: '',
      purchase_date: '',
      expiration_date: '',
      compliance_status: 'unknown',
    })
  }

  useEffect(() => {
    apiClient.get('/contracts')
      .then(r => {
        const data = Array.isArray(r.data)
          ? r.data
          : Array.isArray(r.data?.data)
            ? r.data.data
            : []
        setContracts(data)
      })
      .catch(() => setContracts([]))
  }, [])

  useEffect(() => {
    if (isEditMode && license) {
      setForm({
        name: license.name || '',
        license_key: license.license_key || '',
        vendor: license.vendor || '',
        license_type: license.license_type || '',
        license_model: license.license_model || '',
        total_seats: license.total_seats || 1,
        cost: license.cost || 0,
        contract_id: license.contract_id ? String(license.contract_id) : '',
        purchase_date: license.purchase_date
          ? new Date(license.purchase_date).toISOString().split('T')[0]
          : '',
        expiration_date: license.expiration_date
          ? new Date(license.expiration_date).toISOString().split('T')[0]
          : '',
        // backend kirim status via json tag "status"
        compliance_status: license.status || 'unknown',
      })
    } else {
      resetForm()
    }
  }, [isEditMode, license, isOpen])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setForm(f => ({
      ...f,
      [name]:
        name === 'total_seats' || name === 'cost'
          ? Number(value)
          : value,
    }))
  }

  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (submitting) return
    if (!form.name.trim()) return toast.error("Nama Software wajib diisi.")

    // Jangan kirim compliance_status ke backend (di-maintain oleh compliance job)
    const { compliance_status, ...rest } = form

    const payload = {
      ...rest,
      contract_id: rest.contract_id ? Number(rest.contract_id) : null,
      purchase_date: rest.purchase_date
        ? new Date(rest.purchase_date).toISOString()
        : null,
      expiration_date: rest.expiration_date
        ? new Date(rest.expiration_date).toISOString()
        : null,
    }

    setSubmitting(true)
    const promise = isEditMode
      ? apiClient.patch(`/licenses/${license!.id}`, payload)
      : apiClient.post("/licenses", payload)

    toast.promise(promise, {
      loading: isEditMode ? "Memperbarui lisensi…" : "Menyimpan lisensi…",
      success: () => {
        resetForm()
        onSuccess()
        onClose()
        return isEditMode
          ? "Data lisensi berhasil diperbarui."
          : "Data lisensi berhasil disimpan."
      },
      error: (err) => {
        const msg =
          err?.response?.data?.error ||
          (err?.response?.status === 409
            ? "License key sudah terdaftar."
            : "Gagal menyimpan lisensi.")
        return msg
      },
    })

    try {
      await promise
    } catch (err) {
      console.error("License save error:", err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(o) => {
        if (!o) {
          resetForm()
          onClose()
        }
      }}
    >
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit' : 'Tambah'} Lisensi</DialogTitle>
          <DialogDescription>Lengkapi data lisensi di bawah ini.</DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 py-4">
          <div>
            <Label>Nama Software</Label>
            <Input name="name" value={form.name} onChange={handleChange} />
          </div>
          <div>
            <Label>Kunci Lisensi</Label>
            <Input name="license_key" value={form.license_key} onChange={handleChange} />
          </div>
          <div>
            <Label>Vendor</Label>
            <Input name="vendor" value={form.vendor} onChange={handleChange} />
          </div>
          <div>
            <Label>Jenis Lisensi</Label>
            <Input name="license_type" value={form.license_type} onChange={handleChange} />
          </div>
          <div>
            <Label>Model Lisensi</Label>
            <Input name="license_model" value={form.license_model} onChange={handleChange} />
          </div>
          <div>
            <Label>Jumlah Seats</Label>
            <Input
              name="total_seats"
              type="number"
              value={form.total_seats}
              onChange={handleChange}
              min={1}
            />
          </div>
          <div>
            <Label>Status Kepatuhan (read-only visual)</Label>
            <Select
              onValueChange={(v) => setForm(f => ({ ...f, compliance_status: v }))}
              value={form.compliance_status}
            >
              <SelectTrigger><SelectValue placeholder="Pilih status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="compliant">Compliant</SelectItem>
                <SelectItem value="non-compliant">Non-Compliant</SelectItem>
                <SelectItem value="unknown">Unknown</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <Label>Biaya (IDR)</Label>
            <Input
              name="cost"
              type="number"
              value={form.cost}
              onChange={handleChange}
              min={0}
            />
          </div>
          <div>
            <Label>Tanggal Pembelian</Label>
            <Input
              name="purchase_date"
              type="date"
              value={form.purchase_date}
              onChange={handleChange}
            />
          </div>
          <div>
            <Label>Tanggal Kedaluwarsa</Label>
            <Input
              name="expiration_date"
              type="date"
              value={form.expiration_date}
              onChange={handleChange}
            />
          </div>
          <div>
            <Label>Kontrak</Label>
            <Select
              onValueChange={(v) => setForm(f => ({ ...f, contract_id: v }))}
              value={form.contract_id}
            >
              <SelectTrigger><SelectValue placeholder="Pilih kontrak" /></SelectTrigger>
              <SelectContent>
                {(contracts || []).map(c => (
                  <SelectItem key={c.id} value={String(c.id)}>
                    {c.contract_number}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {submitting ? "Menyimpan…" : isEditMode ? "Perbarui" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
