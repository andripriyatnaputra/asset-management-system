import { useEffect, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import apiClient from "@/services/api"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select"

export interface Location {
  id: number
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
}

export default function LocationFormModal({ isOpen, onClose, onSuccess, location }: LocationFormModalProps) {
  const [site, setSite] = useState("")
  const [building, setBuilding] = useState("")
  const [room, setRoom] = useState("")
  const [description, setDescription] = useState("")
  const [status, setStatus] = useState("active")
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setSite(location?.site ?? "")
      setBuilding(location?.building ?? "")
      setRoom(location?.room ?? "")
      setDescription(location?.description ?? "")
      setStatus(location?.status ?? "active")
    }
  }, [isOpen, location])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const payload = { site, building, room, description, status }
      if (location) {
        // update
        await apiClient.put(`/locations/${location.id}`, payload)
        toast.success("Lokasi berhasil diperbarui")
      } else {
        // create
        await apiClient.post(`/locations`, payload)
        toast.success("Lokasi berhasil ditambahkan")
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
          <DialogTitle>{location ? "Edit Lokasi" : "Tambah Lokasi"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="site">Site</Label>
            <Input id="site" value={site} onChange={(e) => setSite(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="building">Building</Label>
            <Input id="building" value={building} onChange={(e) => setBuilding(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="room">Room</Label>
            <Input id="room" value={room} onChange={(e) => setRoom(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Input id="description" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="status">Status</Label>
            <Select value={status} onValueChange={(v) => setStatus(v)}>
            <SelectTrigger><SelectValue placeholder="Pilih status" /></SelectTrigger>
            <SelectContent>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>Batal</Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}