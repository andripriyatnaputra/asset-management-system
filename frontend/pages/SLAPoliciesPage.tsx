import { useEffect, useState, useMemo } from "react"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction
} from "@/components/ui/alert-dialog"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter
} from "@/components/ui/dialog"
import { toast } from "sonner"
import { Edit, Trash2, Search, ArrowUpDown, ChevronLeft, ChevronRight } from "lucide-react"
import apiClient from "@/services/api"

type SLAPolicy = {
  id: number
  name: string
  category_code?: string
  service_code?: string
  impact: string
  urgency: string
  resulting_priority: string
  response_minutes: number
  resolve_minutes: number
  is_active: boolean
  compliance_score?: number
}

type SLADetail = SLAPolicy & {
  legacy_compliance_score?: number
  created_by_name?: string
  updated_by_name?: string
  created_at?: string
  updated_at?: string
}

const impactOptions = ["Low", "Medium", "High"]
const urgencyOptions = ["Low", "Medium", "High"]
const priorityOptions = ["Low", "Medium", "High", "Critical"]
const pageSizeOptions = [10, 25, 50]

export default function SLAPoliciesPage() {
  const [list, setList] = useState<SLAPolicy[]>([])
  const [loading, setLoading] = useState(true)
  const [form, setForm] = useState<Partial<SLAPolicy>>({ is_active: true })
  const [editId, setEditId] = useState<number | null>(null)
  const [search, setSearch] = useState("")
  const [sortAsc, setSortAsc] = useState(true)
  const [pageSize, setPageSize] = useState(10)
  const [currentPage, setCurrentPage] = useState(1)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleteItem, setDeleteItem] = useState<SLAPolicy | null>(null)

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<SLADetail | null>(null)
  const [duplicating, setDuplicating] = useState(false)

  // =============================
  // SAFE FETCH WITH FALLBACK
  // =============================
const fetchPolicies = async () => {
  setLoading(true);

  try {
    const res = await apiClient.get("/sla-policies");

    console.log("RAW SLA RESPONSE:", res.data);

    // 1) BACA KEY YANG BENAR DARI BACKEND
    let items =
      res.data?.data ||
      res.data?.sla_policies ||     // backend kamu pakai key ini
      [];

    // 2) SAFETY GUARD
    if (!Array.isArray(items)) {
      console.warn("SLA response bukan array:", items);
      items = [];
    }

    // 3) NORMALISASI FIELD AGAR UI AMAN
    items = items.map((p: any) => ({
      ...p,
      name: p.name ?? "",
      category_code: p.category_code ?? "",
      service_code: p.service_code ?? "",
      impact: typeof p.impact === "string"
        ? p.impact.charAt(0).toUpperCase() + p.impact.slice(1).toLowerCase()
        : "Low",
      urgency: typeof p.urgency === "string"
        ? p.urgency.charAt(0).toUpperCase() + p.urgency.slice(1).toLowerCase()
        : "Low",
      resulting_priority: typeof p.resulting_priority === "string"
        ? p.resulting_priority.charAt(0).toUpperCase() + p.resulting_priority.slice(1).toLowerCase()
        : "Low",
      response_minutes: Number(p.response_minutes) || 0,
      resolve_minutes: Number(p.resolve_minutes) || 0,
      is_active: Boolean(p.is_active),
      compliance_score: Number(p.compliance_score ?? 0),
    }));

    // 4) SET DATA
    setList(items);

    // 5) RESET PAGE AGAR TIDAK KOSONG
    setCurrentPage(1);

  } catch (err) {
    console.error("ERROR FETCH SLA POLICIES:", err);
    toast.error("Gagal memuat SLA policies.");
    setList([]);
  } finally {
    setLoading(false);
  }
};


  useEffect(() => { fetchPolicies() }, [])

  // =============================
  // FILTERING (SAFE)
  // =============================
  const filteredList = useMemo(() => {
    const safeList = Array.isArray(list) ? list : []

    return safeList
      .filter(p => (p?.name ?? "").toLowerCase().includes(search.toLowerCase()))
      .sort((a, b) =>
        sortAsc
          ? (a?.name ?? "").localeCompare(b?.name ?? "")
          : (b?.name ?? "").localeCompare(a?.name ?? "")
      )
  }, [list, search, sortAsc])

  // =============================
  // PAGINATION
  // =============================
  const totalPages = Math.max(1, Math.ceil(filteredList.length / pageSize))

  const paginatedList = useMemo(() => {
    const start = (currentPage - 1) * pageSize
    return filteredList.slice(start, start + pageSize)
  }, [filteredList, currentPage, pageSize])

  // =============================
  // CRUD Handlers
  // =============================
  const handleSave = async () => {
    if (!form.name || !form.impact || !form.urgency || !form.resulting_priority) {
      toast.error("Lengkapi semua field wajib.")
      return
    }

    try {
      if (editId) {
        await apiClient.put(`/sla-policies/${editId}`, form)
        toast.success("Policy diperbarui.")
      } else {
        await apiClient.post("/sla-policies", form)
        toast.success("Policy ditambahkan.")
      }

      setForm({ is_active: true })
      setEditId(null)
      fetchPolicies()
    } catch {
      toast.error("Gagal menyimpan policy.")
    }
  }

  const handleEdit = (p: SLAPolicy) => {
    setForm(p)
    setEditId(p.id)
  }

  const handleDelete = async () => {
    if (!deleteItem) return
    try {
      await apiClient.delete(`/sla-policies/${deleteItem.id}`)
      toast.success(`Policy "${deleteItem.name}" dihapus.`)
      setConfirmOpen(false)
      fetchPolicies()
    } catch {
      toast.error("Gagal menghapus policy.")
    }
  }

  const showDetail = async (id: number) => {
    try {
      const res = await apiClient.get(`/sla-policies/${id}`)
      setDetail(res.data.data)
      setDetailOpen(true)
    } catch {
      toast.error("Gagal memuat detail SLA.")
    }
  }

  const duplicatePolicy = async () => {
    if (!detail) return
    setDuplicating(true)

    try {
      const payload = {
        name: `${detail.name} (Copy)`,
        category_code: detail.category_code,
        service_code: detail.service_code,
        impact: detail.impact,
        urgency: detail.urgency,
        resulting_priority: detail.resulting_priority,
        response_minutes: detail.response_minutes,
        resolve_minutes: detail.resolve_minutes,
        is_active: detail.is_active,
      }

      await apiClient.post("/sla-policies", payload)
      toast.success("Policy berhasil diduplikasi!")
      setDetailOpen(false)
      fetchPolicies()
    } catch {
      toast.error("Gagal menduplikasi policy.")
    } finally {
      setDuplicating(false)
    }
  }

  const getPriorityBadge = (priority: string) => {
    const colors: Record<string, string> = {
      Critical: "bg-red-500/20 text-red-600 border-red-400",
      High: "bg-orange-500/20 text-orange-600 border-orange-400",
      Medium: "bg-yellow-500/20 text-yellow-700 border-yellow-400",
      Low: "bg-green-500/20 text-green-600 border-green-400",
    }
    return (
      <span className={`px-2 py-1 rounded-full text-xs font-medium border ${colors[priority] || ""}`}>
        {priority}
      </span>
    )
  }

  const getScoreBar = (score?: number) => {
    const val = score ?? 100
    let color = "bg-green-500"
    if (val < 70) color = "bg-red-500"
    else if (val < 90) color = "bg-yellow-500"

    return (
      <div className="w-[80px]">
        <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
          <div className={`h-2 ${color} transition-all`} style={{ width: `${val}%` }}></div>
        </div>
        <span className="text-xs text-muted-foreground">{val}%</span>
      </div>
    )
  }

  // ============================================================
  // ========================== RETURN ===========================
  // ============================================================

  return (
    <div className="p-6 space-y-6">

      {/* HEADER + SEARCH */}
      <Card>
        <CardHeader className="flex flex-col md:flex-row md:items-center md:justify-between">
          <CardTitle className="text-lg font-semibold flex items-center gap-2">
            📈 SLA Policies Management
          </CardTitle>
          <div className="flex items-center gap-2 mt-3 md:mt-0">
            <Search size={18} />
            <Input
              placeholder="Cari nama SLA..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-[220px]"
            />
            <Button variant="ghost" size="sm" onClick={() => setSortAsc(!sortAsc)}>
              <ArrowUpDown size={18} />
            </Button>
          </div>
        </CardHeader>

        <CardContent>

          {/* FORM INPUT */}
          <div className="grid md:grid-cols-3 gap-4 mb-6">
            <Input placeholder="Nama SLA"
              value={form.name || ""}
              onChange={e => setForm({ ...form, name: e.target.value })}
            />

            <Select value={form.category_code || ""} onValueChange={v => setForm({ ...form, category_code: v })}>
              <SelectTrigger><SelectValue placeholder="Category" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="INCIDENT">Incident</SelectItem>
                <SelectItem value="REQUEST">Service Request</SelectItem>
              </SelectContent>
            </Select>

            <Select value={form.service_code || ""} onValueChange={v => setForm({ ...form, service_code: v })}>
              <SelectTrigger><SelectValue placeholder="Service" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="ENDPOINT">End-user Device</SelectItem>
                <SelectItem value="EMAIL">Email</SelectItem>
                <SelectItem value="NETWORK">Network</SelectItem>
              </SelectContent>
            </Select>

            <Select value={form.impact} onValueChange={v => setForm({ ...form, impact: v })}>
              <SelectTrigger><SelectValue placeholder="Impact" /></SelectTrigger>
              <SelectContent>{impactOptions.map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
            </Select>

            <Select value={form.urgency} onValueChange={v => setForm({ ...form, urgency: v })}>
              <SelectTrigger><SelectValue placeholder="Urgency" /></SelectTrigger>
              <SelectContent>{urgencyOptions.map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
            </Select>

            <Select value={form.resulting_priority} onValueChange={v => setForm({ ...form, resulting_priority: v })}>
              <SelectTrigger><SelectValue placeholder="Priority" /></SelectTrigger>
              <SelectContent>{priorityOptions.map(v => <SelectItem key={v} value={v}>{v}</SelectItem>)}</SelectContent>
            </Select>

            <Input placeholder="Response (menit)" type="number"
              value={form.response_minutes || ""}
              onChange={e => setForm({ ...form, response_minutes: +e.target.value })}
            />

            <Input placeholder="Resolve (menit)" type="number"
              value={form.resolve_minutes || ""}
              onChange={e => setForm({ ...form, resolve_minutes: +e.target.value })}
            />

            <div className="flex items-center space-x-2">
              <Switch checked={!!form.is_active} onCheckedChange={v => setForm({ ...form, is_active: v })} />
              <Label>Aktif</Label>
            </div>

            <Button onClick={handleSave} className="col-span-3">
              {editId ? "Perbarui" : "Tambah"} Policy
            </Button>

            {editId && (
              <Button variant="outline" className="col-span-3"
                onClick={() => { setForm({ is_active: true }); setEditId(null) }}>
                Batal Edit
              </Button>
            )}
          </div>

          {/* TABLE LIST */}
          {loading ? (
            <p>Memuat...</p>
          ) : (
            <div className="overflow-x-auto rounded-lg border">
              <table className="w-full text-sm">
                <thead className="bg-muted text-left">
                  <tr>
                    <th className="p-2">Nama</th>
                    <th>Impact</th>
                    <th>Urgency</th>
                    <th>Priority</th>
                    <th>Response</th>
                    <th>Resolve</th>
                    <th>Score</th>
                    <th>Status</th>
                    <th className="text-center">Aksi</th>
                  </tr>
                </thead>

                <tbody>
                  {paginatedList.map(p => (
                    <tr key={p.id} className="border-t hover:bg-accent/40 transition-all">
                      <td className="p-2">
                        <button onClick={() => showDetail(p.id)}
                          className="text-blue-600 hover:underline text-left">
                          {p.name}
                        </button>
                      </td>
                      <td>{p.impact}</td>
                      <td>{p.urgency}</td>
                      <td>{getPriorityBadge(p.resulting_priority)}</td>
                      <td>{p.response_minutes}m</td>
                      <td>{p.resolve_minutes}m</td>
                      <td>{getScoreBar(p.compliance_score)}</td>
                      <td>{p.is_active ? "✅" : "❌"}</td>

                      <td className="text-center">
                        <TooltipProvider>
                          <div className="flex justify-center gap-2">

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="outline" onClick={() => handleEdit(p)}>
                                  <Edit size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Edit Policy</TooltipContent>
                            </Tooltip>

                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button size="icon" variant="destructive"
                                  onClick={() => { setDeleteItem(p); setConfirmOpen(true) }}>
                                  <Trash2 size={16} />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Hapus Policy</TooltipContent>
                            </Tooltip>

                          </div>
                        </TooltipProvider>
                      </td>
                    </tr>
                  ))}
                </tbody>

              </table>
            </div>
          )}

          {/* PAGINATION */}
          <div className="flex flex-col md:flex-row items-center justify-between mt-4 text-sm gap-3">

            <div className="flex items-center gap-2">
              <Label>Baris per halaman:</Label>
              <Select
                value={pageSize.toString()}
                onValueChange={(v) => { setPageSize(Number(v)); setCurrentPage(1) }}
              >
                <SelectTrigger className="w-[80px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  {pageSizeOptions.map(size => (
                    <SelectItem key={size} value={size.toString()}>{size}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center gap-4">
              <span>Halaman {currentPage} dari {totalPages}</span>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="icon"
                  disabled={currentPage === 1}
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                >
                  <ChevronLeft size={16} />
                </Button>

                <Button variant="outline" size="icon"
                  disabled={currentPage === totalPages}
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                >
                  <ChevronRight size={16} />
                </Button>
              </div>
            </div>

          </div>
        </CardContent>
      </Card>

      {/* DIALOG KONFIRMASI HAPUS */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus SLA Policy</AlertDialogTitle>
            <AlertDialogDescription>
              Anda yakin ingin menghapus policy "<strong>{deleteItem?.name}</strong>"?
              <br />Tindakan ini tidak dapat dibatalkan.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* MODAL DETAIL SLA */}
      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Detail SLA Policy</DialogTitle>
          </DialogHeader>

          {detail ? (
            <div className="space-y-2 text-sm">
              <div>
                <p className="font-semibold text-base">{detail.name}</p>
                <p className="text-xs text-muted-foreground">
                  {detail.is_active ? "Aktif ✅" : "Nonaktif ❌"}
                </p>
              </div>

              <hr />

              <div className="grid grid-cols-2 gap-x-3 gap-y-2">
                <div><strong>Category:</strong> {detail.category_code || "-"}</div>
                <div><strong>Service:</strong> {detail.service_code || "-"}</div>
                <div><strong>Impact:</strong> {detail.impact}</div>
                <div><strong>Urgency:</strong> {detail.urgency}</div>
                <div><strong>Priority:</strong> {detail.resulting_priority}</div>
                <div><strong>Response:</strong> {detail.response_minutes} menit</div>
                <div><strong>Resolve:</strong> {detail.resolve_minutes} menit</div>

                <div>
                  <strong>Compliance:</strong>{" "}
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    (detail.compliance_score ?? 100) < 70
                      ? "bg-red-100 text-red-700"
                      : (detail.compliance_score ?? 100) < 90
                      ? "bg-yellow-100 text-yellow-700"
                      : "bg-green-100 text-green-700"
                  }`}>
                    {detail.compliance_score?.toFixed(2)}%
                  </span>
                </div>

                <div><strong>Legacy:</strong> {detail.legacy_compliance_score?.toFixed(2)}%</div>
              </div>

              <hr />

              <div className="text-xs space-y-1 text-muted-foreground">
                <p>Dibuat oleh: <strong>{detail.created_by_name || "-"}</strong></p>
                <p>Diperbarui oleh: <strong>{detail.updated_by_name || "-"}</strong></p>
                <p>Dibuat pada: {detail.created_at ? new Date(detail.created_at).toLocaleString("id-ID") : "-"}</p>
                <p>Diperbarui pada: {detail.updated_at ? new Date(detail.updated_at).toLocaleString("id-ID") : "-"}</p>
              </div>
            </div>
          ) : (
            <p>Memuat detail SLA...</p>
          )}

          <DialogFooter className="flex justify-between mt-3">
            <Button variant="outline" onClick={() => setDetailOpen(false)}>
              Tutup
            </Button>
            <Button onClick={duplicatePolicy} disabled={duplicating}>
              {duplicating ? "Menduplikasi..." : "Duplicate Policy"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

    </div>
  )
}
