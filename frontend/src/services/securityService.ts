// File: src/services/securityService.ts
import apiClient from "@/services/api"

export async function fetchSecurityAudits(filters?: {
  user_id?: string
  action?: string
  start_date?: string
  end_date?: string
}) {
  try {
    const params = new URLSearchParams()
    if (filters) {
      Object.entries(filters).forEach(([k, v]) => {
        if (v && v.trim() !== "") params.append(k, v)
      })
    }
    const url = `/audit/security${params.size ? `?${params.toString()}` : ""}`
    const res = await apiClient.get(url)
    return res.data?.data || []
  } catch (e) {
    console.error("fetchSecurityAudits failed:", e)
    return []
  }
}

export async function fetchSecurityAuditMeta() {
  try {
    const res = await apiClient.get("/audit/security/meta")
    return (res.data ?? { actors: [], actions: [] }) as {
      actors: { id: number; name: string }[]
      actions: string[]
    }
  } catch (e) {
    console.error("fetchSecurityAuditMeta failed:", e)
    return { actors: [], actions: [] }
  }
}
