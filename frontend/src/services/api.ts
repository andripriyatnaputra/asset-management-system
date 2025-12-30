// ✅ Unified API Client
import axios from "axios"

const apiClient = axios.create({
  baseURL: "/api/v1", // ⚠️ penting! semua endpoint dimulai dari /api/v1
})

// 🔒 Tambahkan token Bearer otomatis
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem("authToken")
  if (token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export default apiClient
