import apiClient from "./api"
import type { AssetCompliance, AuditLog } from "@/types"

// 🔹 Fetch daftar aset + summary
export interface ComplianceSummary {
  compliant: number
  non_compliant: number
  pending: number
  total: number
}

export async function getComplianceSummary(): Promise<{
  data: AssetCompliance[]
  summary: ComplianceSummary
}> {
  const res = await apiClient.get("/assets/compliance-summary")
  return res.data
}

// 🔹 Export CSV
export async function exportComplianceCSV(): Promise<void> {
  const res = await apiClient.get("/assets/compliance-export", {
    responseType: "blob",
    headers: { Accept: "text/csv" },
  })
  const blob = new Blob([res.data], { type: "text/csv;charset=utf-8" })
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = `compliance_report_${new Date().toISOString().split("T")[0]}.csv`
  a.click()
  window.URL.revokeObjectURL(url)
}

// 🔹 Audit logs
export async function getAuditLogs(limit = 10): Promise<AuditLog[]> {
  const res = await apiClient.get(`/compliance/audit-logs?limit=${limit}`)
  return res.data
}

export async function fetchComplianceDetails(category: string) {
  const res = await apiClient.get(`/compliance/details?category=${encodeURIComponent(category)}`)
  return res.data?.data || []
}
