import { useEffect, useState } from "react"
import apiClient from "@/services/api"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"

type AssignmentRecord = {
  employee_name: string
  assigned_at: string
  returned_at: string
  status: string
  assigned_by: string
  returned_by: string
}

export function AssignmentHistory({ assetId }: { assetId: number }) {
  const [list, setList] = useState<AssignmentRecord[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await apiClient.get(`/assets/${assetId}/assignment-history`)
        setList(res.data.history || [])
      } catch (err) {
        console.error(err)
        toast.error("Gagal memuat riwayat penugasan.")
      } finally {
        setLoading(false)
      }
    }
    if (assetId) fetchData()
  }, [assetId])

  if (loading)
    return (
      <div className="space-y-2">
        <Skeleton className="h-4 w-36" />
        <Skeleton className="h-16 w-full" />
      </div>
    )

  if (list.length === 0)
    return <p className="text-sm text-muted-foreground">Belum ada riwayat penugasan untuk aset ini.</p>

  return (
    <div>
      <h4 className="font-semibold mb-3">Assignment History</h4>
      <div className="space-y-2">
        {list.map((item, i) => (
          <Card key={i} className="border">
            <CardContent className="py-3">
              <div className="flex justify-between items-center">
                <p className="font-medium">{item.employee_name}</p>
                <Badge variant={item.status === "Active" ? "default" : "secondary"}>
                  {item.status}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {item.assigned_at} → {item.returned_at}
              </p>
              <p className="text-xs text-muted-foreground mt-1">
                <span className="font-medium">Assigned by:</span> {item.assigned_by || "-"}{" "}
                {item.returned_by && (
                  <>
                    {" | "}
                    <span className="font-medium">Returned by:</span> {item.returned_by}
                  </>
                )}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
