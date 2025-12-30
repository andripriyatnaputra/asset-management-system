// =========================
// 📁 File: src/config/nav.ts
// ISO Grade A++ Navigation
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
  Wrench,
  LineChart,
  BarChart2,
} from "lucide-react"

export type RoleType =
  | "super_admin"
  | "asset_manager"
  | "it_support"
  | "finance"
  | "employee"
  | "manager"

export interface NavItem {
  path: string
  label: string
  icon: any
}

// ---------------------------------------------
// LEVEL 1: BASE MENU (ALL AUTHENTICATED ROLES)
// ---------------------------------------------
const BASE_MENU: NavItem[] = [
  { path: "/", label: "Dashboard", icon: Home },
  { path: "/assets", label: "Assets", icon: Box },            // read-only allowed by backend
  { path: "/tickets", label: "Tickets", icon: HelpCircle },
  { path: "/reports", label: "Reports", icon: FileText },
]

// ---------------------------------------------
// LEVEL 1b: EMPLOYEE PERSONAL MENU
// ---------------------------------------------
const EMPLOYEE_MENU: NavItem[] = [
  { path: "/my-assets", label: "My Assets", icon: Laptop },
  { path: "/my-trainings", label: "My Trainings", icon: GraduationCap },
]

// ---------------------------------------------
// LEVEL 2: MANAGERIAL MENU (manager, it_support, finance, asset_manager)
// ---------------------------------------------
const MANAGERIAL_MENU: NavItem[] = [
  { path: "/asset-types", label: "Asset Types", icon: List },
  { path: "/employees", label: "Employees", icon: Users },
  { path: "/departments", label: "Departments", icon: Users },
  { path: "/locations", label: "Locations", icon: MapPin },

  // Governance-lite operations
  { path: "/compliance", label: "Compliance Summary", icon: Shield },

  // Maintenance & Verification
  { path: "/maintenance", label: "Asset Maintenance", icon: Wrench },

  // SLA / Alerts / Correlation / AI ops
  { path: "/sla-policies", label: "SLA Dashboard", icon: Shield },
  { path: "/alerts", label: "Alerts Center", icon: Bell },
  { path: "/alerts/history", label: "Alerts History", icon: AlertTriangle },
  { path: "/correlation", label: "Correlation Dashboard", icon: Activity },

  // Predictive Dashboard (backend supported)
  { path: "/predictive", label: "Predictive Insights", icon: LineChart },

  // Budget & Finance Operations
  { path: "/budgets", label: "Budgets", icon: DollarSign },
  { path: "/cost-centers", label: "Cost Centers", icon: Building2 },
  { path: "/contracts", label: "Contracts", icon: Briefcase },
]

// ---------------------------------------------
// LEVEL 3: SUPER ADMIN MENU
// ---------------------------------------------
const ADMIN_MENU: NavItem[] = [
  // CRUD Privileges
  { path: "/licenses", label: "Licenses", icon: Shield },
  { path: "/audits", label: "Audit Sessions", icon: Network },

  // Full governance analytics
  { path: "/governance", label: "Governance Score", icon: BarChart2 },
  { path: "/governance/trend", label: "Governance Trend", icon: Activity },

  // Security domain
  { path: "/security", label: "Security Dashboard", icon: Lock },
  { path: "/anomaly", label: "Anomaly Detection", icon: Bug },
  { path: "/risk-advisor", label: "Risk Advisor", icon: Activity },
  { path: "/audit-logs", label: "Audit Logs", icon: FileText },
]

// ---------------------------------------------
// ROLE → MENU MAPPING (ISO A++ Compliance)
// ---------------------------------------------
export const getNavItems = (role: RoleType): NavItem[] => {
  switch (role) {
    case "employee":
      return [...BASE_MENU, ...EMPLOYEE_MENU]

    case "asset_manager":
    case "finance":
    case "it_support":
    case "manager":
      return [...BASE_MENU, ...MANAGERIAL_MENU]

    case "super_admin":
      return [...BASE_MENU, ...MANAGERIAL_MENU, ...ADMIN_MENU]

    default:
      return BASE_MENU
  }
}
