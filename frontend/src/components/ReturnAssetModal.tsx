import { useState, useEffect } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog"
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

interface ReturnAssetModalProps {
  assetId: number
  isOpen: boolean
  onClose: () => void
  onReturned: () => void
}

export default function ReturnAssetModal({ assetId, isOpen, onClose, onReturned }: ReturnAssetModalProps) {
  const [nextStatus, setNextStatus] = useState<'in_stock' | 'maintenance' | 'retired'>('in_stock')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Reset state tiap kali modal ditutup
  useEffect(() => {
    if (!isOpen) {
      setNextStatus('in_stock')
      setNotes('')
      setSubmitting(false)
    }
  }, [isOpen])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (submitting) return
    setSubmitting(true)

    const promise = apiClient.post(`/assets/${assetId}/return`, {
      next_status: nextStatus,
      notes: notes?.trim() || null,
    })

    toast.promise(promise, {
      loading: 'Mengembalikan aset...',
      success: 'Aset berhasil dikembalikan!',
      error: (err) =>
        err?.response?.data?.error ||
        'Gagal mengembalikan aset. Pastikan aset belum dikunci atau sudah di-assign sebelumnya.',
    })

    try {
      const res = await promise
      if (res.status === 200) {
        onReturned()
        onClose()
      }
    } catch (err) {
      console.error('Return asset error:', err)
    } finally {
      setSubmitting(false)
    }
  }



  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent>
        <DialogHeader><DialogTitle>Return Asset</DialogTitle></DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          <div>
            <Label>Status Aset Selanjutnya</Label>
            <Select value={nextStatus} onValueChange={(v) => setNextStatus(v as any)}>
              <SelectTrigger><SelectValue placeholder="Pilih status" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="in_stock">In Stock</SelectItem>
                <SelectItem value="maintenance">Maintenance</SelectItem>
                <SelectItem value="retired">Retired</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Catatan Pengembalian</Label>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Contoh: Dikembalikan oleh Budi, kondisi baik."
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>Batal</Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? 'Mengembalikan…' : 'Kembalikan'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
