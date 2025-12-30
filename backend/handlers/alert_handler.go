package handlers

import (
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 🔹 GET /api/v1/alerts — daftar histori alert sistem
// ============================================================
func GetAllAlerts(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 200

	query := `
		SELECT id, message, severity, category,
		       acknowledged, acknowledged_by, created_at
		  FROM alerts
		 ORDER BY created_at DESC
		 LIMIT $1
	`

	rows, err := database.Pool.Query(ctx, query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data alert", "detail": err.Error()})
		return
	}
	defer rows.Close()

	type Alert struct {
		ID             int64     `json:"id"`
		Message        string    `json:"message"`
		Severity       string    `json:"severity"`
		Category       string    `json:"category"`
		Acknowledged   bool      `json:"acknowledged"`
		AcknowledgedBy *int64    `json:"acknowledged_by,omitempty"`
		CreatedAt      time.Time `json:"created_at"`
	}

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Message, &a.Severity, &a.Category,
			&a.Acknowledged, &a.AcknowledgedBy, &a.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses data alert"})
			return
		}
		alerts = append(alerts, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// ============================================================
// 🔹 PATCH /api/v1/alerts/:id/ack — acknowledge alert
// ============================================================
func AcknowledgeAlert(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	uid, _ := c.Get("user_id")
	var userID int64
	switch v := uid.(type) {
	case int:
		userID = int64(v)
	case int64:
		userID = v
	}

	// Pastikan alert belum di-ack sebelumnya
	var alreadyAck bool
	err := database.Pool.QueryRow(ctx, `
		SELECT acknowledged FROM alerts WHERE id = $1
	`, id).Scan(&alreadyAck)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert tidak ditemukan"})
		return
	}
	if alreadyAck {
		c.JSON(http.StatusOK, gin.H{"message": "Alert sudah di-acknowledge sebelumnya"})
		return
	}

	// Update status acknowledged
	cmd, err := database.Pool.Exec(ctx, `
		UPDATE alerts
		   SET acknowledged = TRUE,
		       acknowledged_by = $1
		 WHERE id = $2
	`, userID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status alert", "detail": err.Error()})
		return
	}

	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert tidak ditemukan"})
		return
	}

	middleware.LogAction(c, "alerts", mustAtoi64(id), "ACKNOWLEDGE", gin.H{"ack_by": userID})
	c.JSON(http.StatusOK, gin.H{"message": "Alert berhasil di-acknowledge"})
}

// ============================================================
// 🔹 GET /api/v1/alerts/unack — alert yang belum di-ack
// ============================================================
func GetUnacknowledgedAlerts(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT id, message, severity, category, created_at
		  FROM alerts
		 WHERE acknowledged = FALSE
		 ORDER BY created_at DESC
		 LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data alert", "detail": err.Error()})
		return
	}
	defer rows.Close()

	type Alert struct {
		ID        int64     `json:"id"`
		Message   string    `json:"message"`
		Severity  string    `json:"severity"`
		Category  string    `json:"category"`
		CreatedAt time.Time `json:"created_at"`
	}

	var list []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Message, &a.Severity, &a.Category, &a.CreatedAt); err == nil {
			list = append(list, a)
		}
	}

	c.JSON(http.StatusOK, gin.H{"unacknowledged": list, "count": len(list)})
}
