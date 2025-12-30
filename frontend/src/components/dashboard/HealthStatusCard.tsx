import { useEffect, useState } from "react"
import { getHealthStatus } from "@/services/healthService"

interface HealthStatus {
  system_uptime: string
  assets_healthy: number
  assets_at_risk: number
  alerts_active: number
}

export default function HealthStatusCard() {
  const [data, setData] = useState<HealthStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getHealthStatus()
      .then((res) => setData(res))
      .catch(() => setError("Unreachable"))
  }, [])

  // loading & error states
  if (error) {
    return (
      <div className="p-4 shadow rounded bg-white">
        <h3 className="text-lg font-semibold mb-2">System Health</h3>
        <p className="text-2xl font-bold text-red-600">{error}</p>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-4 shadow rounded bg-white">
        <h3 className="text-lg font-semibold mb-2">System Health</h3>
        <p className="text-gray-400">Checking...</p>
      </div>
    )
  }

  const overallHealthy =
    data.alerts_active === 0 && data.assets_at_risk < 5
  const color = overallHealthy ? "text-green-600" : "text-yellow-600"

  return (
    <div className="p-4 shadow rounded bg-white">
      <h3 className="text-lg font-semibold mb-2">System Health</h3>
      <p className={`text-2xl font-bold ${color}`}>
        {overallHealthy ? "Healthy" : "At Risk"}
      </p>

      <ul className="mt-3 space-y-1 text-sm text-gray-600">
        <li>
          <span className="font-medium text-gray-700">Uptime:</span>{" "}
          {data.system_uptime}
        </li>
        <li>
          <span className="font-medium text-gray-700">Assets Healthy:</span>{" "}
          {data.assets_healthy}
        </li>
        <li>
          <span className="font-medium text-gray-700">Assets At Risk:</span>{" "}
          {data.assets_at_risk}
        </li>
        <li>
          <span className="font-medium text-gray-700">Active Alerts:</span>{" "}
          {data.alerts_active}
        </li>
      </ul>
    </div>
  )
}
