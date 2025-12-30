// File: src/components/AssetDetailsModal.tsx
import { useEffect, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { FileText } from "lucide-react"
import { toast } from "sonner"
import apiClient from "@/services/api"

import type { Asset } from "@/types"
import InstallSoftwareModal from "@/components/InstallSoftwareModal"
import { AssignmentHistory } from "./AssignmentHistory"
import { AssetAuditHistory } from "./AssetAuditHistory"

interface AssetDetailsModalProps {
  asset: Asset | null
  isOpen: boolean
  onClose: () => void
}

interface InstalledSoftware {
  installation_id: number
  license_name: string
  license_key?: string | null
  installation_date: string
}

export default function AssetDetailsModal({
  asset,
  isOpen,
  onClose,
}: AssetDetailsModalProps) {
  const [detail, setDetail] = useState<Asset | null>(null)
  const [softwareList, setSoftwareList] = useState<InstalledSoftware[]>([])
  const [verifying, setVerifying] = useState(false)
  const [openInstallModal, setOpenInstallModal] = useState(false)

  const data = detail || asset

  const formatIDR = (val?: number | null) =>
    val != null ? `Rp ${val.toLocaleString("id-ID")}` : "-"
  const formatDate = (date?: string | null) =>
    date ? new Date(date).toLocaleDateString("id-ID") : "-"

  const complianceColor =
    data?.compliance_flag === true
      ? "bg-green-100 text-green-700"
      : data?.compliance_flag === false
      ? "bg-red-100 text-red-700"
      : "bg-yellow-100 text-yellow-700"

  // ---------- Loader: detail asset ----------
  const reloadDetail = async () => {
    if (!asset?.id) return
    try {
      const res = await apiClient.get(`/assets/${asset.id}`)
      setDetail(res.data?.asset ?? null)
    } catch {
      setDetail(null)
    }
  }

  // ---------- Loader: software list ----------
  const loadSoftware = async () => {
    if (!asset?.id) return
    try {
      const r = await apiClient.get(`/assets/${asset.id}/software`)
      const raw = r.data?.data ?? r.data ?? []
      setSoftwareList(Array.isArray(raw) ? raw : [])
    } catch (err) {
      console.error("Failed to load software list:", err)
      setSoftwareList([])
    }
  }

  // ---------- Handlers ----------
  const handleUninstall = async (installationId: number) => {
    if (!asset?.id) return
    if (!confirm("Hapus software ini dari aset?")) return
    try {
      await apiClient.delete(`/assets/${asset.id}/software/${installationId}`)
      toast.success("Software dihapus dari aset.")
      loadSoftware()
    } catch (err: any) {
      console.error("Uninstall error:", err)
      toast.error(
        err?.response?.data?.error || "Gagal menghapus software dari aset."
      )
    }
  }

  const handleVerifyCompliance = async () => {
    if (!asset?.id) return
    setVerifying(true)
    try {
      await apiClient.post(`/assets/${asset.id}/verify-compliance`)
      toast.success("Verifikasi compliance berhasil dijalankan")
      await reloadDetail()
    } catch {
      toast.error("Gagal memverifikasi compliance")
    } finally {
      setVerifying(false)
    }
  }

  const handleDownloadReport = async (assetId: number) => {
    try {
      const res = await apiClient.get(`/assets/${assetId}/report`, {
        responseType: "blob",
      })
      const blob = new Blob([res.data], { type: "application/pdf" })
      const url = window.URL.createObjectURL(blob)
      window.open(url, "_blank")
    } catch {
      toast.error("Gagal membuka laporan aset.")
    }
  }

  // ---------- Initial load ----------
  useEffect(() => {
    if (!isOpen || !asset?.id) return
    reloadDetail()
    loadSoftware()
  }, [isOpen, asset?.id])

  if (!data) return null

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span>{data.asset_tag}</span>
            <span>—</span>
            <span>{data.name}</span>
            <Badge variant="secondary" className="capitalize">
              {data.status}
            </Badge>
          </DialogTitle>
        </DialogHeader>

        <Tabs defaultValue="info" className="mt-4">
          <TabsList className="grid grid-cols-6">
            <TabsTrigger value="info">Informasi</TabsTrigger>
            <TabsTrigger value="finance">Keuangan</TabsTrigger>
            <TabsTrigger value="depreciation">Depresiasi</TabsTrigger>
            <TabsTrigger value="software">Software</TabsTrigger>
            <TabsTrigger value="governance">Governance</TabsTrigger>
            <TabsTrigger value="history">History</TabsTrigger>
          </TabsList>

          {/* INFORMASI */}
          <TabsContent value="info" className="space-y-3 pt-3">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <InfoItem label="Tipe" value={data.asset_type_name} />
              <InfoItem label="Lokasi" value={data.current_location_text} />
              <InfoItem label="Departemen" value={data.owner_department_name} />
              <InfoItem
                label="Assigned To"
                value={data.assigned_to_employee_name}
              />
              <InfoItem label="Kondisi" value={data.asset_condition} />
              <InfoItem label="Kepemilikan" value={data.ownership_type} />
              <InfoItem label="Akuisisi" value={data.acquisition_type} />
            </div>
          </TabsContent>

          {/* KEUANGAN */}
          <TabsContent value="finance" className="space-y-3 pt-3">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <InfoItem
                label="Tanggal Pembelian"
                value={formatDate(data.purchase_date as any)}
              />
              <InfoItem
                label="Harga Perolehan"
                value={formatIDR(data.initial_price as any)}
              />
              <InfoItem
                label="Harga Pembelian"
                value={formatIDR(data.purchase_cost as any)}
              />
              <InfoItem label="Vendor" value={data.vendor} />
              <InfoItem
                label="Garansi S/D"
                value={formatDate(data.warranty_expiry as any)}
              />
            </div>
          </TabsContent>

          {/* DEPRESIASI */}
          <TabsContent value="depreciation" className="space-y-3 pt-3">
            {data.depreciation ? (
              <div className="space-y-2 text-sm">
                <InfoItem label="Metode" value={data.depreciation.method} />
                <InfoItem
                  label="Umur Manfaat"
                  value={
                    data.depreciation.useful_life_months
                      ? `${data.depreciation.useful_life_months} bulan`
                      : "-"
                  }
                />
                <InfoItem
                  label="Nilai Residu"
                  value={formatIDR(
                    data.depreciation.salvage_value as number | null
                  )}
                />
                <InfoItem
                  label="Nilai Buku"
                  value={formatIDR(
                    data.depreciation.book_value as number | null
                  )}
                />
                <InfoItem
                  label="Akumulasi"
                  value={formatIDR(
                    data.depreciation.accumulated as number | null
                  )}
                />
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Tidak ada data depresiasi.
              </p>
            )}
          </TabsContent>

          {/* SOFTWARE */}
          <TabsContent value="software" className="space-y-3 pt-3">
            {softwareList && softwareList.length > 0 ? (
              <div className="space-y-2">
                {softwareList.map((sw) => (
                  <div
                    key={sw.installation_id}
                    className="flex justify-between items-center border p-2 rounded-md"
                  >
                    <div>
                      <p className="font-medium">{sw.license_name}</p>
                      <p className="text-sm text-muted-foreground">
                        {sw.license_key || "-"}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        Installed:{" "}
                        {new Date(sw.installation_date).toLocaleDateString(
                          "id-ID"
                        )}
                      </p>
                    </div>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleUninstall(sw.installation_id)}
                    >
                      Hapus
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                Belum ada software terinstal.
              </p>
            )}

            <div className="pt-3">
              <Button size="sm" onClick={() => setOpenInstallModal(true)}>
                Install Software
              </Button>
            </div>

            <InstallSoftwareModal
              isOpen={openInstallModal}
              onClose={() => setOpenInstallModal(false)}
              assetId={asset?.id ?? null}
              onSuccess={() => {
                loadSoftware()
                setOpenInstallModal(false)
              }}
            />
          </TabsContent>

          {/* GOVERNANCE & COMPLIANCE */}
          <TabsContent value="governance" className="space-y-4 pt-3">
            <div className="space-y-2 text-sm">
              <InfoItem label="Budget ID" value={data.budget_id ?? "-"} />
              <InfoItem label="Contract ID" value={data.contract_id ?? "-"} />
              <InfoItem
                label="Lifecycle Stage"
                value={data.lifecycle_stage ?? "-"}
              />
              <InfoItem
                label="Asset Criticality"
                value={data.asset_criticality ?? "-"}
              />

              {/* Health Status */}
              <div className="mt-4">
                <h4 className="font-semibold mb-2">Health Status</h4>
                <div className="flex items-center gap-3">
                  <div className="relative w-2/3">
                    <div className="h-3 w-full bg-gray-200 rounded-full overflow-hidden">
                      <div
                        className={`h-3 ${mapHealthToColor(
                          mapHealthToValue(
                            data.asset_condition,
                            data.asset_health_score as number | null
                          )
                        )} rounded-full transition-all duration-500`}
                        style={{
                          width: `${mapHealthToValue(
                            data.asset_condition,
                            data.asset_health_score as number | null
                          )}%`,
                        }}
                      />
                    </div>
                  </div>
                  <Badge
                    className={
                      data.asset_condition === "Good"
                        ? "bg-green-100 text-green-700"
                        : data.asset_condition === "Fair"
                        ? "bg-yellow-100 text-yellow-700"
                        : "bg-red-100 text-red-700"
                    }
                  >
                    {data.asset_condition || "Unknown"}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-1">
                  Health Score:{" "}
                  {mapHealthToValue(
                    data.asset_condition,
                    data.asset_health_score as number | null
                  )}
                  %
                </p>
              </div>

              {/* Compliance Section */}
              <div className="border-t pt-3 mt-4">
                <h4 className="font-semibold mb-1">Compliance Status</h4>
                <Badge className={complianceColor}>
                  {data.compliance_flag === true
                    ? "Compliant"
                    : data.compliance_flag === false
                    ? "Non-Compliant"
                    : "Pending"}
                </Badge>
                <p className="text-sm text-muted-foreground mt-1">
                  {data.compliance_note || "—"}
                </p>
                <Button
                  size="sm"
                  className="mt-3"
                  onClick={handleVerifyCompliance}
                  disabled={verifying}
                >
                  {verifying ? "Memverifikasi..." : "Verify Compliance"}
                </Button>
              </div>

              {/* Asset Report Export */}
              <div className="border-t pt-4 mt-4 flex justify-between items-center">
                <p className="text-sm text-muted-foreground">
                  Unduh laporan lengkap aset dalam format PDF.
                </p>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleDownloadReport(data.id)}
                  className="flex items-center gap-2"
                >
                  <FileText className="h-4 w-4" />
                  Open Asset Report
                </Button>
              </div>
            </div>
          </TabsContent>

          {/* HISTORY (Assignment + Audit Log) */}
          <TabsContent value="history" className="pt-4">
            <div className="border rounded-md p-2 max-h-[400px] overflow-y-auto space-y-4">
              <div className="border rounded-md p-2">
                <AssignmentHistory assetId={data.id} />
              </div>
              <div className="border rounded-md p-2">
                <AssetAuditHistory assetId={data.id} />
              </div>
            </div>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

/** Komponen kecil label-value */
function InfoItem({
  label,
  value,
}: {
  label: string
  value?: string | number | null
}) {
  return (
    <div>
      <p className="text-muted-foreground">{label}</p>
      <p className="font-medium">{value ?? "-"}</p>
    </div>
  )
}

// --- Health helpers ---
function mapHealthToValue(condition?: string | null, score?: number | null) {
  if (typeof score === "number") {
    return Math.max(0, Math.min(score, 100))
  }
  switch (condition) {
    case "Excellent":
    case "Good":
      return 90
    case "Fair":
    case "Moderate":
      return 60
    case "Poor":
    case "Critical":
      return 30
    default:
      return 50
  }
}

function mapHealthToColor(value: number) {
  if (value >= 80) return "bg-green-500"
  if (value >= 50) return "bg-yellow-500"
  return "bg-red-500"
}
