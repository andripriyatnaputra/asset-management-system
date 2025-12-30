// File: src/components/InstallSoftwareModal.tsx
import { useState, useEffect } from "react"
import apiClient from "@/services/api"
import { toast } from "sonner"
import type { SoftwareLicense } from "@/types"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface InstallSoftwareModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  assetId: number | null
}

export default function InstallSoftwareModal({
  isOpen,
  onClose,
  onSuccess,
  assetId,
}: InstallSoftwareModalProps) {
  const [licenses, setLicenses] = useState<SoftwareLicense[]>([])
  const [selectedLicenseId, setSelectedLicenseId] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!isOpen) return

    setIsLoading(true)
    apiClient
      .get("/licenses")
      .then((res) => {
        const data = res.data?.data ?? res.data ?? []
        setLicenses(Array.isArray(data) ? data : [])
      })
      .catch(() => {
        toast.error("Gagal memuat daftar lisensi.")
        setLicenses([])
      })
      .finally(() => setIsLoading(false))
  }, [isOpen])

  const handleSubmit = async () => {
    if (submitting) return

    if (!assetId || !selectedLicenseId) {
      toast.error("Silakan pilih lisensi terlebih dahulu.")
      return
    }

    setSubmitting(true)

    // ✅ endpoint diseragamkan: /assets/:id/software
    const promise = apiClient.post(`/assets/${assetId}/software`, {
      license_id: Number(selectedLicenseId),
    })

    toast.promise(promise, {
      loading: "Menginstal software...",
      success: () => {
        onSuccess()
        return "Software berhasil diinstal!"
      },
      error: (err: any) => {
        const msg =
          err?.response?.data?.error ||
          (err?.response?.status === 409
            ? "Software ini sudah terpasang pada aset tersebut."
            : "Gagal menginstal software. Pastikan lisensi masih tersedia.")
        return msg
      },
    })

    try {
      await promise
    } catch (err) {
      console.error("Install software error:", err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Install Software Baru</DialogTitle>
        </DialogHeader>
        <div className="py-4">
          <p className="mb-2 text-sm text-muted-foreground">
            Pilih lisensi software yang akan diinstal pada aset ini.
          </p>
          <Select
            value={selectedLicenseId}
            onValueChange={setSelectedLicenseId}
          >
            <SelectTrigger>
              <SelectValue
                placeholder={isLoading ? "Memuat lisensi..." : "Pilih lisensi..."}
              />
            </SelectTrigger>
            <SelectContent>
              {!isLoading &&
                licenses &&
                licenses.map((license) => (
                  <SelectItem key={license.id} value={license.id.toString()}>
                    {license.name} ({license.total_seats} seats)
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {submitting ? "Menginstal…" : "Install"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
