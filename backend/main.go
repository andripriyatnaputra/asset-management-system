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
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/refresh", handlers.RefreshToken)
		}

		// ---------- AUTHENTICATED (ALL USERS) ----------
		authenticated := api.Group("")
		authenticated.Use(middleware.Authenticate())
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
