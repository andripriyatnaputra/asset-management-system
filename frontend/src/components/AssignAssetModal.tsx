import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import type { Employee } from "@/types"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"

interface AssignAssetModalProps {
  assetId: number
  isOpen: boolean
  onClose: () => void
  onAssigned: () => void
}

export default function AssignAssetModal({
  assetId,
  isOpen,
  onClose,
  onAssigned,
}: AssignAssetModalProps) {
  const [employees, setEmployees] = useState<Employee[]>([])
  const [employeeId, setEmployeeId] = useState<string>("")
  const [submitting, setSubmitting] = useState(false)

  // 🔹 Load employees setiap modal dibuka
  useEffect(() => {
    if (!isOpen) return
    apiClient
      .get("/employees")
      .then((res) => {
        const data = res.data?.data ?? res.data ?? []
        setEmployees(Array.isArray(data) ? data : [])
      })
      .catch(() => {
        toast.error("Gagal memuat daftar karyawan.")
        setEmployees([])
      })
  }, [isOpen])

  // 🔹 Handler submit assign
  const handleAssign = async () => {
    if (submitting) return
    if (!employeeId) {
      toast.error("Pilih karyawan terlebih dahulu.")
      return
    }

    setSubmitting(true)
    try {
      const promise = apiClient.post(`/assets/${assetId}/assign`, {
        employee_id: Number(employeeId),
      })

      // Tampilkan toast berbasis promise
      toast.promise(promise, {
        loading: "Meng-assign aset...",
        success: "Aset berhasil diassign.",
        error: (err) =>
          err?.response?.data?.error ||
          "Gagal meng-assign aset. Periksa koneksi atau status aset.",
      })

      // Tunggu hasil Axios-nya
      const res = await promise
      if (res.status === 200) {
        onAssigned() // trigger reload di parent
        onClose()    // tutup modal
      }
    } catch (err) {
      console.error("[AssignAssetModal] Error:", err)
    } finally {
      setSubmitting(false)
    }
  }



  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Assign Asset</DialogTitle>
        </DialogHeader>

        <div className="space-y-3 py-2">
          <Label>Pilih Karyawan</Label>
          <Select value={employeeId} onValueChange={setEmployeeId}>
            <SelectTrigger>
              <SelectValue placeholder="Pilih karyawan" />
            </SelectTrigger>
            <SelectContent>
              {employees.length === 0 ? (
                <p className="text-muted-foreground text-sm px-2 py-1">
                  Tidak ada data karyawan.
                </p>
              ) : (
                employees.map((e) => (
                  <SelectItem key={e.id} value={String(e.id)}>
                    {e.employee_nik ? `[${e.employee_nik}] ` : ""}
                    {e.name}
                  </SelectItem>
                ))
              )}
            </SelectContent>
          </Select>
        </div>

        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleAssign} disabled={submitting}>
            {submitting ? "Meng-assign…" : "Assign"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
