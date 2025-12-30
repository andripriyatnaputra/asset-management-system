package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 📜 Struktur data AuditLog (response model)
// ============================================================
type AuditLogRecord struct {
	ID          int64     `json:"id"`
	EntityName  string    `json:"entity_name"`
	EntityID    *int64    `json:"entity_id,omitempty"`
	Action      string    `json:"action"`
	ActorID     *int64    `json:"actor_id,omitempty"`
	ActorName   *string   `json:"actor_name,omitempty"`
	Changes     *string   `json:"changes,omitempty"`
	IPAddress   *string   `json:"ip_address,omitempty"`
	UserAgent   *string   `json:"user_agent,omitempty"`
	RequestPath *string   `json:"request_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================================
// 🔹 GET /api/v1/audit-logs
// Support: ?entity=assets&actor=5&action=UPDATE&q=text&from=2025-10-01&to=2025-10-30&page=1&limit=25
// ============================================================
func GetAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Query parameters (filter)
	entity := strings.TrimSpace(c.Query("entity"))
	action := strings.TrimSpace(c.Query("action"))
	actor := strings.TrimSpace(c.Query("actor"))
	search := strings.TrimSpace(c.Query("q"))
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 25
	}
	offset := (page - 1) * limit

	// Build query filter
	where := []string{"1=1"}
	args := []interface{}{}
	argPos := 1

	if entity != "" {
		where = append(where, "LOWER(a.entity_name) = LOWER($"+strconv.Itoa(argPos)+")")
		args = append(args, entity)
		argPos++
	}
	if action != "" {
		where = append(where, "LOWER(a.action) = LOWER($"+strconv.Itoa(argPos)+")")
		args = append(args, action)
		argPos++
	}
	if actor != "" {
		where = append(where, "CAST(a.actor_id AS TEXT) = $"+strconv.Itoa(argPos))
		args = append(args, actor)
		argPos++
	}
	if search != "" {
		where = append(where, "(LOWER(a.changes::text) LIKE '%' || LOWER($"+strconv.Itoa(argPos)+") || '%')")
		args = append(args, search)
		argPos++
	}
	if from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err == nil {
			where = append(where, "a.created_at >= $"+strconv.Itoa(argPos))
			args = append(args, t)
			argPos++
		}
	}
	if to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err == nil {
			// tambahkan 1 hari agar inclusive
			where = append(where, "a.created_at < $"+strconv.Itoa(argPos))
			args = append(args, t.Add(24*time.Hour))
			argPos++
		}
	}

	// Final SQL query
	query := `
		SELECT 
			a.id,
			a.entity_name,
			a.entity_id,
			a.action,
			a.actor_id,
			e.name AS actor_name,
			CAST(a.changes AS TEXT),
			a.ip_address,
			a.user_agent,
			a.request_path,
			a.created_at
		FROM audit_logs a
		LEFT JOIN employees e ON e.id = a.actor_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY a.created_at DESC
		LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)

	args = append(args, limit, offset)

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Gagal mengambil data audit logs",
			"detail": err.Error(),
		})
		return
	}
	defer rows.Close()

	var list []AuditLogRecord
	for rows.Next() {
		var r AuditLogRecord
		if err := rows.Scan(
			&r.ID, &r.EntityName, &r.EntityID, &r.Action,
			&r.ActorID, &r.ActorName, &r.Changes,
			&r.IPAddress, &r.UserAgent, &r.RequestPath, &r.CreatedAt,
		); err == nil {
			list = append(list, r)
		}
	}

	// Count total untuk pagination
	var total int
	countArgs := args[:len(args)-2] // hilangkan limit dan offset
	countQuery := `SELECT COUNT(*) FROM audit_logs a WHERE ` + strings.Join(where, " AND ")
	if err := database.Pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		total = len(list)
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": list,
		"pagination": gin.H{
			"page":          page,
			"limit":         limit,
			"total_records": total,
			"total_pages":   (total + limit - 1) / limit,
		},
	})
}

// ============================================================
// 🔹 GET /api/v1/audit-logs/:id
// Detail satu record audit log (untuk inspector)
// ============================================================
func GetAuditLogByID(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var r AuditLogRecord
	err := database.Pool.QueryRow(ctx, `
		SELECT 
			a.id, a.entity_name, a.entity_id, a.action, a.actor_id,
			e.name AS actor_name,
			CAST(a.changes AS TEXT),
			a.ip_address, a.user_agent, a.request_path, a.created_at
		FROM audit_logs a
		LEFT JOIN employees e ON e.id = a.actor_id
		WHERE a.id = $1
	`, id).Scan(
		&r.ID, &r.EntityName, &r.EntityID, &r.Action,
		&r.ActorID, &r.ActorName, &r.Changes,
		&r.IPAddress, &r.UserAgent, &r.RequestPath, &r.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Audit log tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"audit_log": r})
}
