import { useEffect, useState } from "react"
import { getComplianceSummary } from "@/services/complianceService"

export default function ComplianceCard() {
  const [summary, setSummary] = useState<{ compliant: number; total: number } | null>(null)

  useEffect(() => {
    getComplianceSummary()
      .then((res) => setSummary(res.summary))
      .catch(console.error)
  }, [])

  if (!summary)
    return (
      <div className="p-4 shadow rounded bg-white">
        Loading compliance...
      </div>
    )

  const percent = summary.total
    ? Math.round((summary.compliant / summary.total) * 100)
    : 0

  return (
    <div className="p-4 shadow rounded bg-white">
      <h3 className="text-lg font-semibold mb-2">Compliance Status</h3>
      <p
        className={`text-2xl font-bold ${
          percent >= 80
            ? "text-green-600"
            : percent >= 50
            ? "text-yellow-500"
            : "text-red-600"
        }`}
      >
        {percent}%
      </p>
      <p className="text-sm text-gray-500">
        {summary.compliant} of {summary.total} assets compliant
      </p>
    </div>
  )
}
