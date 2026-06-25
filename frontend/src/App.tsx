import { lazy, Suspense } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { AuthProvider } from "@/context/AuthContext"
import ProtectedRoute from "@/router/ProtectedRoute"
import { RoleGuard } from "@/router/RoleGuard"
import Layout from "@/components/Layout"
import AutoLogoutWatcher from "@/components/AutoLogoutWatcher"
export { useWebSocket as useWebSocketAlert } from "@/hooks/useWebSocket"
import { useWebSocket } from "@/hooks/useWebSocket"
import { useTokenRefresh } from "@/hooks/useTokenRefresh"

// ── Lazy-loaded pages — each becomes a separate JS chunk ──────────────────
const LoginPage                = lazy(() => import("../pages/LoginPage"))
const ChangePasswordPage       = lazy(() => import("../pages/ChangePasswordPage"))
const DashboardPage            = lazy(() => import("../pages/DashboardPage"))
const AssetsPage               = lazy(() => import("../pages/AssetsPage"))
const AddAssetPage             = lazy(() => import("../pages/AddAssetPage"))
const AssetTypesPage           = lazy(() => import("../pages/AssetTypesPage"))
const MyAssetsPage             = lazy(() => import("../pages/MyAssetsPage"))
const LicensesPage             = lazy(() => import("../pages/LicensesPage"))
const EmployeesPage            = lazy(() => import("../pages/EmployeesPage"))
const DepartmentsPage          = lazy(() => import("../pages/DepartmentsPage"))
const DepartmentSummaryPage    = lazy(() => import("../pages/DepartmentSummaryPage"))
const BudgetsPage              = lazy(() => import("../pages/BudgetsPage"))
const ReportsPage              = lazy(() => import("../pages/ReportsPage"))
const TicketsPage              = lazy(() => import("../pages/TicketsPage"))
const TicketDetailPage         = lazy(() => import("../pages/TicketDetailPage"))
const ContractsPage            = lazy(() => import("../pages/ContractsPage"))
const AuditsPage               = lazy(() => import("../pages/AuditsPage"))
const AuditSessionPage         = lazy(() => import("../pages/AuditSessionPage"))
const SLAPoliciesPage          = lazy(() => import("../pages/SLAPoliciesPage"))
const ComplianceDashboardPage  = lazy(() => import("../pages/ComplianceDashboardPage"))
const LicensesCompliancePage   = lazy(() => import("../pages/VerificationLogsPage"))
const AlertsCenterPage         = lazy(() => import("../pages/AlertsCenterPage"))
const AlertsHistoryPage        = lazy(() => import("../pages/AlertsHistoryPage"))
const CorrelationDashboardPage = lazy(() => import("../pages/CorrelationDashboardPage"))
const MyTrainingsPage          = lazy(() => import("../pages/MyTrainingsPage"))
const EmployeeTrainingPage     = lazy(() => import("../pages/EmployeeTrainingPage"))
const SecurityDashboardPage    = lazy(() => import("../pages/SecurityDashboardPage"))
const SecurityAnomalyPage      = lazy(() => import("../pages/SecurityAnomalyPage"))
const SecurityRiskAdvisorPage  = lazy(() => import("../pages/SecurityRiskAdvisorPage"))
const AuditLogsDashboardPage   = lazy(() => import("../pages/AuditLogsDashboardPage"))
const LocationsPage            = lazy(() => import("../pages/LocationsPage"))
const CostCentersPage          = lazy(() => import("../pages/CostCentersPage"))
const ProblemsPage             = lazy(() => import("../pages/ProblemsPage"))
const ChangeRequestsPage       = lazy(() => import("../pages/ChangeRequestsPage"))
const ServiceRequestsPage      = lazy(() => import("../pages/ServiceRequestsPage"))
const CompliancePage           = lazy(() => import("../pages/CompliancePage"))
const DRBCPPage                = lazy(() => import("../pages/DRBCPPage"))
const IntegrationsPage         = lazy(() => import("../pages/IntegrationsPage"))

function PageLoader() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
    </div>
  )
}

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

        <Suspense fallback={<PageLoader />}>
          <Routes>
            {/* ---------- PUBLIC ROUTES ---------- */}
            <Route path="/login" element={<LoginPage />} />
            <Route path="/change-password" element={<ChangePasswordPage />} />

            {/* ===================== AUTHENTICATED (ALL ROLES) ===================== */}
            <Route element={<ProtectedRoute />}>
              <Route path="/" element={<Layout />}>
                <Route path="/403" element={
                  <div className="p-8 text-center">
                    <h1 className="text-2xl font-bold text-red-600 mb-2">403 Forbidden</h1>
                    <p className="text-muted-foreground">Anda tidak memiliki izin untuk mengakses halaman ini.</p>
                  </div>
                } />

                {/* DASHBOARD */}
                <Route index element={<DashboardPage />} />

                {/* ASSETS */}
                <Route path="assets" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <AssetsPage />
                  </RoleGuard>
                } />
                <Route path="assets/add" element={
                  <RoleGuard allow={["super_admin"]}>
                    <AddAssetPage />
                  </RoleGuard>
                } />
                <Route path="asset-types" element={
                  <RoleGuard allow={["super_admin"]}>
                    <AssetTypesPage />
                  </RoleGuard>
                } />

                {/* EMPLOYEE PERSONAL */}
                <Route path="my-assets" element={<MyAssetsPage />} />
                <Route path="my-trainings" element={<MyTrainingsPage />} />

                {/* EMPLOYEES */}
                <Route path="employees" element={
                  <RoleGuard allow={["super_admin"]}>
                    <EmployeesPage />
                  </RoleGuard>
                } />
                <Route path="employees/:id/trainings" element={
                  <RoleGuard allow={["super_admin"]}>
                    <EmployeeTrainingPage />
                  </RoleGuard>
                } />

                {/* DEPARTMENTS */}
                <Route path="departments" element={
                  <RoleGuard allow={["super_admin"]}>
                    <DepartmentsPage />
                  </RoleGuard>
                } />
                <Route path="departments/:id/summary" element={
                  <RoleGuard allow={["super_admin"]}>
                    <DepartmentSummaryPage />
                  </RoleGuard>
                } />

                {/* TICKETS */}
                <Route path="tickets" element={<TicketsPage />} />
                <Route path="tickets/:id" element={<TicketDetailPage />} />

                {/* SLA */}
                <Route path="sla-policies" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <SLAPoliciesPage />
                  </RoleGuard>
                } />

                {/* FINANCE */}
                <Route path="budgets" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <BudgetsPage />
                  </RoleGuard>
                } />
                <Route path="contracts" element={<ContractsPage />} />
                <Route path="cost-centers" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <CostCentersPage />
                  </RoleGuard>
                } />

                {/* LICENSES */}
                <Route path="licenses" element={<LicensesPage />} />
                <Route path="licenses/compliance" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <LicensesCompliancePage />
                  </RoleGuard>
                } />

                {/* AUDITS */}
                <Route path="audits" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AuditsPage />
                  </RoleGuard>
                } />
                <Route path="audits/:sessionId" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AuditSessionPage />
                  </RoleGuard>
                } />

                {/* COMPLIANCE */}
                <Route path="compliance" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <ComplianceDashboardPage />
                  </RoleGuard>
                } />

                {/* ALERTS */}
                <Route path="alerts" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AlertsCenterPage />
                  </RoleGuard>
                } />
                <Route path="alerts/history" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <AlertsHistoryPage />
                  </RoleGuard>
                } />

                {/* CORRELATION */}
                <Route path="correlation" element={
                  <RoleGuard allow={["super_admin", "manager", "asset_manager", "finance", "it_support"]}>
                    <CorrelationDashboardPage />
                  </RoleGuard>
                } />

                {/* SECURITY */}
                <Route path="security" element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityDashboardPage />
                  </RoleGuard>
                } />
                <Route path="anomaly" element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityAnomalyPage />
                  </RoleGuard>
                } />
                <Route path="risk-advisor" element={
                  <RoleGuard allow={["super_admin"]}>
                    <SecurityRiskAdvisorPage />
                  </RoleGuard>
                } />
                <Route path="audit-dashboard" element={
                  <RoleGuard allow={["super_admin"]}>
                    <AuditLogsDashboardPage />
                  </RoleGuard>
                } />

                {/* ITSM */}
                <Route path="problems" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <ProblemsPage />
                  </RoleGuard>
                } />
                <Route path="change-requests" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <ChangeRequestsPage />
                  </RoleGuard>
                } />
                <Route path="service-requests" element={<ServiceRequestsPage />} />
                <Route path="itsm-compliance" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <CompliancePage />
                  </RoleGuard>
                } />
                <Route path="dr-bcp" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <DRBCPPage />
                  </RoleGuard>
                } />
                <Route path="integrations" element={
                  <RoleGuard allow={["super_admin", "asset_manager", "it_support"]}>
                    <IntegrationsPage />
                  </RoleGuard>
                } />

                {/* MISC */}
                <Route path="reports" element={<ReportsPage />} />
                <Route path="locations" element={
                  <RoleGuard allow={["super_admin"]}>
                    <LocationsPage />
                  </RoleGuard>
                } />
              </Route>
            </Route>

            {/* FALLBACK */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </AuthProvider>
  )
}
