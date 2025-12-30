import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Contract {
  id?: number
  contract_number: string
  vendor?: string
  contract_type?: string
  start_date?: string
  end_date?: string
  total_value?: number
  currency?: string
  payment_terms?: string
  contact_person?: string
  contact_email?: string
  status?: string
  notes?: string
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  contract: Contract | null
}

const formatDateForInput = (d?: string) => (d ? new Date(d).toISOString().split('T')[0] : '')

export default function ContractFormModal({ isOpen, onClose, onSuccess, contract }: Props) {
  const isEditMode = !!contract
  const [form, setForm] = useState<Contract>({
    contract_number: '',
    vendor: '',
    contract_type: '',
    start_date: '',
    end_date: '',
    total_value: 0,
    currency: 'IDR',
    payment_terms: '',
    contact_person: '',
    contact_email: '',
    status: 'active',
    notes: '',
  })

  useEffect(() => {
    if (isEditMode && contract) {
      setForm({
        ...contract,
        start_date: formatDateForInput(contract.start_date),
        end_date: formatDateForInput(contract.end_date),
      })
    } else {
      setForm({
        contract_number: '',
        vendor: '',
        contract_type: '',
        start_date: '',
        end_date: '',
        total_value: 0,
        currency: 'IDR',
        payment_terms: '',
        contact_person: '',
        contact_email: '',
        status: 'active',
        notes: '',
      })
    }
  }, [contract, isEditMode, isOpen])

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setForm((f) => ({
      ...f,
      [name]: name === 'total_value' ? Number(value) : value,
    }))
  }

  const handleSubmit = async () => {
    if (!form.contract_number.trim()) return toast.error('Nomor kontrak wajib diisi.')
    if (!form.start_date) return toast.error('Tanggal mulai wajib diisi.')

    const payload = {
      ...form,
      start_date: form.start_date ? new Date(form.start_date).toISOString() : null,
      end_date: form.end_date ? new Date(form.end_date).toISOString() : null,
    }

    const promise = isEditMode
      ? apiClient.put(`/contracts/${contract!.id}`, payload)
      : apiClient.post('/contracts', payload)

    toast.promise(promise, {
      loading: isEditMode ? 'Menyimpan perubahan…' : 'Menambahkan kontrak…',
      success: () => { onSuccess(); return isEditMode ? 'Kontrak diperbarui!' : 'Kontrak ditambahkan!' },
      error: (err) => err?.response?.data?.error || 'Gagal menyimpan kontrak.',
    })
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Kontrak' : 'Tambah Kontrak'}</DialogTitle>
          <DialogDescription>Lengkapi detail kontrak di bawah ini.</DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 py-4">
          <div><Label>Nomor Kontrak</Label><Input name="contract_number" value={form.contract_number} onChange={handleChange} /></div>
          <div><Label>Vendor</Label><Input name="vendor" value={form.vendor} onChange={handleChange} /></div>
          <div><Label>Tipe</Label><Input name="contract_type" value={form.contract_type} onChange={handleChange} /></div>
          <div><Label>Tanggal Mulai</Label><Input name="start_date" type="date" value={form.start_date} onChange={handleChange} /></div>
          <div><Label>Tanggal Berakhir</Label><Input name="end_date" type="date" value={form.end_date} onChange={handleChange} /></div>
          <div><Label>Nilai Total (IDR)</Label><Input name="total_value" type="number" value={form.total_value} onChange={handleChange} min={0} /></div>
          <div><Label>Kontak Person</Label><Input name="contact_person" value={form.contact_person} onChange={handleChange} /></div>
          <div><Label>Email</Label><Input name="contact_email" type="email" value={form.contact_email} onChange={handleChange} /></div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
