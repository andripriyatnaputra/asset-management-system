// =========================
// 📁 File: src/config/nav.ts
// =========================
import {
  Home,
  Box,
  List,
  Users,
  Briefcase,
  HelpCircle,
  DollarSign,
  FileText,
  Shield,
  Laptop,
  Network,
  GraduationCap,
  AlertTriangle,
  Bell,
  Bug,
  Lock,
  Activity,
  MapPin,
  Building2,
  GitBranch,
  Headphones,
  ShieldCheck,
  Plug,
} from "lucide-react"

export type RoleType =
  | "super_admin"
  | "asset_manager"
  | "it_support"
  | "finance"
  | "employee"

export interface NavItem {
  path: string
  label: string
  icon: any
}

export const ROLE_LABELS: Record<RoleType, string> = {
  super_admin: "Super Admin",
  asset_manager: "Asset Manager",
  it_support: "IT Support",
  finance: "Finance",
  employee: "Employee",
}

export const getNavItems = (role: RoleType): NavItem[] => {
  const base: NavItem[] = [
    { path: "/", label: "Dashboard", icon: Home },
    { path: "/tickets", label: "Tickets", icon: HelpCircle },
    { path: "/service-requests", label: "Service Requests", icon: Headphones },
  ]

  const employeeOnly: NavItem[] = [
    { path: "/my-assets", label: "My Assets", icon: Laptop },
    { path: "/my-trainings", label: "My Trainings", icon: GraduationCap },
  ]

  const manager: NavItem[] = [
    { path: "/assets", label: "Assets", icon: Box },
    { path: "/asset-types", label: "Asset Types", icon: List },
    { path: "/departments", label: "Departments", icon: Users },
    { path: "/employees", label: "Employees", icon: Users },
    { path: "/reports", label: "Reports", icon: FileText },
    { path: "/locations", label: "Locations", icon: MapPin },
  ]

  const itSupport: NavItem[] = [
    { path: "/sla-policies", label: "SLA Policies", icon: Shield },
    { path: "/alerts", label: "Alerts Center", icon: Bell },
    { path: "/alerts/history", label: "Alerts History", icon: AlertTriangle },
    { path: "/correlation", label: "Correlation Dashboard", icon: Activity },
    { path: "/problems", label: "Problem Management", icon: Bug },
    { path: "/change-requests", label: "Change Management", icon: GitBranch },
    { path: "/itsm-compliance", label: "Compliance & Audit", icon: ShieldCheck },
    { path: "/dr-bcp", label: "DR / BCP", icon: Shield },
    { path: "/integrations", label: "Integrations", icon: Plug },
  ]

  const finance: NavItem[] = [
    { path: "/budgets", label: "Budgets", icon: DollarSign },
    { path: "/contracts", label: "Contracts", icon: Briefcase },
    { path: "/cost-centers", label: "Cost Centers", icon: Building2 },
  ]

  const superAdmin: NavItem[] = [
    { path: "/licenses", label: "Licenses", icon: Shield },
    { path: "/audits", label: "Audits", icon: Network },
    { path: "/compliance", label: "Compliance", icon: Shield },
    { path: "/security", label: "Security Dashboard", icon: Lock },
    { path: "/anomaly", label: "Anomaly Detection", icon: Bug },
    { path: "/risk-advisor", label: "Risk Advisor", icon: Activity },
  ]

  switch (role) {
    case "employee":
      return [...base, ...employeeOnly]
    case "asset_manager":
      return [...base, ...manager]
    case "it_support":
      return [...base, ...manager, ...itSupport]
    case "finance":
      return [...base, ...manager, ...finance]
    case "super_admin":
      return [
        ...base,
        ...manager,
        ...itSupport,
        ...finance,
        ...superAdmin,
      ]
    default:
      return base
  }
}
