import { useEffect, useMemo, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import apiClient from "@/services/api"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select"

export interface Location {
  id: number
  parent_id?: number | null
  parent_name?: string | null
  site: string
  building?: string | null
  room?: string | null
  description?: string | null
  status?: string
}

interface LocationFormModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  location?: Location | null

  /** default value untuk parent selector (misal "__ROOT__") */
  defaultParentValue?: string

  /** kalau true, parent selector akan dikunci (untuk mode "Tambah Parent") */
  lockParent?: boolean
}

const FALLBACK_ROOT = "__ROOT__"

export default function LocationFormModal({
  isOpen,
  onClose,
  onSuccess,
  location,
  defaultParentValue,
  lockParent
}: LocationFormModalProps) {
  const ROOT_VALUE = defaultParentValue ?? FALLBACK_ROOT

  const [site, setSite] = useState("")
  const [building, setBuilding] = useState("")
  const [room, setRoom] = useState("")
  const [description, setDescription] = useState("")
  const [status, setStatus] = useState("active")

  // shadcn Select: value tidak boleh empty string
  const [parentValue, setParentValue] = useState<string>(ROOT_VALUE)
  const [parents, setParents] = useState<Location[]>([])
  const [saving, setSaving] = useState(false)

  const isParentMode = !!lockParent

  const loadParents = async () => {
    try {
      const res = await apiClient.get("/locations")
      const raw = res.data?.data ?? res.data
      const all: Location[] = Array.isArray(raw) ? raw : []

      // kandidat parent = root active (parent_id null)
      const roots = all.filter((l) => !l.parent_id && l.status === "active")
      setParents(roots)
    } catch {
      setParents([])
    }
  }

  // init form tiap modal dibuka
  useEffect(() => {
    if (!isOpen) return

    setSite(location?.site ?? "")
    setBuilding(location?.building ?? "")
    setRoom(location?.room ?? "")
    setDescription(location?.description ?? "")
    setStatus(location?.status ?? "active")

    // mapping parent_id -> select value
    const initialParent =
      location?.parent_id ? String(location.parent_id) : ROOT_VALUE
    setParentValue(initialParent)

    loadParents()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, location])

  // kalau mode parent: paksa root + bersihkan detail agar tidak “ketuker”
  useEffect(() => {
    if (!isOpen) return
    if (isParentMode) {
      setParentValue(ROOT_VALUE)
      // optional, biar parent benar-benar top level
      setBuilding("")
      setRoom("")
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, isParentMode])

  // options parent (hindari self-parent)
  const parentOptions = useMemo(() => {
    return parents.filter((p) => !location || p.id !== location.id)
  }, [parents, location])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)

    try {
      const payload = {
        parent_id: parentValue === ROOT_VALUE ? null : Number(parentValue),
        site: site.trim(),
        building: building.trim() || null,
        room: room.trim() || null,
        description: description.trim() || null,
        status
      }

      if (!payload.site) {
        toast.error("Site wajib diisi.")
        setSaving(false)
        return
      }

      if (location) {
        await apiClient.put(`/locations/${location.id}`, payload)
        toast.success("Lokasi berhasil diperbarui")
      } else {
        await apiClient.post(`/locations`, payload)
        toast.success(isParentMode ? "Parent lokasi berhasil ditambahkan" : "Lokasi berhasil ditambahkan")
      }

      onSuccess()
      onClose()
    } catch (err: any) {
      toast.error(err?.response?.data?.error || "Gagal menyimpan lokasi")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {location ? "Edit Lokasi" : isParentMode ? "Tambah Parent Lokasi" : "Tambah Lokasi"}
          </DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Parent selector */}
          <div className="space-y-2">
            <Label>Parent Location</Label>
            <Select
              value={parentValue}
              onValueChange={(v) => setParentValue(v)}
              disabled={isParentMode}
            >
              <SelectTrigger>
                <SelectValue placeholder="Pilih Parent (opsional)" />
              </SelectTrigger>
              <SelectContent>
                {/* Root option MUST NOT be empty string */}
                <SelectItem value={ROOT_VALUE}>(Root / Tidak ada parent)</SelectItem>

                {parentOptions.map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>
                    {p.site}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {isParentMode && (
              <div className="text-xs text-muted-foreground">
                Mode Parent: lokasi ini akan dibuat sebagai root (tanpa parent).
              </div>
            )}
          </div>

          {/* Site */}
          <div className="space-y-2">
            <Label htmlFor="site">Site</Label>
            <Input
              id="site"
              value={site}
              onChange={(e) => setSite(e.target.value)}
              required
            />
          </div>

          {/* Building */}
          <div className="space-y-2">
            <Label htmlFor="building">Building</Label>
            <Input
              id="building"
              value={building}
              onChange={(e) => setBuilding(e.target.value)}
              disabled={isParentMode}
              placeholder={isParentMode ? "Dinonaktifkan untuk Parent" : ""}
            />
          </div>

          {/* Room */}
          <div className="space-y-2">
            <Label htmlFor="room">Room</Label>
            <Input
              id="room"
              value={room}
              onChange={(e) => setRoom(e.target.value)}
              disabled={isParentMode}
              placeholder={isParentMode ? "Dinonaktifkan untuk Parent" : ""}
            />
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Input
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          {/* Status */}
          <div className="space-y-2">
            <Label>Status</Label>
            <Select value={status} onValueChange={(v) => setStatus(v)}>
              <SelectTrigger>
                <SelectValue placeholder="Pilih status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Batal
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
