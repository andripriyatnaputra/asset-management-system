import apiClient from "@/services/api"

export async function fetchSecurityAnomalies() {
  const res = await apiClient.get("/audit/anomalies")
  return res.data?.data || []
}
