import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { Employee, AssetItem } from '@/types'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function CreateTicketModal({ isOpen, onClose, onSuccess }: Props) {
  const [subject, setSubject] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<'Low'|'Medium'|'High'|'Critical'>('Medium')

  const [category, setCategory] = useState('INCIDENT')
  const [service, setService] = useState('ENDPOINT')
  const [impact, setImpact] = useState<'Low'|'Medium'|'High'>('Medium')
  const [urgency, setUrgency] = useState<'Low'|'Medium'|'High'>('Medium')

  const [employees, setEmployees] = useState<Employee[]>([])
  const [assignee, setAssignee] = useState<string>('unassigned')

  const [assets, setAssets] = useState<AssetItem[]>([])
  const [relatedAssetId, setRelatedAssetId] = useState<string>('none')

  const [slaPreview, setSlaPreview] = useState<{policy_name:string, priority:string, response:number, resolve:number}|null>(null)

  // 🔹 Ambil role dan ID user dari localStorage/session
  const userRole = localStorage.getItem('userRole') || 'employee'
  const currentUserId = Number(localStorage.getItem('userId'))

  // ======================================================
  // 🔹 SLA Preview Auto Update
  // ======================================================
  useEffect(() => {
    if (!impact || !urgency) return
    apiClient.get('/sla-policies/preview', {
      params: { category_code: category, service_code: service, impact, urgency }
    })
      .then(res => setSlaPreview(res.data))
      .catch(() => setSlaPreview(null))
  }, [category, service, impact, urgency])

  // ======================================================
  // 🔹 Load Employees & Assets
  // ======================================================
  useEffect(() => {
    if (!isOpen) return
    apiClient.get('/employees')
      .then(r => setEmployees(r.data?.data ?? r.data ?? []))
      .catch(() => toast.error('Gagal memuat karyawan'))
    apiClient.get('/assets')
      .then(r => setAssets(r.data?.data ?? r.data ?? []))
      .catch(() => {/* opsional */})
  }, [isOpen])

  // ======================================================
  // 🔹 Auto-Map Category & Service dari Asset
  // ======================================================
  useEffect(() => {
    if (relatedAssetId === 'none') return
    const selected = assets.find(a => String(a.id) === relatedAssetId)
    if (!selected) return
    if (selected.category_code && selected.category_code !== category) setCategory(selected.category_code)
    if (selected.service_code && selected.service_code !== service) setService(selected.service_code)
  }, [relatedAssetId, assets])

  // ======================================================
  // 🔹 Submit Ticket
  // ======================================================
  const [submitting, setSubmitting] = useState(false)
  const submit = async () => {
    if (submitting) return
    if (!subject.trim()) return toast.error('Subjek wajib diisi.')

    const payload = {
      subject: subject.trim(),
      description: description.trim(),
      priority,
      category_code: category,
      service_code: service,
      impact,
      urgency,
      related_asset_id: relatedAssetId !== 'none' ? Number(relatedAssetId) : null,
      // hanya kirim assigned_to jika role IT / Admin
      assigned_to_employee_id: (userRole === 'it_support' || userRole === 'super_admin') && assignee !== 'unassigned'
        ? Number(assignee)
        : null,
    }

    setSubmitting(true)
    try {
      const promise = apiClient.post('/tickets', payload)
      toast.promise(promise, {
        loading: 'Membuat tiket...',
        success: () => {
          onSuccess()
          onClose()
          setSubject('')
          setDescription('')
          setRelatedAssetId('none')
          setAssignee('unassigned')
          return 'Tiket berhasil dibuat!'
        },
        error: (e) => e?.response?.data?.error || 'Gagal membuat tiket',
      })
      await promise
    } finally {
      setSubmitting(false)
    }
  }

  // ======================================================
  // 🔹 Render Form
  // ======================================================
  return (
    <Dialog open={isOpen} onOpenChange={(open)=>{ if(!open) onClose() }}>
      <DialogContent>
        <DialogHeader><DialogTitle>Buat Tiket Baru</DialogTitle></DialogHeader>

        <div className="grid gap-3">
          {/* --- Subject & Description --- */}
          <div>
            <Label>Subjek</Label>
            <Input value={subject} onChange={e=>setSubject(e.target.value)} />
          </div>
          <div>
            <Label>Deskripsi</Label>
            <Textarea value={description} onChange={e=>setDescription(e.target.value)} rows={4}/>
          </div>

          {/* --- Category & Service --- */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <Label>Category</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="INCIDENT">Incident</SelectItem>
                  <SelectItem value="REQUEST">Service Request</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Service</Label>
              <Select value={service} onValueChange={setService}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="ENDPOINT">End-user Device</SelectItem>
                  <SelectItem value="EMAIL">Email</SelectItem>
                  <SelectItem value="NETWORK">Network</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* --- Impact, Urgency, Priority --- */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div>
              <Label>Impact</Label>
              <Select value={impact} onValueChange={(v)=>setImpact(v as any)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Urgency</Label>
              <Select value={urgency} onValueChange={(v)=>setUrgency(v as any)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Priority (opsional)</Label>
              <Select value={priority} onValueChange={(v)=>setPriority(v as any)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                  <SelectItem value="Critical">Critical</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* --- SLA Preview --- */}
          {slaPreview && (
            <div className="border rounded-lg p-3 mt-3 bg-muted/30 text-xs text-muted-foreground">
              <p><b>SLA Policy:</b> {slaPreview.policy_name}</p>
              <p><b>Priority:</b> {slaPreview.priority}</p>
              <p>Response Due: {slaPreview.response} menit</p>
              <p>Resolve Due: {slaPreview.resolve} menit</p>
            </div>
          )}

          {/* --- Assignment & Asset --- */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Hanya tampil untuk IT Support & Super Admin */}
            {(userRole === 'it_support' || userRole === 'super_admin') && (
              <div>
                <Label>Assign ke</Label>
                <Select value={assignee} onValueChange={setAssignee}>
                  <SelectTrigger><SelectValue placeholder="Pilih assignee" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="unassigned">— Unassigned —</SelectItem>
                    {employees.map(e => (
                      <SelectItem key={e.id} value={String(e.id)}>{e.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div>
              <Label>Asset Terkait</Label>
              <Select value={relatedAssetId} onValueChange={setRelatedAssetId}>
                <SelectTrigger><SelectValue placeholder="Pilih asset" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">— Tidak ada —</SelectItem>
                  {assets
                    .filter(a => userRole === 'employee'
                      ? a.assigned_to_employee_id === currentUserId
                      : true)
                    .map(a => (
                      <SelectItem key={a.id} value={String(a.id)}>
                        {a.asset_tag} — {a.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Batal</Button>
          <Button onClick={submit} disabled={submitting}>
            {submitting ? 'Membuat…' : 'Buat Tiket'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
