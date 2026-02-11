import { useEffect, useMemo, useState } from "react"
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
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from "@/components/ui/command"

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

  // load employees setiap modal dibuka
  useEffect(() => {
    if (!isOpen) return

    apiClient
      .get("/employees")
      .then((res) => {
        const data = Array.isArray(res?.data?.data)
          ? res.data.data
          : Array.isArray(res?.data)
          ? res.data
          : []
        setEmployees(data)
      })
      .catch(() => {
        toast.error("Gagal memuat daftar karyawan.")
        setEmployees([])
      })
  }, [isOpen])

  // optional: sort biar lebih enak dicari (NIK dulu, lalu name)
  const sortedEmployees = useMemo(() => {
    const list = Array.isArray(employees) ? employees : []
    return [...list].sort((a, b) => {
      const an = (a.employee_nik || "").toString()
      const bn = (b.employee_nik || "").toString()
      if (an && bn && an !== bn) return an.localeCompare(bn)
      return (a.name || "").localeCompare(b.name || "")
    })
  }, [employees])

  const selectedEmployeeLabel = useMemo(() => {
    if (!employeeId) return ""
    const e = sortedEmployees.find((x) => String(x.id) === employeeId)
    if (!e) return ""
    return `${e.employee_nik ? `[${e.employee_nik}] ` : ""}${e.name}`
  }, [employeeId, sortedEmployees])

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

      toast.promise(promise, {
        loading: "Meng-assign aset...",
        success: "Aset berhasil diassign.",
        error: (err) =>
          err?.response?.data?.error ||
          "Gagal meng-assign aset. Periksa koneksi atau status aset.",
      })

      const res = await promise
      if (res.status === 200) {
        onAssigned()
        onClose()
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
              <SelectValue placeholder="Pilih karyawan">
                {selectedEmployeeLabel || undefined}
              </SelectValue>
            </SelectTrigger>

            {/* ✅ max-h + overflow biar bisa scroll */}
            <SelectContent className="max-h-72 overflow-y-auto">
              {sortedEmployees.length === 0 ? (
                <div className="text-muted-foreground text-sm px-2 py-2">
                  Tidak ada data karyawan.
                </div>
              ) : (
                // ✅ Search pakai Command (lebih enak untuk list panjang)
                <Command shouldFilter={true}>
                  <CommandInput placeholder="Cari nama / NIK..." />
                  <CommandEmpty>Tidak ditemukan.</CommandEmpty>

                  <CommandGroup>
                    {sortedEmployees.map((e) => {
                      const value = String(e.id)
                      const label = `${e.employee_nik ? `[${e.employee_nik}] ` : ""}${e.name}`

                      return (
                        <CommandItem
                          key={e.id}
                          value={`${e.name ?? ""} ${e.employee_nik ?? ""} ${value}`}
                          onSelect={() => setEmployeeId(value)}
                        >
                          {/* SelectItem butuh jadi wrapper item selectable; kita render sebagai label saja */}
                          <SelectItem value={value}>{label}</SelectItem>
                        </CommandItem>
                      )
                    })}
                  </CommandGroup>
                </Command>
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
