// File: src/router/RoleGuard.tsx
import type { ReactNode, JSX } from "react"
import { useAuthContext } from "@/context/AuthContext"

interface RoleGuardProps {
  allow?: string[]
  selfOnly?: boolean
  children: ReactNode
}

/**
 * RoleGuard
 * -------------------------------------------------
 * Membatasi akses tampilan berdasarkan role pengguna.
 * - `allow`: daftar role yang diizinkan
 * - `selfOnly`: bila true, semua role boleh akses tapi data difilter ke milik user
 * - Memperhitungkan delegatedRole / effectiveRole
 */
export const RoleGuard = ({
  allow,
  selfOnly = false,
  children,
}: RoleGuardProps): JSX.Element | null => {
  const { role, effectiveRole } = useAuthContext()

  // 🔹 Gunakan effectiveRole (role aktif) jika ada delegasi
  const currentRole = effectiveRole ?? role

  // 🔹 Jika selfOnly → izinkan semua role tapi hanya menampilkan data user sendiri
  if (selfOnly) {
    return <>{children}</>
  }

  // 🔹 Jika tidak ada role atau tidak termasuk yang diizinkan
  if (!currentRole || (allow && !allow.includes(currentRole))) {
    return null
  }

  return <>{children}</>
}
