import { createContext, useContext, useEffect, useState } from "react"
import type { ReactNode } from "react"
import { jwtDecode } from "jwt-decode"
import { toast } from "sonner"

interface JwtClaims {
  user_id: number
  email: string
  role: string
  department_id?: number
  exp?: number
}

interface AuthInfo {
  token: string | null
  role: string | null
  departmentId: number | null
  userId: number | null
  delegatedRole: string | null
  effectiveRole: string | null
  setEffectiveRole: (r: string | null) => void
  login: (token: string, refreshToken?: string) => void
  logout: () => void
  isAdmin: boolean
  isManager: boolean
  isEmployee: boolean
}

const AuthContext = createContext<AuthInfo>({
  token: null,
  role: null,
  departmentId: null,
  userId: null,
  delegatedRole: null,
  effectiveRole: null,
  setEffectiveRole: () => {},
  login: () => {},
  logout: () => {},
  isAdmin: false,
  isManager: false,
  isEmployee: false,
})

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(localStorage.getItem("authToken"))
  const [role, setRole] = useState<string | null>(null)
  const [departmentId, setDepartmentId] = useState<number | null>(null)
  const [userId, setUserId] = useState<number | null>(null)
  const [delegatedRole, setDelegatedRole] = useState<string | null>(null)
  const [effectiveRole, setEffectiveRole] = useState<string | null>(null)
  const [hydrated, setHydrated] = useState(false)

  // 🧩 Decode JWT + refresh otomatis
  useEffect(() => {
    if (!token) {
      setRole(null)
      setDepartmentId(null)
      setUserId(null)
      setEffectiveRole(null)
      setHydrated(true)
      return
    }

    setHydrated(true)

    try {
      const decoded = jwtDecode<JwtClaims>(token.startsWith("Bearer ") ? token.slice(7) : token)
      const normalizedRole = decoded.role?.toLowerCase() ?? null
      setRole(normalizedRole)
      setDepartmentId(decoded.department_id ?? null)
      setUserId(decoded.user_id)
      setEffectiveRole(normalizedRole)

      // ⏳ auto logout jika expired
      if (decoded.exp && decoded.exp * 1000 < Date.now()) {
        toast.error("Sesi Anda telah berakhir. Silakan login ulang.")
        logout()
        return
      }

      // 🔁 auto refresh 1 menit sebelum exp
      if (decoded.exp) {
        const msUntilExpiry = decoded.exp * 1000 - Date.now()
        const refreshIn = Math.max(msUntilExpiry - 60_000, 0)
        const timer = setTimeout(async () => {
          try {
            const refreshToken = localStorage.getItem("refreshToken")
            if (!refreshToken) return
            const res = await fetch("/api/v1/auth/refresh", {
              method: "POST",
              headers: { "X-Refresh-Token": refreshToken },
            })
            if (res.ok) {
              const data = await res.json()
              if (data.token) {
                localStorage.setItem("authToken", data.token)
                setToken(data.token)
              }
            } else logout()
          } catch {
            logout()
          }
        }, refreshIn)
        return () => clearTimeout(timer)
      }
    } catch (err) {
      console.error("JWT decode failed:", err)
      logout()
    }
  }, [token])

  // 🔐 Sinkronisasi logout antar-tab
  useEffect(() => {
    const syncLogout = (e: StorageEvent) => {
      if (e.key === "authToken" && !e.newValue) logout()
    }
    window.addEventListener("storage", syncLogout)
    return () => window.removeEventListener("storage", syncLogout)
  }, [])

  // 🚀 Login & Logout
  const login = (accessToken: string, refreshToken?: string) => {
    localStorage.setItem("authToken", accessToken)
    if (refreshToken) localStorage.setItem("refreshToken", refreshToken)
    setToken(accessToken)
    try {
      const decoded = jwtDecode<JwtClaims>(accessToken.startsWith("Bearer ") ? accessToken.slice(7) : accessToken)
      const normalizedRole = decoded.role?.toLowerCase() ?? null
      setRole(normalizedRole)
      setDepartmentId(decoded.department_id ?? null)
      setUserId(decoded.user_id)
      setEffectiveRole(normalizedRole)
    } catch {
      console.warn("Token decode failed on login()")
    }
  }

  const logout = () => {
    localStorage.removeItem("authToken")
    localStorage.removeItem("refreshToken")
    setToken(null)
    setRole(null)
    setDepartmentId(null)
    setUserId(null)
    setDelegatedRole(null)
    setEffectiveRole(null)
  }

  // Role helpers
  const isAdmin = role === "super_admin"
  const isManager = ["manager", "asset_manager", "finance", "it_support"].includes(role ?? "")
  const isEmployee = role === "employee"

  return (
    <AuthContext.Provider
      value={{
        token,
        role,
        departmentId,
        userId,
        delegatedRole,
        effectiveRole,
        setEffectiveRole,
        login,
        logout,
        isAdmin,
        isManager,
        isEmployee,
      }}
    >
      {hydrated ? (
        children
      ) : (
        <div className="flex h-screen items-center justify-center text-muted-foreground">
          <p>🔄 Memuat sesi pengguna...</p>
        </div>
      )}
    </AuthContext.Provider>
  )
}

export function useAuthContext() {
  return useContext(AuthContext)
}

export default AuthContext
