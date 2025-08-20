// File: backend/main.go
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/handlers"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	gin.SetMode(os.Getenv("GIN_MODE"))
	database.Connect()
	hub := websocket.GetHub()
	go hub.Run()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Use(static.Serve("/", static.LocalFile("./frontend/dist", false)))

	api := r.Group("/api/v1")
	{
		api.GET("/ws", handlers.WebSocketHandler)
		authRoutes := api.Group("/auth")
		{
			authRoutes.POST("/login", handlers.Login)
		}

		// Grup untuk semua pengguna yang sudah login
		authenticated := api.Group("")
		authenticated.Use(middleware.Authenticate())
		{
			// Endpoint yang bisa diakses SEMUA peran
			authenticated.PUT("/employees/me/change-password", handlers.ChangePassword)
			authenticated.GET("/employees/me/assets", handlers.GetMyAssignedAssets)
			authenticated.POST("/tickets", handlers.CreateTicket)
			authenticated.GET("/tickets", handlers.GetAllTickets)
			authenticated.GET("/tickets/:id", handlers.GetTicketByID)
			authenticated.POST("/tickets/:id/comments", handlers.AddCommentToTicket)
			authenticated.GET("/dashboard/stats", handlers.GetDashboardStats)
			authenticated.GET("/reports/assets-by-department", handlers.GetAssetsByDepartmentReport)

			// --- PERBAIKAN DI SINI: Semua endpoint GET yang read-only ditaruh di sini ---
			authenticated.GET("/assets", handlers.GetAllAssets)
			authenticated.GET("/assets/:id", handlers.GetAssetByID)
			authenticated.GET("/employees", handlers.GetAllEmployees)
			authenticated.GET("/departments", handlers.GetAllDepartments)
			authenticated.GET("/asset-types", handlers.GetAllAssetTypes)
			authenticated.GET("/licenses", handlers.GetAllLicenses)
			authenticated.GET("/licenses/:id", handlers.GetLicenseByID)
			authenticated.GET("/assets/:id/history", handlers.GetAssetHistory)
			authenticated.GET("/assets/:id/depreciation", handlers.GetAssetDepreciation)
			authenticated.GET("/assets/:id/maintenance-logs", handlers.GetMaintenanceLogs)
			authenticated.GET("/assets/:id/software", handlers.GetSoftwareForAsset)
		}

		// Grup API khusus untuk SUPER ADMIN (hanya untuk aksi CUD - Create, Update, Delete)
		adminOnly := api.Group("")
		adminOnly.Use(middleware.Authenticate(), middleware.Authorize("super_admin"))
		{
			// Aksi-aksi yang mengubah data
			adminOnly.POST("/assets", handlers.CreateAsset)
			adminOnly.PUT("/assets/:id", handlers.UpdateAsset)
			adminOnly.DELETE("/assets/:id", handlers.DeleteAsset)
			adminOnly.POST("/assets/:id/assign", handlers.AssignAsset)
			adminOnly.POST("/assets/:id/return", handlers.ReturnAsset)
			adminOnly.POST("/assets/:id/maintenance-logs", handlers.AddMaintenanceLog)
			adminOnly.POST("/assets/:id/installations", handlers.InstallSoftwareOnAsset)
			adminOnly.POST("/employees", handlers.CreateEmployee)
			adminOnly.PUT("/employees/:id", handlers.UpdateEmployee)
			adminOnly.DELETE("/employees/:id", handlers.DeleteEmployee)
			adminOnly.POST("/departments", handlers.CreateDepartment)
			adminOnly.PUT("/departments/:id", handlers.UpdateDepartment)
			adminOnly.DELETE("/departments/:id", handlers.DeleteDepartment)
			adminOnly.POST("/asset-types", handlers.CreateAssetType)
			adminOnly.PUT("/asset-types/:id", handlers.UpdateAssetType)
			adminOnly.DELETE("/asset-types/:id", handlers.DeleteAssetType)
			adminOnly.POST("/licenses", handlers.CreateLicense)
			adminOnly.PUT("/licenses/:id", handlers.UpdateLicense)
			adminOnly.POST("/budgets", handlers.CreateBudget)
			adminOnly.PUT("/budgets/:id", handlers.UpdateBudget)
			adminOnly.PUT("/tickets/:id", handlers.UpdateTicket)

			// Endpoint GET yang khusus admin juga bisa di sini
			adminOnly.GET("/audits", handlers.GetAllAuditSessions)
			adminOnly.POST("/audits", handlers.CreateAuditSession)
			adminOnly.GET("/audits/:id", handlers.GetAuditSessionDetails)
			adminOnly.POST("/audits/:id/scan", handlers.ScanAssetInSession)
			adminOnly.PUT("/audits/:id/complete", handlers.CompleteAuditSession)
			adminOnly.GET("/budgets", handlers.GetAllBudgets)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.File("./frontend/dist/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		}
	})

	log.Println("Starting server on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
