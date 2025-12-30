import { BrowserRouter, Routes, Route, Navigate  } from "react-router-dom"
import { AuthProvider } from "@/context/AuthContext"
import ProtectedRoute from "@/router/ProtectedRoute"
import { RoleGuard } from "@/router/RoleGuard"
import Layout from "@/components/Layout"
import AutoLogoutWatcher from "@/components/AutoLogoutWatcher"
export { useWebSocket as useWebSocketAlert } from "@/hooks/useWebSocket"
import { useTokenRefresh } from "@/hooks/useTokenRefresh"

// Pages
import LoginPage from "../pages/LoginPage"
import ChangePasswordPage from "../pages/ChangePasswordPage"
import DashboardPage from "../pages/DashboardPage"
import AssetsPage from "../pages/AssetsPage"
import AddAssetPage from "../pages/AddAssetPage"
import AssetTypesPage from "../pages/AssetTypesPage"
import MyAssetsPage from "../pages/MyAssetsPage"
import LicensesPage from "../pages/LicensesPage"
import EmployeesPage from "../pages/EmployeesPage"
import DepartmentsPage from "../pages/DepartmentsPage"
import DepartmentSummaryPage from "../pages/DepartmentSummaryPage"
import BudgetsPage from "../pages/BudgetsPage"
import ReportsPage from "../pages/ReportsPage"
import TicketsPage from "../pages/TicketsPage"
import TicketDetailPage from "../pages/TicketDetailPage"
import ContractsPage from "../pages/ContractsPage"
import AuditsPage from "../pages/AuditsPage"
import AuditSessionPage from "../pages/AuditSessionPage"
import SLAPoliciesPage from "../pages/SLAPoliciesPage"
import ComplianceDashboardPage from "../pages/ComplianceDashboardPage"
import LicensesCompliancePage from "../pages/VerificationLogsPage"
import AlertsCenterPage from "../pages/AlertsCenterPage"
import AlertsHistoryPage from "../pages/AlertsHistoryPage"
import CorrelationDashboardPage from "../pages/CorrelationDashboardPage"
import MyTrainingsPage from "../pages/MyTrainingsPage"
import EmployeeTrainingPage from "../pages/EmployeeTrainingPage"
import SecurityDashboardPage from "../pages/SecurityDashboardPage"
import SecurityAnomalyPage from "../pages/SecurityAnomalyPage"
import SecurityRiskAdvisorPage from "../pages/SecurityRiskAdvisorPage"
import AuditLogsDashboardPage from "../pages/AuditLogsDashboardPage"
import LocationsPage from "../pages/LocationsPage"
import CostCentersPage from "../pages/CostCentersPage"

import { useWebSocket } from "@/hooks/useWebSocket"

export default function App() {
  const { connected } = useWebSocket()
  useTokenRefresh()

  return (
    <AuthProvider>
      <BrowserRouter>
        <AutoLogoutWatcher />
        <div className="fixed bottom-2 right-4 text-xs text-muted-foreground">
          {connected ? "🟢 Realtime Connected" : "⚫ Offline"}
        </div>

        <Routes>
          {/* ---------- PUBLIC ROUTES ---------- */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/change-password" element={<ChangePasswordPage />} />

          {/* ===================== AUTHENTICATED (ALL ROLES) ===================== */}
          <Route element={<ProtectedRoute />}>
            <Route path="/" element={<Layout />}> {/* 403 fallback */} <Route path="/403" element={ <div className="p-8 text-center"> <h1 className="text-2xl font-bold text-red-600 mb-2"> 403 Forbidden </h1> <p className="text-muted-foreground"> Anda tidak memiliki izin untuk mengakses halaman ini. </p> </div> } />

              {/* DASHBOARD */}
              <Route index element={<DashboardPage />} />

              {/* ASSETS (READ FOR ALL ROLES) */}
              <Route path="assets" element={ <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}> <AssetsPage /> </RoleGuard> } />
              

              {/* ADD ASSET → ADMIN ONLY */}
              <Route
                path="assets/add"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <AddAssetPage />
                  </RoleGuard>
                }
              />

              {/* ASSET TYPES → ADMIN ONLY */}
              <Route
                path="asset-types"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <AssetTypesPage />
                  </RoleGuard>
                }
              />

              {/* EMPLOYEE PERSONAL */}
              <Route path="my-assets" element={<MyAssetsPage />} />
              <Route path="my-trainings" element={<MyTrainingsPage />} />

              {/* EMPLOYEES CRUD → SUPER ADMIN ONLY */}
              <Route
                path="employees"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <EmployeesPage />
                  </RoleGuard>
                }
              />

              {/* EMPLOYEE TRAINING → SUPER ADMIN ONLY */}
              <Route
                path="employees/:id/trainings"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <EmployeeTrainingPage />
                  </RoleGuard>
                }
              />

              {/* DEPARTMENTS CRUD → SUPER ADMIN ONLY */}
              <Route
                path="departments"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <DepartmentsPage />
                  </RoleGuard>
                }
              />

              <Route
                path="departments/:id/summary"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <DepartmentSummaryPage />
                  </RoleGuard>
                }
              />

              {/* TICKETS (ALL AUTHENTICATED) */}
              <Route path="tickets" element={<TicketsPage />} />
              <Route path="tickets/:id" element={<TicketDetailPage />} />

              {/* SLA POLICIES → MANAGERIAL */}
              <Route
                path="sla-policies"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <SLAPoliciesPage />
                  </RoleGuard>
                }
              />

              {/* FINANCE / CONTRACTS */}
              <Route
                path="budgets"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <BudgetsPage />
                  </RoleGuard>
                }
              />

              <Route path="contracts" element={<ContractsPage />} />

              {/* LICENSES (READ ONLY FOR ALL) */}
              <Route path="licenses" element={<LicensesPage />} />

              {/* LICENSE COMPLIANCE → MANAGERIAL */}
              <Route
                path="licenses/compliance"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <LicensesCompliancePage />
                  </RoleGuard>
                }
              />

              {/* COST CENTERS → MANAGERIAL */}
              <Route
                path="cost-centers"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <CostCentersPage />
                  </RoleGuard>
                }
              />

              {/* AUDITS */}
              <Route
                path="audits"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AuditsPage />
                  </RoleGuard>
                }
              />

              <Route
                path="audits/:sessionId"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AuditSessionPage />
                  </RoleGuard>
                }
              />

              {/* COMPLIANCE DASHBOARD → MANAGERIAL */}
              <Route
                path="compliance"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <ComplianceDashboardPage />
                  </RoleGuard>
                }
              />

              {/* ACTIVE ALERTS → MANAGERIAL */}
              <Route
                path="alerts"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AlertsCenterPage />
                  </RoleGuard>
                }
              />

              <Route
                path="alerts/history"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AlertsHistoryPage />
                  </RoleGuard>
                }
              />

              {/* CORRELATION → MANAGERIAL */}
              <Route
                path="correlation"
                element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <CorrelationDashboardPage />
                  </RoleGuard>
                }
              />

              {/* SECURITY & GOVERNANCE → SUPER ADMIN ONLY */}
              <Route
                path="security"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityDashboardPage />
                  </RoleGuard>
                }
              />

              <Route
                path="anomaly"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityAnomalyPage />
                  </RoleGuard>
                }
              />

              <Route
                path="risk-advisor"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityRiskAdvisorPage />
                  </RoleGuard>
                }
              />

              <Route
                path="audit-dashboard"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <AuditLogsDashboardPage />
                  </RoleGuard>
                }
              />

              {/* REPORTS */}
              <Route path="reports" element={<ReportsPage />} />

              {/* LOCATIONS → ADMIN ONLY */}
              <Route
                path="locations"
                element={
                  <RoleGuard allow={["super_admin"]}>
                    <LocationsPage />
                  </RoleGuard>
                }
              />

            </Route>
          </Route>

          {/* FALLBACK */}
          <Route path="*" element={<Navigate to="/" replace />} />


        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
