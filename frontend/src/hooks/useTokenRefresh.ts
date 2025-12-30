import { useEffect } from "react"
import { jwtDecode } from "jwt-decode"
import { useAuthContext } from "@/context/AuthContext"
import { toast } from "sonner"
import apiClient from "@/services/api"

export function useTokenRefresh() {
  const { token, login, logout } = useAuthContext()

  useEffect(() => {
    if (!token) return
    try {
      const { exp }: any = jwtDecode(token)
      if (!exp) return

      const now = Date.now()
      const msToExpire = exp * 1000 - now
      const refreshTime = msToExpire - 60_000 // refresh 1 menit sebelum expired

      if (refreshTime <= 0) {
        handleRefresh()
        return
      }

      const timer = setTimeout(handleRefresh, refreshTime)
      return () => clearTimeout(timer)
    } catch {
      logout()
    }
  }, [token])

  async function handleRefresh() {
    try {
      const res = await apiClient.post("/auth/refresh", {}, {
        headers: { "X-Refresh-Token": localStorage.getItem("refreshToken") },
      })
      const newToken = res.data.token
      if (newToken) {
        localStorage.setItem("authToken", newToken)
        login(newToken)
        toast.success("Session refreshed securely.")
      }
    } catch (err) {
      toast.error("Session expired. Please login again.")
      logout()
    }
  }
}
