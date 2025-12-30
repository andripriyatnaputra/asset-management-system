// File: src/components/AutoLogoutWatcher.tsx
import { useEffect } from "react"
import { jwtDecode } from "jwt-decode"
import { toast } from "sonner"
import { useAuthContext } from "@/context/AuthContext"

export default function AutoLogoutWatcher() {
  const { token, logout } = useAuthContext()

  useEffect(() => {
    if (!token) return
    try {
      const { exp }: any = jwtDecode(token)
      if (!exp) return
      const timeout = exp * 1000 - Date.now() - 60000 // 1 menit sebelum expired
      if (timeout > 0) {
        const timer = setTimeout(() => {
          toast.warning("Sesi hampir berakhir. Silakan login ulang.")
          logout()
        }, timeout)
        return () => clearTimeout(timer)
      }
    } catch {}
  }, [token])

  return null
}
