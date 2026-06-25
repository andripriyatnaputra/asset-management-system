import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, act, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// ── Mocks ──────────────────────────────────────────────────────────────────

vi.mock('@/services/api', () => ({
  default: {
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('sonner', () => ({
  toast: Object.assign(vi.fn(), {
    success: vi.fn(),
    error: vi.fn(),
  }),
}))

// shadcn DropdownMenu: render langsung tanpa portal agar query DOM bisa menemukan konten
vi.mock('@/components/ui/dropdown-menu', () => ({
  DropdownMenu: ({ children, open, onOpenChange }: any) => (
    <div data-testid="dropdown-root" onClick={() => onOpenChange?.(!open)}>{children}</div>
  ),
  DropdownMenuTrigger: ({ children }: any) => <div data-testid="trigger">{children}</div>,
  DropdownMenuContent: ({ children }: any) => <div data-testid="content">{children}</div>,
  DropdownMenuLabel: ({ children, className }: any) => <div className={className}>{children}</div>,
  DropdownMenuSeparator: () => <hr />,
}))

vi.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, ...rest }: any) => (
    <button onClick={onClick} disabled={disabled} {...rest}>{children}</button>
  ),
}))

import apiClient from '@/services/api'
import { toast } from 'sonner'
import NotificationBell from './NotificationBell'

// ── Helpers ────────────────────────────────────────────────────────────────

function mockUnreadCount(n: number) {
  vi.mocked(apiClient.get).mockImplementation((url: string) => {
    if (url.includes('unread-count')) return Promise.resolve({ data: { unread_count: n } })
    if (url.includes('/notifications')) return Promise.resolve({ data: { data: [] } })
    return Promise.resolve({ data: {} })
  })
}

function dispatchWSNotif(detail: object) {
  window.dispatchEvent(new CustomEvent('ws:notification', { detail }))
}

// ── Tests ──────────────────────────────────────────────────────────────────

describe('NotificationBell', () => {
  beforeEach(() => {
    mockUnreadCount(0)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('merender tombol bell', async () => {
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    expect(screen.getByRole('button', { name: /notifikasi/i })).toBeInTheDocument()
  })

  it('tidak menampilkan badge jika unread = 0', async () => {
    mockUnreadCount(0)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    expect(screen.queryByText(/^\d+$/)).toBeNull()
  })

  it('menampilkan badge dengan angka jika unread > 0', async () => {
    mockUnreadCount(5)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    await waitFor(() => expect(screen.getByText('5')).toBeInTheDocument())
  })

  it('menampilkan "99+" jika unread > 99', async () => {
    mockUnreadCount(120)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    await waitFor(() => expect(screen.getByText('99+')).toBeInTheDocument())
  })

  it('WS push menaikkan unread count dan memunculkan toast', async () => {
    mockUnreadCount(2)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    await waitFor(() => expect(screen.getByText('2')).toBeInTheDocument())

    act(() => {
      dispatchWSNotif({ title: 'Lisensi hampir expired', message: 'Harap perbarui segera' })
    })

    await waitFor(() => expect(screen.getByText('3')).toBeInTheDocument())
    expect(toast).toHaveBeenCalledWith(
      'Lisensi hampir expired',
      expect.objectContaining({ description: 'Harap perbarui segera' })
    )
  })

  it('WS push dengan title kosong menggunakan fallback text', async () => {
    mockUnreadCount(0)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })

    act(() => { dispatchWSNotif({}) })

    await waitFor(() => expect(screen.getByText('1')).toBeInTheDocument())
    expect(toast).toHaveBeenCalledWith(
      'Notifikasi baru',
      expect.objectContaining({ duration: 5000 })
    )
  })

  it('polling dipanggil saat komponen mount', async () => {
    mockUnreadCount(3)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })
    // fetchUnreadCount dipanggil sekali saat mount
    expect(apiClient.get).toHaveBeenCalledWith('/notifications/unread-count')
  })

  it('menampilkan pesan kosong saat list notifikasi kosong dan dropdown dibuka', async () => {
    mockUnreadCount(0)
    render(<NotificationBell />)
    await act(async () => { await Promise.resolve() })

    await userEvent.click(screen.getByTestId('dropdown-root'))
    await act(async () => { await Promise.resolve() })

    await waitFor(() =>
      expect(screen.getByText(/tidak ada notifikasi/i)).toBeInTheDocument()
    )
  })
})
