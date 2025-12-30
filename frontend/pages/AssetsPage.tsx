import { useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import apiClient from '@/services/api'
import {
  Plus, Search, Info, Edit3, Trash2,
  FileText, ClipboardCheck, ChevronLeft, ChevronRight, Layers, FileDown,
  UserPlus
} from 'lucide-react'
import {
  Card, CardHeader, CardTitle, CardContent
} from '@/components/ui/card'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Tooltip, TooltipProvider, TooltipTrigger, TooltipContent
} from "@/components/ui/tooltip"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"

import AssetFormModal from '@/components/AssetFormModal'
import AssetDetailsModal from '@/components/AssetDetailsModal'
import AssignAssetModal from '@/components/AssignAssetModal'
import ReturnAssetModal from '@/components/ReturnAssetModal'
import MaintenanceLogModal from '@/components/MaintenanceLogModal'
import type { Asset } from '@/types'

type AssetRow = Asset & {
  asset_type_name?: string | null
  owner_department_name?: string | null
  current_location_text?: string | null
  assigned_to_employee_name?: string | null
  cost_center_name?: string | null
  //governance_score?: number | null
  disposed?: boolean | null
  disposal_date?: string | null
  disposed_approved_by?: number | null
  asset_health_score?: number | null
}

const fmtIDR = (v?: number | null) =>
  typeof v === 'number'
    ? v.toLocaleString('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 })
    : '-'

const pageSizeOptions = [10, 25, 50]

export default function AssetsPage() {
  const [assets, setAssets] = useState<AssetRow[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [q, setQ] = useState("")
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // filters
  const [statusFilter, setStatusFilter] = useState<string>("")
  const [typeFilter, setTypeFilter] = useState<string>("")
  const [assetTypes, setAssetTypes] = useState<{ id: number; name: string }[]>([])
  const [departments, setDepartments] = useState<any[]>([])
  const [locations, setLocations] = useState<any[]>([])

  // pagination & sort
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [totalPages, setTotalPages] = useState(1)
  const [totalRecords, setTotalRecords] = useState(0)
  const [sortBy, setSortBy] = useState('updated_at')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

  // modals
  const [openForm, setOpenForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [selectedAsset, setSelectedAsset] = useState<AssetRow | null>(null)
  const [assignId, setAssignId] = useState<number | null>(null)
  const [returnId, setReturnId] = useState<number | null>(null)
  const [logId, setLogId] = useState<number | null>(null)

  // alert dialog
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<AssetRow | null>(null)

  const normalizeArray = (res: any) => {
      if (Array.isArray(res)) return res
      if (Array.isArray(res?.data)) return res.data
      return []
    }

    const loadReferences = async () => {
      try {
        const [t, d, l] = await Promise.all([
          apiClient.get("/asset-types"),
          apiClient.get("/departments"),
          apiClient.get("/locations"),
        ])

        setAssetTypes(normalizeArray(t.data))
        setDepartments(normalizeArray(d.data))
        setLocations(normalizeArray(l.data))
      } catch (err) {
        console.error("Gagal memuat referensi aset:", err)
        setAssetTypes([])
        setDepartments([])
        setLocations([])
      }
    }

  const loadAssets = async (customQuery?: string) => {
    setIsLoading(true)
    try {
      const ps = new URLSearchParams()
      ps.set("page", String(page))
      ps.set("limit", String(pageSize))
      ps.set("sort_by", sortBy)
      ps.set("sort_order", sortOrder)
      const queryStr = customQuery ?? q.trim()
      if (queryStr) ps.set("q", queryStr)
      if (statusFilter) ps.set("status", statusFilter)
      if (typeFilter) ps.set("type_id", typeFilter)
      const r = await apiClient.get(`/assets?${ps.toString()}`)
      const list = r.data?.data ?? r.data ?? []
      const pg = r.data?.pagination ?? {}
      setAssets(Array.isArray(list) ? list : [])
      setTotalPages(pg.total_pages ?? 1)
      setTotalRecords(pg.total_records ?? list.length)
    } catch {
      toast.error("Gagal memuat aset.")
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => { loadReferences() }, [])
  useEffect(() => { loadAssets() }, [page, pageSize, sortBy, sortOrder, q, statusFilter, typeFilter])

  const onChangeSearch = (val: string) => {
    setQ(val)
    if (searchTimer.current) clearTimeout(searchTimer.current)
    searchTimer.current = setTimeout(() => {
      setPage(1)
      loadAssets(val)
    }, 400)
  }

  const handleSort = (col: string) => {
    let newOrder: "asc" | "desc" = "asc"
    if (sortBy === col) newOrder = sortOrder === "asc" ? "desc" : "asc"
    setSortBy(col)
    setSortOrder(newOrder)
    setPage(1)
    loadAssets()
  }

  const sortArrow = (col: string) => sortBy === col ? (sortOrder === "asc" ? " ↑" : " ↓") : ""

  const handleExportCSV = async () => {
    try {
      const ps = new URLSearchParams()
      ps.set('page', '1')
      ps.set('limit', String(totalRecords || 1000))
      ps.set('sort_by', sortBy)
      ps.set('sort_order', sortOrder)
      if (q.trim()) ps.set('q', q.trim())
      const r = await apiClient.get(`/assets?${ps.toString()}`)
      const list: AssetRow[] = r.data?.data ?? []
      const header = ['ID', 'Tag', 'Nama', 'Tipe', 'Status', 'Departemen', 'Cost Center', 'Lokasi', 'Assigned To', 'Harga', 'Governance', 'Disposed']
      const rows = list.map(a => [
        a.id, a.asset_tag, a.name,
        a.asset_type_name ?? '-',
        a.status,
        a.owner_department_name ?? '-',
        a.cost_center_name ?? '-',
        a.current_location_text ?? '-',
        a.assigned_to_employee_name ?? '-',
        fmtIDR(a.initial_price ?? a.purchase_cost),
        //a.governance_score ?? '-',
        a.disposed ? 'Yes' : 'No'
      ])
      const csv = [header, ...rows].map(r => r.join(',')).join('\n')
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', 'assets.csv')
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
    } catch {
      toast.error('Gagal export CSV')
    }
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/assets/${deleteItem.id}`)
      toast.success(`Aset "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      loadAssets()
    } catch {
      toast.error("Gagal menghapus aset.")
    }
  }

  async function handleVerifyCompliance(assetId: number) {
    try {
      await apiClient.post(`/assets/${assetId}/verify-compliance`)
      toast.success("Verifikasi compliance berhasil")
      loadAssets()
    } catch (err: any) {
      console.error("Verify error:", err)
      toast.error("Gagal memverifikasi compliance")
    }
  }

  const PageInfo = useMemo(() => {
    const start = (page - 1) * pageSize + 1
    const end = Math.min(page * pageSize, totalRecords)
    return totalRecords ? `${start}–${end} dari ${totalRecords}` : '0 data'
  }, [page, pageSize, totalRecords])

  // ============================ UI ============================
  return (
    <div className="p-6 space-y-6">
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            <Layers size={18} /> Manajemen Aset
          </CardTitle>
          <div className="flex items-center gap-2">
            <Button onClick={() => setOpenForm(true)}>
              <Plus size={16} className="mr-1" /> Tambah Aset
            </Button>
            <Button variant="outline" onClick={handleExportCSV}>
              <FileDown size={16} className="mr-1" /> Export CSV
            </Button>
          </div>
        </CardHeader>

        <CardContent>
          {/* Filter Bar */}
          <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
            <div className="flex items-center gap-2 flex-grow min-w-[280px]">
              <div className="relative flex-grow">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Cari aset (nama, tag, serial...)"
                  value={q}
                  onChange={(e) => onChangeSearch(e.target.value)}
                  className="pl-8 w-full"
                />
              </div>
              <select
                value={statusFilter}
                onChange={(e) => { setStatusFilter(e.target.value); setPage(1); loadAssets() }}
                className="border rounded-md px-2 py-1 text-sm h-9 bg-background"
              >
                <option value="">Semua Status</option>
                <option value="in_stock">In Stock</option>
                <option value="assigned">Assigned</option>
                <option value="maintenance">Maintenance</option>
                <option value="retired">Retired</option>
                <option value="disposed">Disposed</option>
              </select>
            </div>

            <div className="flex items-center gap-2">
              <select
                value={typeFilter}
                onChange={(e) => { setTypeFilter(e.target.value); setPage(1); loadAssets() }}
                className="border rounded-md px-2 py-1 text-sm h-9 bg-background"
              >
                <option value="">Semua Tipe</option>
                {assetTypes.map((t) => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
              <Select
                value={String(pageSize)}
                onValueChange={(v) => { setPageSize(Number(v)); setPage(1) }}
              >
                <SelectTrigger className="w-[120px] h-9">
                  <SelectValue placeholder="Tampil" />
                </SelectTrigger>
                <SelectContent>
                  {pageSizeOptions.map(size => (
                    <SelectItem key={size} value={String(size)}>
                      {size} / halaman
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Table */}
          <div className="rounded-lg border overflow-x-auto">
            <Table className="min-w-[1150px] text-sm">
              <TableHeader>
                <TableRow className="bg-muted/40">
                  <TableHead onClick={() => handleSort('asset_tag')} className="cursor-pointer select-none w-[100px]">
                    Tag{sortArrow('asset_tag')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('name')} className="cursor-pointer select-none min-w-[150px]">
                    Nama{sortArrow('name')}
                  </TableHead>
                  <TableHead className="min-w-[100px]">Tipe</TableHead>
                  <TableHead className="text-center w-[90px]">Status</TableHead>
                  <TableHead className="min-w-[120px]">Departemen</TableHead>
                  <TableHead className="min-w-[110px]">Cost Center</TableHead>
                  <TableHead className="min-w-[150px]">Lokasi</TableHead>
                  <TableHead className="min-w-[130px]">Assigned To</TableHead>
                  <TableHead className="text-right w-[100px]">Harga</TableHead>
                  <TableHead className="text-center w-[100px]">Health</TableHead>
                  <TableHead className="text-center w-[120px]">Compliance</TableHead>
                  <TableHead className="text-center w-[90px]">Disposal</TableHead>
                  <TableHead className="text-right w-[220px]">Aksi</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell colSpan={13} className="text-center py-6 text-muted-foreground">
                      Memuat data...
                    </TableCell>
                  </TableRow>
                ) : assets.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={13} className="text-center py-6 text-muted-foreground">
                      Tidak ada data aset.
                    </TableCell>
                  </TableRow>
                ) : (
                  assets.map((a) => (
                    <TableRow key={a.id} className="hover:bg-accent/30 transition-all">
                      <TableCell className="font-mono text-xs">{a.asset_tag}</TableCell>
                      <TableCell className="font-medium">{a.name}</TableCell>
                      <TableCell>{a.asset_type_name || '-'}</TableCell>

                      {/* STATUS */}
                      <TableCell className="text-center">
                        <Badge
                          variant="outline"
                          className={`px-2 py-0.5 text-xs border-none capitalize ${
                            a.status === 'in_stock'
                              ? 'bg-blue-50 text-blue-700'
                              : a.status === 'assigned'
                              ? 'bg-green-50 text-green-700'
                              : a.status === 'maintenance'
                              ? 'bg-yellow-50 text-yellow-800'
                              : a.status === 'retired'
                              ? 'bg-gray-200 text-gray-700'
                              : 'bg-red-50 text-red-700'
                          }`}
                        >
                          {a.status}
                        </Badge>
                      </TableCell>

                      <TableCell>{a.owner_department_name || '-'}</TableCell>
                      <TableCell>{a.cost_center_name || '-'}</TableCell>
                      <TableCell className="max-w-[150px] truncate">{a.current_location_text || '-'}</TableCell>
                      <TableCell>{a.assigned_to_employee_name || '-'}</TableCell>
                      <TableCell className="text-right font-medium whitespace-nowrap">
                        {fmtIDR(a.initial_price ?? a.purchase_cost)}
                      </TableCell>

                      {/* GOVERNANCE */}
                      <TableCell className="text-center">
                        {a.asset_health_score != null ? (
                          <div className="flex flex-col items-center gap-1">
                            {/* progress bar */}
                            <div className="w-full max-w-[80px] h-2 bg-gray-200 rounded-full overflow-hidden">
                              <div
                                className={`
                                  h-2 rounded-full
                                  ${
                                    a.asset_health_score >= 80
                                      ? "bg-green-500"
                                      : a.asset_health_score >= 50
                                      ? "bg-yellow-500"
                                      : "bg-red-500"
                                  }
                                `}
                                style={{ width: `${a.asset_health_score}%` }}
                              ></div>
                            </div>
                            <span className="text-xs font-medium text-muted-foreground">
                              {Number(a.asset_health_score).toFixed(0)}%
                            </span>
                          </div>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>

                      {/* COMPLIANCE */}
                      <TableCell className="text-center">
                        {a.compliance_flag === null ? (
                          <Badge className="bg-yellow-100 text-yellow-800 border-none">Pending</Badge>
                        ) : a.compliance_flag ? (
                          <Badge className="bg-green-100 text-green-700 border-none">Compliant</Badge>
                        ) : (
                          <Badge className="bg-red-100 text-red-700 border-none">Non-Compliant</Badge>
                        )}
                      </TableCell>

                      {/* DISPOSAL */}
                      <TableCell className="text-center text-xs">
                        {a.disposed ? (
                          <div className="flex flex-col items-center gap-0.5">
                            <span className="text-red-600 font-medium">Disposed</span>
                            {a.disposal_date && (
                              <span className="text-muted-foreground">
                                {new Date(a.disposal_date).toLocaleDateString('id-ID')}
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-muted-foreground">Active</span>
                        )}
                      </TableCell>

                      {/* ACTIONS */}
                      <TableCell className="text-right whitespace-nowrap">
                        <TooltipProvider>
                          <div className="flex flex-wrap justify-end gap-1">
                            {/* Assign / Return */}
                            {a.status === 'in_stock' ? (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    size="sm"
                                    variant="secondary"
                                    onClick={() => setAssignId(a.id)}
                                    className="flex items-center gap-1 text-xs"
                                  >
                                    <UserPlus className="h-3 w-3" /> Assign
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>Assign aset ke karyawan</TooltipContent>
                              </Tooltip>
                            ) : (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    size="sm"
                                    variant="secondary"
                                    onClick={() => setReturnId(a.id)}
                                    className="flex items-center gap-1 text-xs"
                                  >
                                    <UserPlus className="h-3 w-3 rotate-180" /> Return
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>Kembalikan aset ke stok</TooltipContent>
                              </Tooltip>
                            )}

                            {/* Verify */}
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="outline"
                                  onClick={() => handleVerifyCompliance(a.id)}
                                  className="h-7 w-7"
                                >
                                  <ClipboardCheck size={14} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Verify Compliance</TooltipContent>
                            </Tooltip>

                            {/* Detail */}
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="outline"
                                  onClick={() => setSelectedAsset(a)}
                                  className="h-7 w-7"
                                >
                                  <Info size={14} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Lihat Detail</TooltipContent>
                            </Tooltip>

                            {/* Log */}
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="outline"
                                  onClick={() => setLogId(a.id)}
                                  className="h-7 w-7"
                                >
                                  <FileText size={14} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Lihat Log</TooltipContent>
                            </Tooltip>

                            {/* Edit */}
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="outline"
                                  onClick={() => {
                                    setEditId(a.id)
                                    setOpenForm(true)
                                  }}
                                  className="h-7 w-7"
                                >
                                  <Edit3 size={14} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Aset</TooltipContent>
                            </Tooltip>

                            {/* Delete */}
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  size="icon"
                                  variant="destructive"
                                  onClick={() => {
                                    setDeleteItem(a)
                                    setConfirmOpen(true)
                                  }}
                                  className="h-7 w-7"
                                >
                                  <Trash2 size={14} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Aset</TooltipContent>
                            </Tooltip>
                          </div>
                        </TooltipProvider>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>


          {/* Pagination */}
          <div className="flex items-center justify-between mt-4 text-sm">
            <p className="text-muted-foreground">{PageInfo}</p>
            <div className="flex items-center gap-2">
              <Button size="icon" variant="outline" disabled={page <= 1} onClick={() => setPage(p => Math.max(1, p - 1))}>
                <ChevronLeft size={16} />
              </Button>
              <span>Halaman {page} dari {totalPages}</span>
              <Button size="icon" variant="outline" disabled={page >= totalPages} onClick={() => setPage(p => Math.min(totalPages, p + 1))}>
                <ChevronRight size={16} />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Modals */}
      <AssetFormModal open={openForm} onOpenChange={setOpenForm} assetId={editId ? String(editId) : null}
        onSaved={loadAssets} assetTypes={assetTypes} departments={departments} locations={locations} />
      <AssetDetailsModal asset={selectedAsset} isOpen={!!selectedAsset} onClose={() => setSelectedAsset(null)} />
      <AssignAssetModal assetId={assignId ?? 0} isOpen={!!assignId} onClose={() => setAssignId(null)} onAssigned={loadAssets} />
      <ReturnAssetModal assetId={returnId ?? 0} isOpen={!!returnId} onClose={() => setReturnId(null)} onReturned={loadAssets} />
      <MaintenanceLogModal assetId={logId ?? 0} isOpen={!!logId} onClose={() => setLogId(null)} />

      {/* Konfirmasi Hapus */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus Aset</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus aset "<strong>{deleteItem?.name}</strong>"?
              <br />Tindakan ini tidak dapat dibatalkan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
