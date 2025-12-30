// File: backend/handlers/audit_handler.go
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

type CreateAuditRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateAuditSession membuat sesi audit baru dan mengisi item asetnya
func CreateAuditSession(c *gin.Context) {
	var req CreateAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session name is required"})
		return
	}
	//userID, _ := c.Get("userID")
	userID, _ := c.Get("user_id")

	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Buat sesi audit baru
	var sessionID int64
	sessionQuery := `INSERT INTO audit_sessions (name, created_by_employee_id) VALUES ($1, $2) RETURNING id`
	err = tx.QueryRow(context.Background(), sessionQuery, req.Name, userID).Scan(&sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create audit session"})
		return
	}

	// 2. Ambil semua ID aset yang aktif
	var assetIDs []int64
	assetRows, err := tx.Query(context.Background(), "SELECT id FROM assets WHERE deleted_at IS NULL")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assets for audit"})
		return
	}
	for assetRows.Next() {
		var assetID int64
		assetRows.Scan(&assetID)
		assetIDs = append(assetIDs, assetID)
	}
	assetRows.Close()

	// 3. Masukkan semua aset ke dalam tabel audited_assets
	for _, assetID := range assetIDs {
		_, err = tx.Exec(context.Background(),
			"INSERT INTO audited_assets (session_id, asset_id) VALUES ($1, $2)",
			sessionID, assetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to populate audit items"})
			return
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Audit session created successfully", "session_id": sessionID})
}

// GetAllAuditSessions mengambil semua sesi audit
func GetAllAuditSessions(c *gin.Context) {
	// Untuk saat ini, kita buat sederhana. Nanti bisa ditambahkan pagination.
	var sessions []models.AuditSession
	query := `SELECT id, name, status, created_by_employee_id, created_at, completed_at 
			  FROM audit_sessions ORDER BY created_at DESC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit sessions"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s models.AuditSession
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.CreatedByEmployeeID, &s.CreatedAt, &s.CompletedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan session data"})
			return
		}
		sessions = append(sessions, s)
	}
	c.JSON(http.StatusOK, sessions)
}

type AuditedAssetInfo struct {
	AssetName   string     `json:"asset_name"`
	AssetTag    string     `json:"asset_tag"`
	AssetStatus string     `json:"asset_status"` // Status aset saat ini (In Stock, etc)
	AuditStatus string     `json:"audit_status"` // Status audit (Found, Missing)
	FoundAt     *time.Time `json:"found_at"`
}

// GetAuditSessionDetails mengambil detail lengkap dari sebuah sesi audit
func GetAuditSessionDetails(c *gin.Context) {
	sessionID := c.Param("id")
	var session models.AuditSession
	var items []AuditedAssetInfo

	// Ambil detail sesi
	err := database.Pool.QueryRow(context.Background(), "SELECT id, name, status, created_at FROM audit_sessions WHERE id = $1", sessionID).Scan(&session.ID, &session.Name, &session.Status, &session.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audit session not found"})
		return
	}

	// Ambil daftar item aset dalam sesi audit
	queryItems := `
		SELECT a.name, a.asset_tag, a.status, aa.status, aa.found_at
		FROM audited_assets aa
		JOIN assets a ON aa.asset_id = a.id
		WHERE aa.session_id = $1 ORDER BY a.name`

	rows, err := database.Pool.Query(context.Background(), queryItems, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audited items"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var item AuditedAssetInfo
		rows.Scan(&item.AssetName, &item.AssetTag, &item.AssetStatus, &item.AuditStatus, &item.FoundAt)
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"items":   items,
	})
}

type ScanRequest struct {
	AssetTag string `json:"asset_tag" binding:"required"`
}

// ScanAssetInSession menandai sebuah aset sebagai "Found"
func ScanAssetInSession(c *gin.Context) {
	sessionID := c.Param("id")
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset_tag is required"})
		return
	}

	// Cari asset_id berdasarkan asset_tag
	var assetID int64
	err := database.Pool.QueryRow(context.Background(), "SELECT id FROM assets WHERE asset_tag = $1 AND deleted_at IS NULL", req.AssetTag).Scan(&assetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset with this tag not found"})
		return
	}

	// Update status di audited_assets
	query := `
		UPDATE audited_assets 
		SET status = 'Found', found_at = NOW() 
		WHERE session_id = $1 AND asset_id = $2 AND status = 'Missing'`

	commandTag, err := database.Pool.Exec(context.Background(), query, sessionID, assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update audit status"})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Asset not in this session or already found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Asset %s marked as found.", req.AssetTag)})
}

func CompleteAuditSession(c *gin.Context) {
	sessionID := c.Param("id")

	query := `UPDATE audit_sessions SET status = 'Completed', completed_at = NOW() WHERE id = $1`

	commandTag, err := database.Pool.Exec(context.Background(), query, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete session"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audit session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Audit session completed successfully"})
}

// =============================================================
// 📘 Governance Audit Log Dashboard (ISO/IEC 19770-10 A.6.3)
// =============================================================

// GetAuditLogsDashboard menampilkan 200 aktivitas terakhir dari tabel audit_logs
func GetAuditLogsDashboard(c *gin.Context) {
	rows, err := database.Pool.Query(
		context.Background(),
		`
		SELECT id,
			   COALESCE(actor_id, 0) AS actor_id,
			   entity_name,
			   entity_id,
			   action,
			   COALESCE(changes::text, '{}') AS changes,
			   COALESCE(ip_address, '-') AS ip_address,
			   COALESCE(user_agent, '-') AS user_agent,
			   COALESCE(request_path, '-') AS request_path,
			   created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT 200
		`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}
	defer rows.Close()

	type LogItem struct {
		ID          int64  `json:"id"`
		ActorID     int64  `json:"actor_id"`
		EntityName  string `json:"entity_name"`
		EntityID    int64  `json:"entity_id"`
		Action      string `json:"action"`
		Changes     string `json:"changes"`
		IPAddress   string `json:"ip_address"`
		UserAgent   string `json:"user_agent"`
		RequestPath string `json:"request_path"`
		CreatedAt   string `json:"created_at"`
	}

	var logs []LogItem
	for rows.Next() {
		var l LogItem
		_ = rows.Scan(
			&l.ID, &l.ActorID, &l.EntityName, &l.EntityID, &l.Action,
			&l.Changes, &l.IPAddress, &l.UserAgent, &l.RequestPath, &l.CreatedAt,
		)
		logs = append(logs, l)
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
