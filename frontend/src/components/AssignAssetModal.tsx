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
  const [loadingEmployees, setLoadingEmployees] = useState(false)

  const normalizeList = (res: any) =>
    Array.isArray(res?.data?.data) ? res.data.data :
    Array.isArray(res?.data) ? res.data :
    []

  // ✅ Load ALL employees (respect pagination)
  useEffect(() => {
    if (!isOpen) return

    let cancelled = false

    const fetchAllEmployees = async () => {
      setLoadingEmployees(true)
      try {
        const all: Employee[] = []

        // fetch page 1
        const first = await apiClient.get("/employees", {
          params: { page: 1, limit: 10 },
        })

        const firstRows = normalizeList(first)
        all.push(...firstRows)

        const totalPages: number = first?.data?.pagination?.total_pages ?? 1

        // fetch remaining pages
        for (let page = 2; page <= totalPages; page++) {
          const res = await apiClient.get("/employees", {
            params: { page, limit: 10 },
          })
          all.push(...normalizeList(res))
        }

        if (!cancelled) setEmployees(all)
      } catch {
        if (!cancelled) {
          setEmployees([])
          toast.error("Gagal memuat daftar karyawan.")
        }
      } finally {
        if (!cancelled) setLoadingEmployees(false)
      }
    }

    fetchAllEmployees()

    return () => {
      cancelled = true
    }
  }, [isOpen])

  const sortedEmployees = useMemo(() => {
    const list = Array.isArray(employees) ? employees : []
    return [...list].sort((a, b) => {
      const anik = (a.employee_nik || "").toString()
      const bnik = (b.employee_nik || "").toString()
      if (anik && bnik && anik !== bnik) return anik.localeCompare(bnik)
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
              <SelectValue placeholder={loadingEmployees ? "Memuat karyawan..." : "Pilih karyawan"}>
                {selectedEmployeeLabel || undefined}
              </SelectValue>
            </SelectTrigger>

            {/* ✅ scroll */}
            <SelectContent className="max-h-72 overflow-y-auto">
              {loadingEmployees ? (
                <div className="text-muted-foreground text-sm px-2 py-2">
                  Memuat data karyawan…
                </div>
              ) : sortedEmployees.length === 0 ? (
                <div className="text-muted-foreground text-sm px-2 py-2">
                  Tidak ada data karyawan.
                </div>
              ) : (
                <Command shouldFilter>
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
                          {/* tampilkan label */}
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
          <Button onClick={handleAssign} disabled={submitting || loadingEmployees}>
            {submitting ? "Meng-assign…" : "Assign"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
