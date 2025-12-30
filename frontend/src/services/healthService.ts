import apiClient from "./api"
import type {
  HealthHeatmapRow,
  PredictiveForecastRow,
  CorrelationRow,
} from "@/types"

// 🔹 Heatmap kesehatan per departemen
export async function getHealthHeatmap(): Promise<HealthHeatmapRow[]> {
  const res = await apiClient.get("/dashboard/health-heatmap")
  return res.data.data
}

// 🔹 Prediksi 7 & 30 hari ke depan
export async function getPredictiveForecast(): Promise<PredictiveForecastRow[]> {
  const res = await apiClient.get("/dashboard/predictive-forecast")
  return res.data.data
}

// 🔹 Korelasi aset–ticket–alert
export async function getCorrelationMatrix(): Promise<CorrelationRow[]> {
  const res = await apiClient.get("/dashboard/correlation")
  return res.data.data
}

// 🔹 Status kesehatan sistem global (uptime, aset sehat, alert aktif, dsb)
export async function getHealthStatus() {
  const res = await apiClient.get("/dashboard/health-status")
  return res.data.data
}
