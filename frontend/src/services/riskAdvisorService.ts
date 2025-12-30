import apiClient from "@/services/api"

export async function fetchSecurityRisks() {
  const res = await apiClient.get("/security/risk-insight")
  return res.data?.recommendations || []
}
