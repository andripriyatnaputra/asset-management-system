// File: backend/handlers/maintenance_handler.go
package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// AddMaintenanceLog menambahkan catatan maintenance baru untuk sebuah aset
func AddMaintenanceLog(c *gin.Context) {
	assetID := c.Param("id")
	var newLog models.AssetMaintenanceLog

	if err := c.ShouldBindJSON(&newLog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `
		INSERT INTO asset_maintenance_logs (asset_id, ticket_id, log_type, description, cost, log_date)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`

	err := database.Pool.QueryRow(context.Background(), query,
		assetID, newLog.TicketID, newLog.LogType, newLog.Description, newLog.Cost, newLog.LogDate,
	).Scan(&newLog.ID, &newLog.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add maintenance log"})
		return
	}

	newLog.AssetID, _ = strconv.ParseInt(assetID, 10, 64)
	c.JSON(http.StatusCreated, newLog)
}

// GetMaintenanceLogs mengambil semua catatan maintenance untuk sebuah aset
func GetMaintenanceLogs(c *gin.Context) {
	assetID := c.Param("id")
	var logs []models.AssetMaintenanceLog

	query := "SELECT id, asset_id, log_type, description, cost, log_date, created_at FROM asset_maintenance_logs WHERE asset_id = $1 ORDER BY log_date DESC"

	rows, err := database.Pool.Query(context.Background(), query, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch maintenance logs"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var log models.AssetMaintenanceLog
		if err := rows.Scan(&log.ID, &log.AssetID, &log.LogType, &log.Description, &log.Cost, &log.LogDate, &log.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan log data"})
			return
		}
		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, logs)
}
