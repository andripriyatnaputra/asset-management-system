import { useEffect, useState, useCallback, useRef } from 'react'
import apiClient from '@/services/api'
import { Bell, Check, CheckCheck, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { toast } from 'sonner'

interface Notification {
  id: number
  type: string
  title: string
  message: string
  entity_type?: string
  entity_id?: number
  is_read: boolean
  created_at: string
}

interface WSNotifDetail {
  type: string
  title: string
  message: string
  entity_type?: string
  entity_id?: number
}

const TYPE_COLORS: Record<string, string> = {
  license_expiry:  'bg-orange-100 text-orange-700',
  dr_test_due:     'bg-blue-100 text-blue-700',
  evidence_expired:'bg-red-100 text-red-700',
  ticket_assigned: 'bg-green-100 text-green-700',
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'Baru saja'
  if (mins < 60) return `${mins} menit lalu`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs} jam lalu`
  return `${Math.floor(hrs / 24)} hari lalu`
}

export default function NotificationBell() {
  const [unreadCount, setUnreadCount]     = useState(0)
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [open, setOpen]                   = useState(false)
  const [loading, setLoading]             = useState(false)
  const pollRef   = useRef<ReturnType<typeof setInterval> | null>(null)
  const wsActive  = useRef(false)   // true once first WS push is received
  const openRef   = useRef(open)
  openRef.current = open

  const fetchUnreadCount = useCallback(() => {
    apiClient.get('/notifications/unread-count')
      .then(res => setUnreadCount(res.data.unread_count ?? 0))
      .catch(() => {})
  }, [])

  const fetchNotifications = useCallback(() => {
    setLoading(true)
    apiClient.get('/notifications?limit=15')
      .then(res => setNotifications(res.data.data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  // Poll every 60s as fallback (halved from 30s — WS is primary)
  useEffect(() => {
    fetchUnreadCount()
    pollRef.current = setInterval(() => {
      if (!wsActive.current) fetchUnreadCount()
    }, 60000)
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
  }, [fetchUnreadCount])

  // Subscribe to realtime WS push from useWebSocket hook (via CustomEvent)
  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent<WSNotifDetail>).detail
      wsActive.current = true           // WS is working — polling becomes backup only

      // Bump the badge immediately
      setUnreadCount(prev => prev + 1)

      // Show toast
      toast(detail?.title ?? 'Notifikasi baru', {
        description: detail?.message,
        duration: 5000,
        position: 'top-right',
      })

      // If dropdown is open, refresh the list so the new item appears
      if (openRef.current) fetchNotifications()
    }

    window.addEventListener('ws:notification', handler)
    return () => window.removeEventListener('ws:notification', handler)
  }, [fetchNotifications])

  // Fetch list when dropdown opens
  useEffect(() => {
    if (open) fetchNotifications()
  }, [open, fetchNotifications])

  const markRead = (id: number) => {
    apiClient.put(`/notifications/${id}/read`)
      .then(() => {
        setNotifications(prev => prev.map(n => n.id === id ? { ...n, is_read: true } : n))
        setUnreadCount(prev => Math.max(0, prev - 1))
      })
      .catch(() => {})
  }

  const markAllRead = () => {
    apiClient.put('/notifications/read-all')
      .then(() => {
        setNotifications(prev => prev.map(n => ({ ...n, is_read: true })))
        setUnreadCount(0)
        toast.success('Semua notifikasi ditandai sudah dibaca')
      })
      .catch(() => toast.error('Gagal menandai notifikasi'))
  }

  const deleteNotif = (id: number, wasUnread: boolean) => {
    apiClient.delete(`/notifications/${id}`)
      .then(() => {
        setNotifications(prev => prev.filter(n => n.id !== id))
        if (wasUnread) setUnreadCount(prev => Math.max(0, prev - 1))
      })
      .catch(() => {})
  }

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative" aria-label="Notifikasi">
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-[10px] font-bold text-white">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-96 p-0" sideOffset={8}>
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <DropdownMenuLabel className="p-0 font-semibold">
            Notifikasi
            {unreadCount > 0 && (
              <span className="ml-2 rounded-full bg-red-100 text-red-700 text-xs px-1.5 py-0.5">
                {unreadCount} baru
              </span>
            )}
          </DropdownMenuLabel>
          {unreadCount > 0 && (
            <Button variant="ghost" size="sm" className="h-7 text-xs gap-1" onClick={markAllRead}>
              <CheckCheck className="h-3.5 w-3.5" /> Tandai semua
            </Button>
          )}
        </div>

        {/* List */}
        <div className="max-h-96 overflow-y-auto">
          {loading ? (
            <div className="py-8 text-center text-sm text-muted-foreground">Memuat...</div>
          ) : notifications.length === 0 ? (
            <div className="py-10 text-center">
              <Bell className="h-8 w-8 mx-auto text-muted-foreground/40 mb-2" />
              <p className="text-sm text-muted-foreground">Tidak ada notifikasi</p>
            </div>
          ) : notifications.map(n => (
            <div
              key={n.id}
              className={`group flex gap-3 px-4 py-3 border-b last:border-b-0 hover:bg-muted/50 transition-colors ${
                !n.is_read ? 'bg-blue-50/60 dark:bg-blue-950/20' : ''
              }`}
            >
              <div className="mt-1.5 flex-shrink-0">
                {!n.is_read
                  ? <span className="h-2 w-2 rounded-full bg-blue-500 block" />
                  : <span className="h-2 w-2 rounded-full bg-transparent block" />
                }
              </div>

              <div className="flex-1 min-w-0">
                <div className="flex items-start justify-between gap-2">
                  <p className={`text-sm leading-snug ${!n.is_read ? 'font-medium' : ''}`}>{n.title}</p>
                  <span className={`flex-shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium ${
                    TYPE_COLORS[n.type] ?? 'bg-gray-100 text-gray-600'
                  }`}>
                    {n.type.replace(/_/g, ' ')}
                  </span>
                </div>
                <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{n.message}</p>
                <p className="mt-1 text-[11px] text-muted-foreground/60">{timeAgo(n.created_at)}</p>
              </div>

              <div className="flex flex-col gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
                {!n.is_read && (
                  <button
                    onClick={() => markRead(n.id)}
                    className="p-1 rounded hover:bg-green-100 text-green-600"
                    title="Tandai sudah dibaca"
                  >
                    <Check className="h-3.5 w-3.5" />
                  </button>
                )}
                <button
                  onClick={() => deleteNotif(n.id, !n.is_read)}
                  className="p-1 rounded hover:bg-red-100 text-red-500"
                  title="Hapus"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>

        {notifications.length > 0 && (
          <>
            <DropdownMenuSeparator />
            <div className="px-4 py-2 text-center">
              <Button variant="ghost" size="sm" className="text-xs w-full" onClick={fetchNotifications}>
                Muat ulang
              </Button>
            </div>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
