import React from "react"
import { Navigate, Outlet } from "react-router-dom"
import { useAuthContext } from "@/context/AuthContext"

interface ProtectedRouteProps {
  allowedRoles?: string[]
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ allowedRoles }) => {
  const { token, role, effectiveRole } = useAuthContext()

  // 🔒 Belum login
  if (!token) {
    return <Navigate to="/login" replace />
  }

  // 🔎 Tentukan role aktif (effectiveRole > role)
  const currentRole = (effectiveRole || role || "").toLowerCase()

  // ⏳ Role belum terdeteksi (misal decoding token sedang berlangsung)
  if (!currentRole) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground">
        <p>🔍 Memverifikasi peran pengguna...</p>
      </div>
    )
  }

  // 🎯 Tidak ada batasan role → siapa pun yang login boleh masuk
  if (!allowedRoles || allowedRoles.length === 0) {
    return <Outlet />
  }

  // Normalisasi daftar role yang diperbolehkan
  const normalizedAllowed = allowedRoles.map((r) => r.toLowerCase())

  // ✔ super_admin selalu override
  if (currentRole === "super_admin") {
    return <Outlet />
  }

  // ❌ Jika role tidak termasuk yang diizinkan
  if (!normalizedAllowed.includes(currentRole)) {
    return <Navigate to="/403" replace />
  }

  // ✔ Jika lolos seluruh pengecekan
  return <Outlet />
}

export default ProtectedRoute
