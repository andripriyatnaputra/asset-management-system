import { useEffect, useMemo, useState, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import apiClient from '@/services/api'
import { toast } from 'sonner'
import type { TicketInfo, PaginationData } from '@/types'

import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Pagination } from '@/components/ui/pagination'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import CreateTicketModal from '@/components/CreateTicketModal'

type SortKey = 'id' | 'subject' | 'updated_at'
type SortDir = 'asc' | 'desc'

export default function TicketsPage() {
  const [tickets, setTickets] = useState<TicketInfo[]>([])
  const [pagination, setPagination] = useState<PaginationData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [currentPage, setCurrentPage] = useState(1)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  const [statusFilter, setStatusFilter] = useState('all')
  const [priorityFilter, setPriorityFilter] = useState('all')
  const [query, setQuery] = useState('')

  const [sortKey, setSortKey] = useState<SortKey>('updated_at')
  const [sortDir, setSortDir] = useState<SortDir>('desc')

  // init from URL
  useEffect(() => {
    setStatusFilter(searchParams.get('status') || 'all')
    setPriorityFilter(searchParams.get('priority') || 'all')
    setQuery(searchParams.get('q') || '')
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // write to URL
  useEffect(() => {
    const next = new URLSearchParams(searchParams)
    next.set('status', statusFilter)
    next.set('priority', priorityFilter)
    next.set('q', query)
    setSearchParams(next, { replace: true })
  }, [statusFilter, priorityFilter, query])

  const fetchTickets = useCallback((pageToFetch: number) => {
    setIsLoading(true)
    const params = {
      page: pageToFetch.toString(),
      limit: '10',
      ...(statusFilter !== 'all' && { status: statusFilter }),
      ...(priorityFilter !== 'all' && { priority: priorityFilter }),
      ...(query.trim() && { q: query.trim() }),
    }
    const qs = new URLSearchParams(params).toString()

    apiClient.get(`/tickets?${qs}`)
      .then(res => {
        setTickets(res.data.data ?? [])
        setPagination(res.data.pagination ?? null)
      })
      .catch(() => toast.error('Gagal memuat data tiket.'))
      .finally(() => setIsLoading(false))
  }, [statusFilter, priorityFilter, query])

  useEffect(() => { fetchTickets(currentPage) }, [currentPage, fetchTickets])
  useEffect(() => { setCurrentPage(1); fetchTickets(1) }, [statusFilter, priorityFilter, query, fetchTickets])

  const filteredAndSorted = useMemo(() => {
    const sorted = [...tickets].sort((a, b) => {
      const av = sortKey === 'id'
        ? a.id
        : sortKey === 'subject'
        ? (a.subject || '').toLowerCase()
        : new Date(a.updated_at).getTime()
      const bv = sortKey === 'id'
        ? b.id
        : sortKey === 'subject'
        ? (b.subject || '').toLowerCase()
        : new Date(b.updated_at).getTime()
      if (av < bv) return sortDir === 'asc' ? -1 : 1
      if (av > bv) return sortDir === 'asc' ? 1 : -1
      return 0
    })
    return sorted
  }, [tickets, sortKey, sortDir])

  const toggleSort = (key: SortKey) => {
    if (key === sortKey) setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    else { setSortKey(key); setSortDir('asc') }
  }

  const getStatusVariant = (status: string) => {
    switch (status.toLowerCase()) {
      case 'open': return 'default'
      case 'in progress': return 'secondary'
      case 'closed': return 'outline'
      default: return 'secondary'
    }
  }

  const handleExport = () => {
    const csvQs = new URLSearchParams({
      ...(statusFilter !== 'all' && { status: statusFilter }),
      ...(priorityFilter !== 'all' && { priority: priorityFilter }),
      ...(query.trim() && { q: query.trim() }),
      format: 'csv'
    }).toString()
    window.location.href = `/api/v1/tickets?${csvQs}`
  }

  const handleSuccess = () => {
    setIsCreateModalOpen(false)
    fetchTickets(currentPage)
  }

  return (
    <div className="container mx-auto py-8 space-y-6">
      <div className="flex flex-wrap gap-3 justify-between items-center">
        <h1 className="text-3xl font-bold">Help Desk Tiket</h1>
        <div className="flex gap-2">
          <Input placeholder="Cari ID / Subjek / Pelapor…" value={query} onChange={(e)=>setQuery(e.target.value)} className="w-64" />
          <Button variant="outline" onClick={handleExport}>Export CSV</Button>
          <Button onClick={() => setIsCreateModalOpen(true)}>+ Buat Tiket</Button>
        </div>
      </div>

      <div className="border bg-card p-4 rounded-lg flex flex-wrap items-center gap-3">
        <Select value={statusFilter} onValueChange={setStatusFilter}>
          <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Status" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            <SelectItem value="Open">Open</SelectItem>
            <SelectItem value="In Progress">In Progress</SelectItem>
            <SelectItem value="Closed">Closed</SelectItem>
          </SelectContent>
        </Select>
        <Select value={priorityFilter} onValueChange={setPriorityFilter}>
          <SelectTrigger className="w-[180px]"><SelectValue placeholder="Semua Prioritas" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Prioritas</SelectItem>
            <SelectItem value="Low">Low</SelectItem>
            <SelectItem value="Medium">Medium</SelectItem>
            <SelectItem value="High">High</SelectItem>
            <SelectItem value="Critical">Critical</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="border bg-card p-0 rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead onClick={() => toggleSort('id')} className="cursor-pointer text-muted-foreground">
                ID {sortKey === 'id' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
              </TableHead>
              <TableHead onClick={() => toggleSort('subject')} className="cursor-pointer text-muted-foreground">
                Subjek {sortKey === 'subject' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
              </TableHead>
              <TableHead className="text-muted-foreground">Dilaporkan Oleh</TableHead>
              <TableHead className="text-muted-foreground">Status</TableHead>
              <TableHead className="text-muted-foreground">Prioritas</TableHead>
              <TableHead>SLA Due</TableHead>
              <TableHead onClick={() => toggleSort('updated_at')} className="cursor-pointer text-muted-foreground">
                Update Terakhir {sortKey === 'updated_at' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={7} className="text-center h-24 text-muted-foreground">Memuat data…</TableCell></TableRow>
            ) : filteredAndSorted.length === 0 ? (
              <TableRow><TableCell colSpan={7} className="text-center h-24 text-muted-foreground">Tidak ada tiket.</TableCell></TableRow>
            ) : (
              filteredAndSorted.map((t) => (
                <TableRow key={t.id} onClick={() => navigate(`/tickets/${t.id}`)} className="cursor-pointer hover:bg-muted/50">
                  <TableCell className="font-mono">#{t.id}</TableCell>
                  <TableCell className="font-medium">{t.subject}</TableCell>
                  <TableCell>{t.created_by_employee_name}</TableCell>
                  <TableCell><Badge variant={getStatusVariant(t.status)}>{t.status}</Badge></TableCell>
                  <TableCell>{t.priority}</TableCell>
                  <TableCell>
                    {t.sla_due_at ? new Date(t.sla_due_at).toLocaleString('id-ID') : '-'}
                  </TableCell>
                  <TableCell>{new Date(t.updated_at).toLocaleString('id-ID')}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {pagination && (
        <Pagination
          currentPage={pagination.current_page}
          totalPages={pagination.total_pages}
          onPageChange={setCurrentPage}
        />
      )}

      <CreateTicketModal isOpen={isCreateModalOpen} onClose={() => setIsCreateModalOpen(false)} onSuccess={handleSuccess} />
    </div>
  )
}
