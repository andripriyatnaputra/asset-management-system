// File: src/components/ChangePasswordDialog.tsx
import { useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function ChangePasswordDialog({ open, onOpenChange }: Props) {
  const [oldPw, setOldPw] = useState('')
  const [pw, setPw] = useState('')
  const [cf, setCf] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = () => {
    if (pw.length < 8) return toast.error('Minimal 8 karakter.')
    if (pw !== cf) return toast.error('Konfirmasi tidak cocok.')

    setBusy(true)
    const p = apiClient.put('/employees/me/change-password', {
      old_password: oldPw,
      new_password: pw,
    })

    // tampilkan toast berdasarkan status promise p
    toast.promise(p, {
      loading: 'Menyimpan…',
      success: () => {
        setOldPw('')
        setPw('')
        setCf('')
        onOpenChange(false)
        return 'Password diperbarui!'
      },
      error: (e) => e?.response?.data?.error || 'Gagal memperbarui.',
    })

    // kontrol state loading di promise aslinya
    p.finally(() => setBusy(false))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ganti Password</DialogTitle>
        </DialogHeader>

        <div className="grid gap-3 py-2">
          <div>
            <Label>Password Lama</Label>
            <Input type="password" value={oldPw} onChange={(e) => setOldPw(e.target.value)} disabled={busy} />
          </div>
          <div>
            <Label>Password Baru</Label>
            <Input type="password" value={pw} onChange={(e) => setPw(e.target.value)} disabled={busy} />
          </div>
          <div>
            <Label>Konfirmasi Password Baru</Label>
            <Input type="password" value={cf} onChange={(e) => setCf(e.target.value)} disabled={busy} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
            Batal
          </Button>
          <Button onClick={submit} disabled={busy}>
            {busy ? 'Menyimpan…' : 'Simpan'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
