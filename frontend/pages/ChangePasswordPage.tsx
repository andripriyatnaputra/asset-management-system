import { useMemo, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { Eye, EyeOff, ShieldCheck } from 'lucide-react'

function scorePassword(pw: string) {
  let score = 0
  if (pw.length >= 8) score += 25
  if (/[A-Z]/.test(pw)) score += 15
  if (/[a-z]/.test(pw)) score += 15
  if (/\d/.test(pw)) score += 15
  if (/[^A-Za-z0-9]/.test(pw)) score += 15
  if (pw.length >= 12) score += 15
  return Math.min(score, 100)
}

export default function ChangePasswordPage() {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [show, setShow] = useState({ old: false, new: false, confirm: false })

  const complexity = useMemo(() => scorePassword(newPassword), [newPassword])

  const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault()
  if (newPassword.length < 8) {
    toast.error('Password baru harus minimal 8 karakter.')
    return
  }
  if (newPassword !== confirmPassword) {
    toast.error('Password baru dan konfirmasi tidak cocok.')
    return
  }

  setSubmitting(true)
  const p = apiClient.put('/employees/me/change-password', {
    old_password: oldPassword,
    new_password: newPassword,
  })

  // tampilkan toast yang mengikuti status promise p
  toast.promise(p, {
    loading: 'Memperbarui password...',
    success: (res) => {
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
      return res?.data?.message || 'Password berhasil diperbarui!'
    },
    error: (err) => err?.response?.data?.error || 'Gagal memperbarui password.',
  })

  // kontrol tombol loading di sini, bukan chaining ke toastId
  p.finally(() => setSubmitting(false))
}

  return (
    <div className="container mx-auto py-8">
      <Card className="mx-auto max-w-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" /> Ganti Password
          </CardTitle>
          <CardDescription>
            Gunakan password kuat (8+ karakter, kombinasi huruf besar/kecil, angka, dan simbol).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="old-password">Password Lama</Label>
              <div className="relative">
                <Input
                  id="old-password"
                  type={show.old ? 'text' : 'password'}
                  value={oldPassword}
                  onChange={(e) => setOldPassword(e.target.value)}
                  required
                  disabled={submitting}
                />
                <button
                  type="button"
                  onClick={() => setShow((s) => ({ ...s, old: !s.old }))}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
                  aria-label="toggle old password visibility"
                >
                  {show.old ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="new-password">Password Baru</Label>
              <div className="relative">
                <Input
                  id="new-password"
                  type={show.new ? 'text' : 'password'}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  disabled={submitting}
                />
                <button
                  type="button"
                  onClick={() => setShow((s) => ({ ...s, new: !s.new }))}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
                  aria-label="toggle new password visibility"
                >
                  {show.new ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {/* strength meter */}
              <div className="space-y-1">
                <Progress value={complexity} />
                <p className="text-xs text-muted-foreground">
                  Kekuatan: {complexity < 35 ? 'Lemah' : complexity < 70 ? 'Sedang' : 'Kuat'}
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirm-password">Konfirmasi Password Baru</Label>
              <div className="relative">
                <Input
                  id="confirm-password"
                  type={show.confirm ? 'text' : 'password'}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  disabled={submitting}
                />
                <button
                  type="button"
                  onClick={() => setShow((s) => ({ ...s, confirm: !s.confirm }))}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
                  aria-label="toggle confirm password visibility"
                >
                  {show.confirm ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            <Button type="submit" className="w-full" disabled={submitting}>
              {submitting ? 'Menyimpan…' : 'Simpan Perubahan'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
