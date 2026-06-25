// File: backend/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/handlers"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/security"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	// ✅ Swagger
	_ "github.com/andripriyatnaputra/asset-management-system/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title IT Asset & Compliance API
// @version 1.1
// @description Backend API implementing ISO/IEC 19770-10 Grade A++ RBAC model
// @BasePath /api/v1
func main() {
	_ = godotenv.Load()
	gin.SetMode(os.Getenv("GIN_MODE"))
	database.Connect()

	// --- background jobs ---
	hub := websocket.GetHub()
	go hub.Run()
	go services.RunAlertJobs()
	go services.AdaptiveSLAEngine()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go services.RunAssetComplianceWatcher(ctx)
	go services.RunAutoRemediationWatcher(ctx)
	go services.RunNotificationJobs(ctx)
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				services.RunModelCalibration(ctx)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = services.BuildKnowledgeGraph(ctx)
			}
		}
	}()

	ensureDefaultSLA()

	r := gin.Default()

	// --- unified CORS policy ---
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// --- serve frontend ---
	r.Use(static.Serve("/", static.LocalFile("./frontend/dist", false)))

	// --- swagger & health ---
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/api/v1/health", handlers.HealthCheck)

	api := r.Group("/api/v1")
	{
		api.Use(middleware.AuditLogger())
		api.GET("/ws", handlers.WebSocketHandler)

		// ---------- PUBLIC ----------
		auth := api.Group("/auth")
		auth.Use(middleware.RateLimitAuth())
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/refresh", handlers.RefreshToken)
		}

		// ---------- AUTHENTICATED (ALL USERS) ----------
		authenticated := api.Group("")
		authenticated.Use(middleware.Authenticate(), middleware.RateLimitAPI())
		{
			authenticated.PUT("/employees/me/change-password", handlers.ChangePassword)
			authenticated.GET("/employees/me/assets", handlers.GetMyAssignedAssets)
			authenticated.GET("/employees/me", handlers.GetMyProfile)
			authenticated.PUT("/employees/me", handlers.UpdateMyProfile)
			authenticated.POST("/tickets/:id/comments", handlers.AddCommentToTicket)
			authenticated.GET("/tickets/:id/comments/recent", handlers.GetRecentComments)

			authenticated.GET("/reports/assets-by-department", handlers.GetAssetsByDepartmentReport)
			authenticated.GET("/reports/assets-by-employee", handlers.GetAssetsByEmployeeReport)
			authenticated.GET("/reports/tickets-by-asset-type", handlers.GetTicketsByAssetTypeReport)

			// read-only reference data
			authenticated.GET("/assets", handlers.GetAllAssets)
			authenticated.GET("/assets/:id", handlers.GetAssetByID)
			authenticated.GET("/employees", handlers.ListEmployees)
			authenticated.GET("/departments", handlers.GetAllDepartments)
			authenticated.GET("/departments/:id/summary", handlers.GetDepartmentSummary)
			authenticated.GET("/asset-types", handlers.GetAllAssetTypes)
			authenticated.GET("/licenses", handlers.GetAllLicenses)
			authenticated.GET("/licenses/:id", handlers.GetLicenseByID)
			authenticated.GET("/assets/:id/history", handlers.GetAssetHistory)
			authenticated.GET("/assets/:id/assignment-history", handlers.GetAssetAssignmentHistory)
			authenticated.GET("/assets/:id/depreciation", handlers.GetAssetDepreciation)
			authenticated.GET("/assets/:id/maintenance-logs", handlers.GetMaintenanceLogs)
			authenticated.GET("/assets/:id/software", handlers.GetSoftwareForAsset)
			authenticated.GET("/assets/:id/report", handlers.GetAssetReport)
			authenticated.GET("/locations", handlers.GetAllLocations)
			authenticated.GET("/contracts", handlers.GetAllContracts)
			authenticated.GET("/contracts/:id", handlers.GetContractByID)
			authenticated.GET("/contracts/:id/licenses", handlers.GetLicensesByContract)
			authenticated.GET("/licenses/compliance", handlers.GetLicenseCompliance)
			authenticated.GET("/budgets/dashboard", handlers.GetBudgetDashboard)
			authenticated.GET("/budgets/:id", handlers.GetBudgetByID)
			authenticated.POST("/tickets", handlers.CreateTicket)
			authenticated.GET("/tickets", handlers.GetAllTickets)
			authenticated.GET("/tickets/:id", handlers.GetTicketByID)
			authenticated.GET("/tickets/:id/history", handlers.GetTicketHistory)
			authenticated.GET("/ticket-categories", handlers.GetTicketCategories)
			authenticated.GET("/services", handlers.GetServices)
			authenticated.GET("/dashboard/stats", handlers.GetDashboardStats)
			authenticated.GET("/dashboard/sla", handlers.GetSLADashboard)
			authenticated.POST("/employees/:id/trainings", handlers.AddEmployeeTraining)
			authenticated.GET("/employees/me/trainings", handlers.GetMyTrainings)
			authenticated.GET("/employees/:id/trainings", handlers.GetEmployeeTrainings)
			authenticated.GET("/dashboard/health-status", handlers.GetHealthStatus)
			authenticated.GET("/sla-policies/preview", handlers.PreviewSLA)
			authenticated.GET("/compliance/details", handlers.GetComplianceDetails)
			authenticated.GET("/budget-transactions/audit", handlers.GetBudgetAuditByAsset)
			authenticated.GET("/dashboard/predictive-risk", handlers.GetPredictiveRisk)
			authenticated.GET("/kg/neighborhood", handlers.GetKGNeighborhood)
			authenticated.GET("/employees/:id", handlers.GetEmployee)
			authenticated.GET("/dashboard/kg-summary", handlers.GetKGSummary)

			// Notifications (semua authenticated user)
			authenticated.GET("/notifications", handlers.GetMyNotifications)
			authenticated.GET("/notifications/unread-count", handlers.GetUnreadCount)
			authenticated.PUT("/notifications/:id/read", handlers.MarkNotificationRead)
			authenticated.PUT("/notifications/read-all", handlers.MarkAllRead)
			authenticated.DELETE("/notifications/:id", handlers.DeleteNotification)

			// Phase 7 — Webhooks (read — support+)
			authenticated.GET("/webhooks", handlers.GetWebhooks)
			authenticated.GET("/webhooks/:id/logs", handlers.GetWebhookDeliveryLogs)

			// Phase 7 — QR Codes (read)
			authenticated.GET("/assets/:id/qr", handlers.GetAssetQRData)
			authenticated.GET("/assets/qr/lookup", handlers.LookupByQRData)

			// Phase 7 — DR/BCP (read)
			authenticated.GET("/dr/plans", handlers.GetAllDRPlans)
			authenticated.GET("/dr/plans/:id", handlers.GetDRPlanByID)
			authenticated.GET("/dr/tests", handlers.GetDRTests)
			authenticated.GET("/dr/tests/:id", handlers.GetDRTestByID)
			authenticated.GET("/dr/tests/:id/results", handlers.GetTestResults)

			// Phase 6 — Compliance Reporting (read)
			authenticated.GET("/compliance/frameworks", handlers.GetAllFrameworks)
			authenticated.GET("/compliance/frameworks/:id", handlers.GetFrameworkByID)
			authenticated.GET("/compliance/frameworks/:id/controls", handlers.GetControlsByFramework)
			authenticated.GET("/compliance/controls/:id/evidence", handlers.GetEvidenceByControl)
			authenticated.GET("/compliance/summary", handlers.GetComplianceSummaryByFramework)
			authenticated.GET("/compliance/disposal", handlers.GetDisposalCompliance)

			// Phase 6 — Vendor Performance (read)
			authenticated.GET("/vendors/performance", handlers.GetAllVendorPerformance)
			authenticated.GET("/vendors/performance/:id", handlers.GetVendorPerformanceByID)

			// Phase 6 — Service Availability (read)
			authenticated.GET("/services/availability", handlers.GetServiceAvailability)
			authenticated.GET("/services/availability/summary", handlers.GetServiceAvailabilitySummary)

			// Phase 5 — Asset Specs, SAM Usage, Disposal (read)
			authenticated.GET("/assets/:id/spec", handlers.GetAssetSpecification)
			authenticated.GET("/assets/disposals", handlers.GetAllDisposalRecords)
			authenticated.GET("/assets/:id/disposal", handlers.GetDisposalRecord)
			authenticated.GET("/licenses/reconciliation", handlers.GetLicenseReconciliation)
			authenticated.GET("/licenses/:id/usage", handlers.GetSoftwareUsageLogs)

			// Service Catalog (read — semua authenticated)
			authenticated.GET("/service-catalog", handlers.GetServiceCatalog)

			// Service Requests (read — semua authenticated)
			authenticated.GET("/service-requests", handlers.GetAllServiceRequests)
			authenticated.GET("/service-requests/:id", handlers.GetServiceRequestByID)

			// Approval Workflows (read — semua authenticated)
			authenticated.GET("/approvals", handlers.GetApprovalsByEntity)

			// Change Management (read — semua authenticated)
			authenticated.GET("/change-requests", handlers.GetAllChangeRequests)
			authenticated.GET("/change-requests/:id", handlers.GetChangeRequestByID)

			// Problem Management (read — semua authenticated)
			authenticated.GET("/problems", handlers.GetAllProblems)
			authenticated.GET("/problems/:id", handlers.GetProblemByID)
			authenticated.GET("/tickets/:ticket_id/postmortem", handlers.GetPostmortem)

			// Escalation Rules (read)
			authenticated.GET("/escalation-rules", handlers.GetEscalationRules)

		}

		// ---------- MANAGERIAL ROLES ----------
		managerGroup := api.Group("")
		managerGroup.Use(
			middleware.Authenticate(),
			// hanya role manajerial yang boleh (bukan employee biasa)
			middleware.Authorize("super_admin", "asset_manager", "finance", "it_support", "manager"),
		)
		{
			// Asset Manager & IT Support
			managerGroup.POST("/assets/:id/maintenance-logs", handlers.AddMaintenanceLog)
			managerGroup.POST("/assets/:id/verify-compliance", handlers.VerifyAssetCompliance)
			managerGroup.GET("/assets/compliance-summary", handlers.GetComplianceSummary)
			managerGroup.GET("/audits", handlers.GetAllAuditSessions)

			// Finance
			managerGroup.POST("/budgets/transactions", handlers.CreateBudgetTransactionHandler)
			managerGroup.PUT("/budgets/transactions/:id", handlers.UpdateBudgetTransactionHandler)
			managerGroup.GET("/budgets/summary", handlers.GetBudgetSummary)
			managerGroup.GET("/budgets/transactions", handlers.GetBudgetTransactions)

			// Analytics & Insights
			managerGroup.GET("/dashboard/trend-health-sla", handlers.GetHealthSLATrend)
			managerGroup.GET("/dashboard/forecast-health-sla", handlers.GetHealthSLAForecast)
			managerGroup.GET("/dashboard/alert-stats", handlers.GetAlertStats)
			managerGroup.GET("/dashboard/alert-trends", handlers.GetAlertTrends)
			managerGroup.GET("/dashboard/health-heatmap", handlers.GetDepartmentHealthHeatmap)
			managerGroup.GET("/dashboard/predictive-forecast", handlers.GetPredictiveForecast)
			managerGroup.GET("/dashboard/recommendations", handlers.GetRecommendations)
			managerGroup.GET("/dashboard/correlation", handlers.GetCorrelationMatrix)

			//COst Center
			managerGroup.GET("/cost-centers", handlers.GetCostCenters)
			managerGroup.POST("/cost-centers", handlers.CreateCostCenter)
			managerGroup.PUT("/cost-centers/:id", handlers.UpdateCostCenter)
			managerGroup.DELETE("/cost-centers/:id", handlers.DeleteCostCenter)

			// Excel Export (rate-limited — heavy operations)
			managerGroup.GET("/export/assets.xlsx", middleware.RateLimitExport(), handlers.ExportAssetsExcel)
			managerGroup.GET("/export/licenses.xlsx", middleware.RateLimitExport(), handlers.ExportLicensesExcel)
			managerGroup.GET("/export/audit-logs.xlsx", middleware.RateLimitExport(), handlers.ExportAuditLogsExcel)

			// PDF Export (rate-limited)
			managerGroup.GET("/export/assets.pdf", middleware.RateLimitExport(), handlers.ExportAssetsPDF)
			managerGroup.GET("/export/licenses.pdf", middleware.RateLimitExport(), handlers.ExportLicensesPDF)
			managerGroup.GET("/export/compliance.pdf", middleware.RateLimitExport(), handlers.ExportCompliancePDF)

		}

		// ---------- SUPPORT / ADMIN ----------
		support := api.Group("")
		support.Use(
			middleware.Authenticate(),
			middleware.Authorize("super_admin", "it_support"),
		)
		{
			support.PUT("/tickets/:id", handlers.UpdateTicket)
			support.POST("/tickets/:id/resolve", handlers.ResolveTicket)
			support.POST("/tickets/:id/escalate", handlers.EscalateTicket)

			// Service Requests (write — support & semua authenticated bisa buat)
			support.POST("/service-requests", handlers.CreateServiceRequest)
			support.PUT("/service-requests/:id", handlers.UpdateServiceRequest)
			support.POST("/service-requests/:id/start", handlers.StartFulfillmentServiceRequest)
			support.POST("/service-requests/:id/fulfill", handlers.FulfillServiceRequest)
			support.POST("/service-requests/:id/cancel", handlers.CancelServiceRequest)

			// Approval Workflows (write — support)
			support.POST("/approvals", handlers.AddApprover)
			support.POST("/approvals/decision", handlers.SubmitApprovalDecision)

			// Change Management (write — support & admin)
			support.POST("/change-requests", handlers.CreateChangeRequest)
			support.PUT("/change-requests/:id", handlers.UpdateChangeRequest)
			support.POST("/change-requests/:id/submit", handlers.SubmitChangeRequest)
			support.POST("/change-requests/:id/schedule", handlers.ScheduleChangeRequest)
			support.POST("/change-requests/:id/implement", handlers.ImplementChangeRequest)
			support.POST("/change-requests/:id/complete", handlers.CompleteChangeRequest)
			support.POST("/change-requests/:id/verify", handlers.VerifyChangeRequest)
			support.POST("/change-requests/:id/approvers", handlers.AddCABApprover)
			support.POST("/change-requests/:id/approvers/decision", handlers.SubmitCABDecision)
			support.POST("/change-requests/:id/tasks", handlers.AddChangeTask)
			support.PUT("/change-requests/:id/tasks/:task_id", handlers.UpdateChangeTask)
			support.DELETE("/change-requests/:id/tasks/:task_id", handlers.DeleteChangeTask)

			// Problem Management (write — support & admin)
			support.POST("/problems", handlers.CreateProblem)
			support.PUT("/problems/:id", handlers.UpdateProblem)
			support.POST("/problems/:id/incidents", handlers.LinkIncidentToProblem)
			support.DELETE("/problems/:id/incidents/:ticket_id", handlers.UnlinkIncidentFromProblem)

			// Post-mortem (write — support & admin)
			support.POST("/tickets/:ticket_id/postmortem", handlers.CreatePostmortem)
			support.POST("/tickets/:ticket_id/postmortem/review", handlers.ReviewPostmortem)

			// Phase 5 — Asset Specs & SAM Usage (write — support)
			support.PUT("/assets/:id/spec", handlers.UpsertAssetSpecification)
			support.POST("/licenses/:id/usage", handlers.LogSoftwareUsage)

			// Phase 7 — QR Codes (write — support)
			support.POST("/assets/:id/qr", handlers.GenerateAssetQRCode)
			support.POST("/assets/:id/qr/print", handlers.LogQRPrint)

			// Phase 7 — DR/BCP (write — support)
			support.POST("/dr/plans", handlers.CreateDRPlan)
			support.PUT("/dr/plans/:id", handlers.UpdateDRPlan)
			support.POST("/dr/plans/:id/steps", handlers.AddDRPlanStep)
			support.PUT("/dr/plans/:id/steps/:step_id", handlers.UpdateDRPlanStep)
			support.DELETE("/dr/plans/:id/steps/:step_id", handlers.DeleteDRPlanStep)
			support.POST("/dr/tests/:id/start", handlers.StartDRTest)
			support.POST("/dr/tests/:id/complete", handlers.CompleteDRTest)
			support.POST("/dr/tests/:id/results", handlers.RecordTestResult)

			// Phase 6 — Compliance Evidence & Controls (write — support)
			support.POST("/compliance/frameworks/:id/controls", handlers.CreateControl)
			support.PUT("/compliance/controls/:id", handlers.UpdateControl)
			support.POST("/compliance/controls/:id/evidence", handlers.AddEvidence)
			support.PUT("/compliance/evidence/:id/review", handlers.ReviewEvidence)

			// Phase 6 — Vendor Performance (write — support)
			support.POST("/vendors/performance", handlers.CreateVendorPerformance)
			support.PUT("/vendors/performance/:id", handlers.UpdateVendorPerformance)

			// Phase 6 — Service Availability (write — support)
			support.POST("/services/availability", handlers.RecordServiceAvailability)
			support.PUT("/services/availability/:id", handlers.UpdateServiceAvailability)
		}

		// ---------- ADMIN / SUPER ADMIN ----------
		admin := api.Group("")
		admin.Use(middleware.Authenticate(), middleware.Authorize("super_admin"))
		{
			// Assets
			admin.POST("/assets", handlers.CreateAsset)
			admin.PUT("/assets/:id", handlers.UpdateAsset)
			admin.DELETE("/assets/:id", handlers.DeleteAsset)
			admin.PUT("/assets/:id/dispose", handlers.DisposeAsset)
			admin.POST("/assets/:id/dispose", handlers.DisposeAsset)
			admin.POST("/assets/:id/assign", handlers.AssignAsset)
			admin.POST("/assets/:id/return", handlers.ReturnAsset)
			admin.POST("/assets/:id/software", handlers.InstallSoftwareOnAsset)
			admin.DELETE("/assets/:id/software/:installation_id", handlers.UninstallSoftwareFromAsset)

			// Compliance & Governance
			admin.GET("/assets/compliance-export", handlers.ExportComplianceCSV)
			admin.GET("/alerts", handlers.GetAllAlerts)
			admin.POST("/alerts/:id/ack", handlers.AcknowledgeAlert)
			admin.GET("/alerts/unack", handlers.GetUnacknowledgedAlerts)

			// Employees / Departments / Types
			admin.POST("/employees", handlers.CreateEmployee)
			admin.PUT("/employees/:id", handlers.UpdateEmployee)
			admin.DELETE("/employees/:id", handlers.DeleteEmployee)
			admin.POST("/employees/:id/reset-password", handlers.ResetEmployeePassword)
			admin.POST("/employees/import", handlers.ImportEmployeesFromCSV)
			admin.POST("/departments", handlers.CreateDepartment)
			admin.PUT("/departments/:id", handlers.UpdateDepartment)
			admin.DELETE("/departments/:id", handlers.DeleteDepartment)
			admin.POST("/asset-types", handlers.CreateAssetType)
			admin.PUT("/asset-types/:id", handlers.UpdateAssetType)
			admin.DELETE("/asset-types/:id", handlers.DeleteAssetType)

			// Licenses / Budgets / Locations / Contracts / Audit
			admin.POST("/licenses", handlers.CreateLicense)
			admin.PUT("/licenses/:id", handlers.UpdateLicense)
			admin.DELETE("/licenses/:id", handlers.DeleteLicense)
			admin.POST("/budgets", handlers.CreateBudget)
			admin.PUT("/budgets/:id", handlers.UpdateBudget)
			admin.GET("/budgets", handlers.GetAllBudgets)
			admin.GET("/budgets/report", handlers.GetBudgetReport)
			admin.DELETE("/budgets/transactions/:id", handlers.DeleteBudgetTransaction)
			admin.DELETE("/budgets/:id", handlers.DeleteBudget)
			admin.POST("/locations", handlers.CreateLocation)
			admin.PUT("/locations/:id", handlers.UpdateLocation)
			admin.DELETE("/locations/:id", handlers.DeleteLocation)
			admin.POST("/contracts", handlers.CreateContract)
			admin.PUT("/contracts/:id", handlers.UpdateContract)
			admin.DELETE("/contracts/:id", handlers.DeleteContract)
			admin.POST("/audits", handlers.CreateAuditSession)
			admin.GET("/audits/:id", handlers.GetAuditSessionDetails)
			admin.POST("/audits/:id/scan", handlers.ScanAssetInSession)
			admin.PUT("/audits/:id/complete", handlers.CompleteAuditSession)
			admin.GET("/audit-logs", handlers.GetAuditLogs)
			admin.GET("/audit-logs/:id", handlers.GetAuditLogByID)
			admin.DELETE("/tickets/:id", handlers.DeleteTicket)
			admin.POST("/tickets/:id/assign", handlers.AssignTicket)
			admin.POST("/tickets/:id/close", handlers.CloseTicket)
			admin.GET("/tickets/metrics", handlers.GetTicketMetrics)

			// SLA Policies CRUD
			admin.GET("/sla-policies", handlers.GetAllSLAPolicies)
			admin.POST("/sla-policies", handlers.CreateSLAPolicy)
			admin.PUT("/sla-policies/:id", handlers.UpdateSLAPolicy)
			admin.DELETE("/sla-policies/:id", handlers.DeleteSLAPolicy)
			admin.GET("/sla-policies/:id", handlers.GetSLAPolicyByID)

			admin.GET("/dashboard/audit-logs", handlers.GetAuditLogsDashboard)
			admin.GET("/governance/score-summary", handlers.GetGovernanceScoreSummary)
			admin.GET("/governance/trend", handlers.GetGovernanceTrend)
			admin.GET("/reports/compliance-export", handlers.ExportSystemComplianceCSV)
			admin.GET("/audit/security", handlers.GetSecurityAuditLogs)
			admin.GET("/audit/anomalies", handlers.GetSecurityAnomalies)
			admin.GET("/audit/security/meta", handlers.GetSecurityAuditMeta)
			admin.GET("/security/risk-insight", handlers.GetSecurityRiskInsight)

			// Service Catalog (admin CRUD)
			admin.POST("/service-catalog", handlers.CreateServiceCatalogItem)
			admin.PUT("/service-catalog/:id", handlers.UpdateServiceCatalogItem)
			admin.DELETE("/service-catalog/:id", handlers.DeleteServiceCatalogItem)
			admin.DELETE("/service-requests/:id", handlers.DeleteServiceRequest)

			// Change Management (admin actions)
			admin.POST("/change-requests/:id/approve", handlers.ApproveChangeRequest)
			admin.POST("/change-requests/:id/reject", handlers.RejectChangeRequest)
			admin.POST("/change-requests/:id/close", handlers.CloseChangeRequest)
			admin.DELETE("/change-requests/:id", handlers.DeleteChangeRequest)

			// Problem Management (delete — admin only)
			admin.DELETE("/problems/:id", handlers.DeleteProblem)

			// Phase 5 — Asset Disposal Records (admin only)
			admin.POST("/assets/:id/disposal", handlers.CreateDisposalRecord)
			admin.PUT("/assets/:id/disposal", handlers.UpdateDisposalRecord)

			// Phase 6 — Compliance Frameworks (admin CRUD)
			admin.POST("/compliance/frameworks", handlers.CreateFramework)
			admin.PUT("/compliance/frameworks/:id", handlers.UpdateFramework)
			admin.DELETE("/compliance/frameworks/:id", handlers.DeleteFramework)
			admin.DELETE("/compliance/controls/:id", handlers.DeleteControl)
			admin.DELETE("/compliance/evidence/:id", handlers.DeleteEvidence)

			// Phase 6 — Vendor & Service Availability (admin delete)
			admin.DELETE("/vendors/performance/:id", handlers.DeleteVendorPerformance)
			admin.DELETE("/services/availability/:id", handlers.DeleteServiceAvailability)

			// Phase 7 — Webhooks (admin CRUD)
			admin.POST("/webhooks", handlers.CreateWebhook)
			admin.PUT("/webhooks/:id", handlers.UpdateWebhook)
			admin.DELETE("/webhooks/:id", handlers.DeleteWebhook)
			admin.POST("/webhooks/:id/test", handlers.TestWebhook)

			// Phase 7 — LDAP/AD (admin only)
			admin.GET("/ldap/configs", handlers.GetLDAPConfigs)
			admin.POST("/ldap/configs", handlers.CreateLDAPConfig)
			admin.PUT("/ldap/configs/:id", handlers.UpdateLDAPConfig)
			admin.DELETE("/ldap/configs/:id", handlers.DeleteLDAPConfig)
			admin.POST("/ldap/configs/:id/sync", handlers.TriggerLDAPSync)
			admin.GET("/ldap/configs/:id/logs", handlers.GetLDAPSyncLogs)

			// Phase 7 — DR/BCP (admin: activate, schedule test, delete plan)
			admin.POST("/dr/plans/:id/activate", handlers.ActivateDRPlan)
			admin.DELETE("/dr/plans/:id", handlers.DeleteDRPlan)
			admin.POST("/dr/plans/:id/tests", handlers.ScheduleDRTest)

			// Escalation Rules CRUD (admin only)
			admin.POST("/escalation-rules", handlers.CreateEscalationRule)
			admin.PUT("/escalation-rules/:id", handlers.UpdateEscalationRule)
			admin.DELETE("/escalation-rules/:id", handlers.DeleteEscalationRule)

			admin.POST("/internal/revoke", security.RequireRoles("super_admin"), handlers.RevokeTokenHandler)

		}

	}

	// ---------- SPA FALLBACK ----------
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api") &&
			!strings.HasPrefix(path, "/swagger") &&
			!strings.HasSuffix(path, ".js") &&
			!strings.HasSuffix(path, ".css") &&
			!strings.HasSuffix(path, ".ico") &&
			!strings.HasSuffix(path, ".png") {
			c.File("./frontend/dist/index.html")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
	})

	log.Println("✅ ITAM Server A++ started on :8080 (Swagger → /swagger/index.html)")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("❌ failed to run server: %v", err)
	}
}

func ensureDefaultSLA() {
	_, err := database.Pool.Exec(context.Background(), `
		INSERT INTO sla_policies (name, category_code, service_code, impact, urgency, resulting_priority, response_minutes, resolve_minutes)
		SELECT 'Default SLA', NULL, NULL, 'Medium', 'Medium', 'Medium', 60, 240
		WHERE NOT EXISTS (
			SELECT 1 FROM sla_policies WHERE category_code IS NULL AND service_code IS NULL
		);
	`)
	if err != nil {
		log.Println("[WARN] gagal memastikan default SLA:", err)
	}
}
