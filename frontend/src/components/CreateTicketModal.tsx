import { useEffect, useState } from 'react'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { Employee, AssetItem } from '@/types'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

type Priority = 'Low' | 'Medium' | 'High' | 'Critical'
type ImpactUrgency = 'Low' | 'Medium' | 'High'

export default function CreateTicketModal({ isOpen, onClose, onSuccess }: Props) {
  const [subject, setSubject] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<Priority>('Medium')

  const [category, setCategory] = useState('INCIDENT')
  const [service, setService] = useState('ENDPOINT')
  const [impact, setImpact] = useState<ImpactUrgency>('Medium')
  const [urgency, setUrgency] = useState<ImpactUrgency>('Medium')

  const [employees, setEmployees] = useState<Employee[]>([])
  const [assignee, setAssignee] = useState<string>('unassigned')

  const [assets, setAssets] = useState<AssetItem[]>([])
  const [relatedAssetId, setRelatedAssetId] = useState<string>('none')

  const [slaPreview, setSlaPreview] = useState<{
    policy_name: string
    priority: string
    response: number
    resolve: number
  } | null>(null)

  // 🔹 Ambil role dan ID user dari localStorage/session
  const userRole = localStorage.getItem('userRole') || 'employee'
  const currentUserIdRaw = localStorage.getItem('userId')
  const currentUserId = currentUserIdRaw ? Number(currentUserIdRaw) : null

  const isIT = userRole === 'it_support' || userRole === 'super_admin'

  // helper: normalize response list agar selalu array
  const normalizeList = (r: any) => {
    if (Array.isArray(r?.data?.data)) return r.data.data
    if (Array.isArray(r?.data)) return r.data
    return []
  }

  // ======================================================
  // 🔹 SLA Preview Auto Update
  // ======================================================
  useEffect(() => {
    if (!isOpen) return
    if (!impact || !urgency) return

    apiClient
      .get('/sla-policies/preview', {
        params: { category_code: category, service_code: service, impact, urgency },
      })
      .then((res) => setSlaPreview(res?.data ?? null))
      .catch(() => setSlaPreview(null))
  }, [isOpen, category, service, impact, urgency])

  // ======================================================
  // 🔹 Load Employees & Assets (FIX pagination)
  // ======================================================
  useEffect(() => {
    if (!isOpen) return

    let cancelled = false

    const fetchAllEmployees = async () => {
      try {
        const res = await apiClient.get('/employees', {
          params: { page: 1, limit: 1000 },
        })
        if (!cancelled) setEmployees(normalizeList(res))
      } catch {
        if (!cancelled) {
          setEmployees([])
          toast.error('Gagal memuat karyawan')
        }
      }
    }

    const fetchAllAssets = async () => {
      try {
        const all: AssetItem[] = []

        const first = await apiClient.get('/assets', {
          params: { page: 1, limit: 10 },
        })

        all.push(...normalizeList(first))

        const totalPages: number =
          first?.data?.pagination?.total_pages ?? 1

        for (let page = 2; page <= totalPages; page++) {
          const res = await apiClient.get('/assets', {
            params: { page, limit: 10 },
          })
          all.push(...normalizeList(res))
        }

        if (!cancelled) setAssets(all)
      } catch {
        if (!cancelled) setAssets([])
      }
    }

    fetchAllEmployees()
    fetchAllAssets()

    return () => {
      cancelled = true
    }
  }, [isOpen])


  // ======================================================
  // 🔹 Auto-Map Category & Service dari Asset
  // ======================================================
  useEffect(() => {
    if (relatedAssetId === 'none') return
    const safeAssets = Array.isArray(assets) ? assets : []
    const selected = safeAssets.find((a) => String(a.id) === relatedAssetId)
    if (!selected) return

    if (selected.category_code && selected.category_code !== category) {
      setCategory(selected.category_code)
    }
    if (selected.service_code && selected.service_code !== service) {
      setService(selected.service_code)
    }
  }, [relatedAssetId, assets, category, service])

  // ======================================================
  // 🔹 Submit Ticket
  // ======================================================
  const [submitting, setSubmitting] = useState(false)

  const resetForm = () => {
    setSubject('')
    setDescription('')
    setPriority('Medium')
    setCategory('INCIDENT')
    setService('ENDPOINT')
    setImpact('Medium')
    setUrgency('Medium')
    setRelatedAssetId('none')
    setAssignee('unassigned')
    setSlaPreview(null)
  }

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
      assigned_to_employee_id:
        isIT && assignee !== 'unassigned' ? Number(assignee) : null,
    }

    setSubmitting(true)
    try {
      const promise = apiClient.post('/tickets', payload)

      toast.promise(promise, {
        loading: 'Membuat tiket...',
        success: () => {
          onSuccess()
          onClose()
          resetForm()
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
  const safeEmployees = Array.isArray(employees) ? employees : []
  const safeAssets = Array.isArray(assets) ? assets : []

  const visibleAssets =
    userRole === 'employee'
      ? safeAssets.filter((a) =>
          currentUserId ? a.assigned_to_employee_id === currentUserId : false
        )
      : safeAssets

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Buat Tiket Baru</DialogTitle>
        </DialogHeader>

        <div className="grid gap-3">
          {/* --- Subject & Description --- */}
          <div>
            <Label>Subjek</Label>
            <Input value={subject} onChange={(e) => setSubject(e.target.value)} />
          </div>

          <div>
            <Label>Deskripsi</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
            />
          </div>

          {/* --- Category & Service --- */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <Label>Category</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="INCIDENT">Incident</SelectItem>
                  <SelectItem value="REQUEST">Service Request</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label>Service</Label>
              <Select value={service} onValueChange={setService}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
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
              <Select value={impact} onValueChange={(v) => setImpact(v as any)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label>Urgency</Label>
              <Select value={urgency} onValueChange={(v) => setUrgency(v as any)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Low">Low</SelectItem>
                  <SelectItem value="Medium">Medium</SelectItem>
                  <SelectItem value="High">High</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label>Priority (opsional)</Label>
              <Select value={priority} onValueChange={(v) => setPriority(v as any)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
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
              <p>
                <b>SLA Policy:</b> {slaPreview.policy_name}
              </p>
              <p>
                <b>Priority:</b> {slaPreview.priority}
              </p>
              <p>Response Due: {slaPreview.response} menit</p>
              <p>Resolve Due: {slaPreview.resolve} menit</p>
            </div>
          )}

          {/* --- Assignment & Asset --- */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Hanya tampil untuk IT Support & Super Admin */}
            {isIT && (
              <div>
                <Label>Assign ke</Label>
                <Select value={assignee} onValueChange={setAssignee}>
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih assignee" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="unassigned">— Unassigned —</SelectItem>
                    {safeEmployees.map((e) => (
                      <SelectItem key={e.id} value={String(e.id)}>
                        {e.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div>
              <Label>Asset Terkait</Label>
              <Select value={relatedAssetId} onValueChange={setRelatedAssetId}>
                <SelectTrigger>
                  <SelectValue placeholder="Pilih asset" />
                </SelectTrigger>
                <SelectContent className="max-h-72 overflow-y-auto">
                  <SelectItem value="none">— Tidak ada —</SelectItem>

                  {visibleAssets.length === 0 && (
                    <div className="text-xs text-muted-foreground px-2 py-2">
                      Tidak ada asset terkait.
                    </div>
                  )}

                  {visibleAssets.map((a) => (
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
          <Button variant="outline" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={submit} disabled={submitting}>
            {submitting ? 'Membuat…' : 'Buat Tiket'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
