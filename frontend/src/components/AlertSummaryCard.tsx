import { AlertTriangle, CheckCircle2 } from "lucide-react"

export default function AlertSummaryCard({ data }: { data: any[] }) {
  if (!data?.length) return null
  return (
    <div className="grid grid-cols-2 gap-2">
      {data.map((a, idx) => (
        <div
          key={idx}
          className="flex items-center justify-between rounded-md border p-2 shadow-sm"
        >
          <div className="flex items-center gap-2">
            {a.title.toLowerCase().includes("critical") ? (
              <AlertTriangle className="h-4 w-4 text-destructive" />
            ) : (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            )}
            <span className="text-sm font-medium">{a.title}</span>
          </div>
          <span className="font-semibold">{a.value}</span>
        </div>
      ))}
    </div>
  )
}
