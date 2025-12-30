import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { toast } from "sonner"

type AuditRecord = {
  id: number
  action: string
  actor_id?: number
  changes?: any
  created_at: string
}

export function AssetAuditHistory({ assetId }: { assetId: number }) {
  const [logs, setLogs] = useState<AuditRecord[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchLogs = async () => {
      try {
        const res = await apiClient.get(`/assets/${assetId}/history`)
        setLogs(res.data.history || [])
      } catch (err) {
        console.error(err)
        toast.error("Gagal memuat audit log aset.")
      } finally {
        setLoading(false)
      }
    }
    if (assetId) fetchLogs()
  }, [assetId])

  if (loading)
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-16 w-full" />
      </div>
    )

  if (logs.length === 0)
    return <p className="text-sm text-muted-foreground">Belum ada audit log untuk aset ini.</p>

  return (
    <div>
      <h4 className="font-semibold mb-3">System Audit Log</h4>
      <div className="space-y-2">
        {logs.map((log) => (
          <Card key={log.id} className="border">
            <CardContent className="py-3 space-y-1">
              <div className="flex justify-between items-center">
                <span className="font-medium capitalize">{log.action}</span>
                <span className="text-xs text-muted-foreground">
                  {new Date(log.created_at).toLocaleString("id-ID")}
                </span>
              </div>

              {log.changes && (
                <p className="text-xs text-muted-foreground">
                  {typeof log.changes === "string"
                    ? log.changes
                    : JSON.stringify(log.changes)}
                </p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
