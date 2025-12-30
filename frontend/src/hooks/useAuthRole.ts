// File: src/hooks/useAuthRole.ts
export type AppRole = "super_admin" | "asset_manager" | "it_support" | "finance" | "employee";

function readRoleFromToken(): AppRole {
  try {
    const token = localStorage.getItem("authToken");
    if (!token) return "employee";
    const parts = token.split(".");
    if (parts.length !== 3) return "employee";
    const payload = JSON.parse(atob(parts[1].replace(/-/g, "+").replace(/_/g, "/")));
    const role = (payload?.role ?? "employee") as AppRole;
    return role;
  } catch {
    return "employee";
  }
}

export function useAuthRole() {
  // simple read-only; kalau mau reactive terhadap token change,
  // bisa tambahkan event listener storage atau context global
  const role = readRoleFromToken();
  const canSeePrice = role === "super_admin" || role === "finance";
  const canDoAdminActions = role === "super_admin" || role === "asset_manager";
  return { role, canSeePrice, canDoAdminActions };
}
