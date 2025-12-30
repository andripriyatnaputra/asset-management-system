import apiClient from "./api"
import type { BudgetOverview } from "@/types"

export async function getBudgetDashboard(): Promise<{
  budgets: BudgetOverview[]
  utilization_percent: number
}> {
  const res = await apiClient.get("/dashboard/budgets")
  return res.data
}
