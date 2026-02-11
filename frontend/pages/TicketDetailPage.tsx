import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useParams } from 'react-router-dom'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import { jwtDecode } from 'jwt-decode'
import type { Status, TicketDetail } from '@/types'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import { Paperclip, AlertTriangle } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { Employee } from '@/types'

type Decoded = { role?: string }

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [ticket, setTicket] = useState<TicketDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [newComment, setNewComment] = useState('')
  const [files, setFiles] = useState<FileList | null>(null)
  const [now, setNow] = useState(() => Date.now())
  const [highlightedCommentId, setHighlightedCommentId] = useState<number | null>(null)

  const userRole = useMemo(() => {
    const token = localStorage.getItem('authToken')
    if (!token) return null
    try { return (jwtDecode(token) as Decoded).role ?? null } catch { return null }
  }, [])

  const fetchTicket = () => {
    if (!id) return
    setIsLoading(true)
    apiClient.get(`/tickets/${id}`)
      .then(res => setTicket(res.data))
      .catch(() => toast.error('Gagal memuat detail tiket.'))
      .finally(() => setIsLoading(false))
  }

  const refreshComments = async () => {
    try {
      const res = await apiClient.get(`/tickets/${id}/comments/recent`)
      setTicket((prev) => prev ? { ...prev, comments: res.data.comments } : null)
    } catch {
      console.warn("Gagal memuat komentar terbaru (fallback)")
    }
  }

  const [employees, setEmployees] = useState<Employee[]>([])
  //const [assignee, setAssignee] = useState<string>('')
  const [assignee, setAssignee] = useState("unassigned"); 
  
  useEffect(() => {
    if (!['super_admin', 'it_support'].includes(userRole || '')) return

    apiClient.get('/employees', { params: { page: 1, limit: 1000 } })
      .then(res => {
        const data = res.data?.data ?? res.data ?? []
        setEmployees(Array.isArray(data) ? data : [])
      })
      .catch(() => toast.error('Gagal memuat daftar karyawan'))
  }, [userRole])

  const handleAssign = (value: string) => {
    setAssignee(value)
    const empId = Number(value)
    if (!empId) return

    const p = apiClient.post(`/tickets/${id}/assign`, { assignee_id: empId })
    toast.promise(p, {
      loading: 'Mengubah penugasan...',
      success: () => {
        fetchTicket()
        return 'Tiket berhasil di-assign'
      },
      error: (e) => e?.response?.data?.error || 'Gagal mengubah penugasan.'
    })
  }


  useEffect(() => { fetchTicket() }, [id])
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(t)
  }, [])

  const dueAt = ticket?.sla_due_at ? new Date(ticket.sla_due_at).getTime() : null
  const remainMs = dueAt ? dueAt - now : null
  const breached = dueAt ? remainMs! < 0 : false

  const fmt = (ms: number) => {
    const s = Math.max(0, Math.floor(ms / 1000))
    const d = Math.floor(s / 86400); const h = Math.floor((s % 86400) / 3600); const m = Math.floor((s % 3600) / 60)
    return d > 0 ? `${d}d ${h}h ${m}m` : `${h}h ${m}m`
  }

  const fmtDate = (v?: string | null) => v ? new Date(v).toLocaleString('id-ID') : '-'

  // ============================================================
  // 🔹 WebSocket realtime listener
  // ============================================================
  useEffect(() => {
    const token = localStorage.getItem('authToken')
    if (!token || !id) return

    const ws = new WebSocket(`ws://${window.location.host}/api/v1/ws?token=${token}`)
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.event === `ticket_comment:${id}`) {
          const newComment = msg.data
          setTicket((prev) => {
            if (!prev) return prev
            const updated = { ...prev, comments: [...(prev.comments || []), newComment] }
            setHighlightedCommentId(newComment.id)
            setTimeout(() => setHighlightedCommentId(null), 2000)
            return updated
          })
        }
      } catch (e) {
        console.warn("Invalid WS message", e)
      }
    }
    ws.onclose = () => console.log("🔌 WS closed for ticket", id)
    return () => ws.close()
  }, [id])

  // ============================================================
  // 🔹 Tambah komentar + fallback refresh
  // ============================================================
  const handleCommentSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!newComment.trim() && (!files || files.length === 0))
      return toast.error('Tulis komentar atau lampirkan file.')

    if (ticket?.status === 'Closed')
      return toast.error('Tiket sudah ditutup, tidak dapat dikomentari.')

    let p: Promise<any>
    if (files && files.length > 0) {
      const form = new FormData()
      form.append('comment', newComment.trim())
      Array.from(files).forEach(f => form.append('attachments', f))
      p = apiClient.post(`/tickets/${id}/comments`, form, { headers: { 'Content-Type': 'multipart/form-data' } })
    } else {
      p = apiClient.post(`/tickets/${id}/comments`, { comment: newComment.trim() })
    }

    toast.promise(p, {
      loading: 'Mengirim...',
      success: () => {
        setNewComment('')
        setFiles(null)
        setTimeout(refreshComments, 1500) // fallback refresh (cadangan WS)
        return 'Komentar terkirim!'
      },
      error: (e) => e?.response?.data?.error || 'Gagal menambahkan komentar.'
    })
  }

  // ============================================================
  // 🔹 Update tiket (status / assignee / escalate)
  // ============================================================
  const update = (payload: any, successMsg = 'Tiket diperbarui!') => {
    if (!payload || Object.keys(payload).length === 0) return
    const p = apiClient.put(`/tickets/${id}`, payload)
    toast.promise(p, {
      loading: 'Memperbarui...',
      success: () => {
        fetchTicket()
        return successMsg
      },
      error: (e) => e?.response?.data?.error || 'Gagal memperbarui tiket.'
    })
}

  const getStatusVariant = (s: string) => {
    switch (s.toLowerCase()) {
      case 'open': return 'default'
      case 'in progress': return 'secondary'
      case 'resolved': return 'blue'     // 🌊 warna baru
      case 'closed': return 'destructive'
      default: return 'outline'
    }
  }

  if (isLoading) return <div className="p-8">Memuat detail tiket…</div>
  if (!ticket) return <div className="p-8">Ticket tidak ditemukan.</div>

  return (
    <div className="container mx-auto py-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-muted-foreground">Ticket #{ticket.id}</p>
          <h1 className="text-3xl font-bold">{ticket.subject}</h1>
          <div className="flex items-center space-x-4 mt-2">
            <Badge variant={getStatusVariant(ticket.status)}>{ticket.status}</Badge>
            {dueAt && (
            <Badge
              variant={
                ticket.status === 'Closed'
                  ? breached
                    ? 'destructive'
                    : 'secondary'
                  : breached
                  ? 'destructive'
                  : 'secondary'
              }
            >
              {ticket.status === 'Closed'
                ? breached
                  ? 'BREACHED'
                  : 'ON TIME'
                : breached
                ? `BREACHED ${fmt(Math.abs(remainMs!))}`
                : `DUE IN ${fmt(remainMs!)}`}
            </Badge>
          )}
          </div>
          {(ticket.escalation_level ?? 0) > 0 && (
            <div className="mt-2 flex items-center text-xs text-amber-600">
              <AlertTriangle size={14} className="mr-1" />
              Escalation level: {ticket.escalation_level}
            </div>
          )}
          <div className="mt-2 text-sm text-muted-foreground">
            {ticket.assigned_to_employee_name && (
              <p>Assigned to: <strong>{ticket.assigned_to_employee_name}</strong></p>
            )}
            {ticket.last_assigned_by_name && (
              <p>Last assigned by: <strong>{ticket.last_assigned_by_name}</strong></p>
            )}
            {ticket.last_assigned_at && (
              <p>Last assigned at: <strong>{fmtDate(ticket.last_assigned_at)}</strong></p>
            )}
          </div>
        </div>

        {/* 🔹 Action Buttons */}
        {['super_admin', 'it_support'].includes(userRole || '') && (
          <div className="flex flex-wrap gap-2">
             {/* 🔹 Dropdown Assign Ticket */}
           <Select value={assignee} onValueChange={handleAssign}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="Assign Ticket..." />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="unassigned">— Unassigned —</SelectItem>
              {employees.map(e => (
                <SelectItem key={e.id} value={String(e.id)}>
                  {e.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
            {ticket.status !== 'Closed' && (
              <Button variant="outline" onClick={() => update({ escalate: true }, 'Ticket berhasil di-escalate')}>
                Escalate Ticket
              </Button>
            )}
            {ticket.status === 'Open' && (
              <Button variant="secondary" onClick={() => update({ status: 'In Progress' }, 'Status diubah: In Progress')}>
                Start Progress
              </Button>
            )}
            {ticket.status === 'In Progress' && (
              <Button variant="blue" onClick={() => update({ status: 'Resolved' }, 'Status diubah: Resolved')}>
                Mark Resolved
              </Button>
            )}
            {ticket.status === 'Resolved' && (
              <Button variant="destructive" onClick={() => update({ status: 'Closed' }, 'Ticket ditutup (Closed)')}>
                Close Ticket
              </Button>
            )}
            {ticket.status === 'Closed' && (
              <Button variant="default" onClick={() => update({ status: 'Open' }, 'Ticket dibuka kembali (Reopen)')}>
                Reopen Ticket
              </Button>
            )}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6 mt-6">
        {/* Left: conversation */}
        <div className="md:col-span-2 lg:col-span-3 space-y-6">
          <Card>
            <CardHeader><CardTitle>Deskripsi Masalah</CardTitle></CardHeader>
            <CardContent><p className="whitespace-pre-wrap text-sm">{ticket.description || 'Tidak ada deskripsi.'}</p></CardContent>
          </Card>

          <div className="space-y-6">
            {ticket.comments?.map(c => (
              <div
                key={c.id}
                className={`flex items-start space-x-3 transition-all duration-300 ${highlightedCommentId === c.id ? 'bg-blue-50 rounded-lg p-2' : ''}`}
              >
                <Avatar><AvatarFallback>{c.employee_name.charAt(0)}</AvatarFallback></Avatar>
                <div className="flex-1">
                  <div className="border bg-card rounded-lg p-4">
                    <div className="flex justify-between items-center mb-1">
                      <p className="font-semibold text-sm">{c.employee_name}</p>
                      <p className="text-xs text-muted-foreground">{fmtDate(c.created_at)}</p>
                    </div>
                    <p className="text-sm whitespace-pre-wrap">{c.comment}</p>
                    {c.attachments && c.attachments.length > 0 && (
                      <div className="mt-3 space-y-1">
                        <p className="text-xs text-muted-foreground">Lampiran:</p>
                        <ul className="list-disc pl-5 space-y-1">
                          {c.attachments.map(a => (
                            <li key={a.id}>
                              <a href={(a.url ?? a.path) || '#'} className="underline" target="_blank" rel="noreferrer">
                                {a.filename}
                              </a>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>

          {ticket.status !== 'Closed' && (
            <Card>
              <CardHeader><CardTitle>Tambah Komentar</CardTitle></CardHeader>
              <CardContent>
                <form onSubmit={handleCommentSubmit} className="space-y-4">
                  <Textarea value={newComment} onChange={e => setNewComment(e.target.value)} placeholder="Tulis balasan…" />
                  <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                      <Paperclip className="h-4 w-4 text-muted-foreground" />
                      <Input type="file" multiple onChange={(e) => setFiles(e.target.files)} />
                    </div>
                    <Button type="submit" disabled={(ticket.status as Status) === 'Closed'}>Kirim</Button>
                  </div>
                </form>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Right: detail panel */}
        <div className="md:col-span-1 lg:col-span-1 space-y-4">
          <Card>
            <CardHeader><CardTitle>Detail Tiket</CardTitle></CardHeader>
            <CardContent className="space-y-4 text-sm">
              <div>
                <Label className="text-xs">Priority</Label>
                <p className="font-semibold">{ticket.priority}</p>
              </div>
              <div>
                <Label className="text-xs">Impact</Label>
                <p className="font-semibold">{ticket.impact || '-'}</p>
              </div>
              <div>
                <Label className="text-xs">Urgency</Label>
                <p className="font-semibold">{ticket.urgency || '-'}</p>
              </div>
              <div>
                <Label className="text-xs">SLA</Label>
                {ticket.sla_due_at ? (
                  <div
                    className={`font-semibold ${
                      breached ? 'text-destructive' : ticket.status === 'Closed' || ticket.status === 'Resolved' ? 'text-muted-foreground' : ''
                    }`}
                  >
                    {ticket.status === 'Closed' || ticket.status === 'Resolved' ? (
                      breached ? 'BREACHED' : 'ON TIME'
                    ) : breached ? (
                      <>BREACHED {fmt(Math.abs(remainMs!))}</>
                    ) : (
                      <>DUE IN {fmt(remainMs!)}</>
                    )}
                  </div>
                ) : (
                  <p className="text-muted-foreground">Tidak ditetapkan</p>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
