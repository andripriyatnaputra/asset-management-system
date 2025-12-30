package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/helpers"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ============================================================
// ENUM VALIDATION (IMPACT, URGENCY, PRIORITY, CATEGORY, SERVICE)
// ============================================================
var validImpact = map[string]bool{"Low": true, "Medium": true, "High": true}
var validUrgency = map[string]bool{"Low": true, "Medium": true, "High": true}
var validPriority = map[string]bool{"Low": true, "Medium": true, "High": true}

var validCategories = map[string]bool{
	"INCIDENT":    true,
	"REQUEST":     true,
	"MAINTENANCE": true,
}

var validServices = map[string]bool{
	"NETWORK":  true,
	"HARDWARE": true,
	"SOFTWARE": true,
}

// ============================================================
// GET ALL SLA POLICIES
// ============================================================
func GetAllSLAPolicies(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT id,name,category_code,service_code,
			   impact,urgency,resulting_priority,
			   response_minutes,resolve_minutes,
			   is_active,compliance_score,
			   legacy_compliance_score,
			   created_by,updated_by,
			   created_at,updated_at
		  FROM sla_policies
		 WHERE deleted_at IS NULL
		 ORDER BY impact, urgency`)
	if err != nil {
		c.JSON(500, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID                int64      `json:"id"`
		Name              string     `json:"name"`
		CategoryCode      *string    `json:"category_code"`
		ServiceCode       *string    `json:"service_code"`
		Impact            string     `json:"impact"`
		Urgency           string     `json:"urgency"`
		ResultingPriority string     `json:"resulting_priority"`
		ResponseMinutes   int        `json:"response_minutes"`
		ResolveMinutes    int        `json:"resolve_minutes"`
		IsActive          bool       `json:"is_active"`
		Score             *float64   `json:"compliance_score"`
		LegacyScore       *float64   `json:"legacy_compliance_score"`
		CreatedBy         *int64     `json:"created_by"`
		UpdatedBy         *int64     `json:"updated_by"`
		CreatedAt         *time.Time `json:"created_at"`
		UpdatedAt         *time.Time `json:"updated_at"`
	}

	var list []Row

	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.ID, &r.Name, &r.CategoryCode, &r.ServiceCode,
			&r.Impact, &r.Urgency, &r.ResultingPriority,
			&r.ResponseMinutes, &r.ResolveMinutes,
			&r.IsActive, &r.Score, &r.LegacyScore,
			&r.CreatedBy, &r.UpdatedBy, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			c.JSON(500, gin.H{"error": "scan failed"})
			return
		}
		list = append(list, r)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, gin.H{"error": "iteration error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sla_policies": list})
}

// ============================================================
// CREATE SLA POLICY
// ============================================================
func CreateSLAPolicy(c *gin.Context) {
	var input struct {
		Name              string  `json:"name" binding:"required"`
		CategoryCode      *string `json:"category_code"`
		ServiceCode       *string `json:"service_code"`
		Impact            string  `json:"impact" binding:"required"`
		Urgency           string  `json:"urgency" binding:"required"`
		ResultingPriority string  `json:"resulting_priority" binding:"required"`
		ResponseMinutes   int     `json:"response_minutes" binding:"required"`
		ResolveMinutes    int     `json:"resolve_minutes" binding:"required"`
		IsActive          *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// ENUM VALIDATION
	if !validImpact[input.Impact] {
		c.JSON(400, gin.H{"error": "invalid impact value"})
		return
	}
	if !validUrgency[input.Urgency] {
		c.JSON(400, gin.H{"error": "invalid urgency value"})
		return
	}
	if !validPriority[input.ResultingPriority] {
		c.JSON(400, gin.H{"error": "invalid resulting_priority value"})
		return
	}
	if input.CategoryCode != nil && !validCategories[*input.CategoryCode] {
		c.JSON(400, gin.H{"error": "invalid category_code value"})
		return
	}
	if input.ServiceCode != nil && !validServices[*input.ServiceCode] {
		c.JSON(400, gin.H{"error": "invalid service_code value"})
		return
	}

	// Default is_active
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	// Compute score
	score := helpers.ComputeDynamicScore(
		input.ResponseMinutes,
		input.ResolveMinutes,
		input.Impact,
		input.Urgency,
	)
	legacyScore := score

	// Audit
	uidRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	createdBy := uidRaw.(int64)

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO sla_policies
		    (name,category_code,service_code,impact,urgency,
		 	 resulting_priority,response_minutes,resolve_minutes,
			 is_active,compliance_score,legacy_compliance_score,
			 created_by,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW())
		RETURNING id`,
		input.Name, input.CategoryCode, input.ServiceCode,
		input.Impact, input.Urgency, input.ResultingPriority,
		input.ResponseMinutes, input.ResolveMinutes, isActive,
		score, legacyScore, createdBy,
	).Scan(&id)

	if err != nil {
		c.JSON(500, gin.H{"error": "insert failed"})
		return
	}

	middleware.LogAction(
		c,
		"sla_policies",
		id,
		"CREATE",
		map[string]any{
			"name":    input.Name,
			"impact":  input.Impact,
			"urgency": input.Urgency,
		},
	)

	c.JSON(201, gin.H{
		"message":                 "SLA Policy created",
		"id":                      id,
		"compliance_score":        score,
		"legacy_compliance_score": legacyScore,
		"created_by":              createdBy,
		"created_at_timestamp":    time.Now(),
	})
}

// ============================================================
// UPDATE SLA POLICY
// ============================================================
func UpdateSLAPolicy(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		Name              string  `json:"name" binding:"required"`
		CategoryCode      *string `json:"category_code"`
		ServiceCode       *string `json:"service_code"`
		Impact            string  `json:"impact" binding:"required"`
		Urgency           string  `json:"urgency" binding:"required"`
		ResultingPriority string  `json:"resulting_priority" binding:"required"`
		ResponseMinutes   int     `json:"response_minutes" binding:"required"`
		ResolveMinutes    int     `json:"resolve_minutes" binding:"required"`
		IsActive          *bool   `json:"is_active"`
		ResetLegacy       bool    `json:"reset_legacy"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// ENUM VALIDATION
	if !validImpact[input.Impact] {
		c.JSON(400, gin.H{"error": "invalid impact"})
		return
	}
	if !validUrgency[input.Urgency] {
		c.JSON(400, gin.H{"error": "invalid urgency"})
		return
	}
	if !validPriority[input.ResultingPriority] {
		c.JSON(400, gin.H{"error": "invalid resulting_priority"})
		return
	}
	if input.CategoryCode != nil && !validCategories[*input.CategoryCode] {
		c.JSON(400, gin.H{"error": "invalid category_code"})
		return
	}
	if input.ServiceCode != nil && !validServices[*input.ServiceCode] {
		c.JSON(400, gin.H{"error": "invalid service_code"})
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	uidRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	updatedBy := uidRaw.(int64)

	score := helpers.ComputeDynamicScore(
		input.ResponseMinutes,
		input.ResolveMinutes,
		input.Impact,
		input.Urgency,
	)

	query := `
		UPDATE sla_policies
		   SET name=$1,category_code=$2,service_code=$3,
		       impact=$4,urgency=$5,resulting_priority=$6,
			   response_minutes=$7,resolve_minutes=$8,
			   is_active=$9,compliance_score=$10,
			   updated_by=$11,updated_at=NOW()`
	args := []interface{}{
		input.Name, input.CategoryCode, input.ServiceCode,
		input.Impact, input.Urgency, input.ResultingPriority,
		input.ResponseMinutes, input.ResolveMinutes,
		isActive, score, updatedBy,
	}

	if input.ResetLegacy {
		query += `, legacy_compliance_score=$13`
	}

	query += ` WHERE id=$12 AND deleted_at IS NULL`

	args = append(args, id)

	if input.ResetLegacy {
		args = append(args, score) // arg ke-13
	}

	cmdTag, err := database.Pool.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": "update failed"})
		return
	}
	if cmdTag.RowsAffected() == 0 {
		c.JSON(404, gin.H{"error": "SLA policy not found"})
		return
	}

	middleware.LogAction(
		c,
		"sla_policies",
		mustAtoi64(id),
		"UPDATE",
		map[string]any{
			"impact":       input.Impact,
			"urgency":      input.Urgency,
			"priority":     input.ResultingPriority,
			"reset_legacy": input.ResetLegacy,
		},
	)

	c.JSON(200, gin.H{
		"message":          "SLA Policy updated",
		"updated_by":       updatedBy,
		"compliance_score": score,
		"reset_legacy":     input.ResetLegacy,
	})
}

// ============================================================
// DELETE SLA POLICY (SOFT DELETE)
// ============================================================
func DeleteSLAPolicy(c *gin.Context) {
	id := c.Param("id")

	cmdTag, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE sla_policies SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)

	if err != nil {
		c.JSON(500, gin.H{"error": "delete failed"})
		return
	}
	if cmdTag.RowsAffected() == 0 {
		c.JSON(404, gin.H{"error": "SLA policy not found"})
		return
	}

	c.JSON(200, gin.H{"message": "SLA Policy deleted (soft)"})
}

// ============================================================
// GET SLA POLICY BY ID
// ============================================================
func GetSLAPolicyByID(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT sp.id,sp.name,sp.category_code,sp.service_code,
		       sp.impact,sp.urgency,sp.resulting_priority,
			   sp.response_minutes,sp.resolve_minutes,
			   sp.is_active,sp.compliance_score,
			   sp.legacy_compliance_score,
			   sp.created_by,sp.updated_by,
			   c1.name AS created_by_name,
			   c2.name AS updated_by_name,
			   sp.created_at,sp.updated_at
		  FROM sla_policies sp
		  LEFT JOIN employees c1 ON sp.created_by=c1.id
		  LEFT JOIN employees c2 ON sp.updated_by=c2.id
		 WHERE sp.id=$1 AND sp.deleted_at IS NULL`

	var row struct {
		ID                int64      `json:"id"`
		Name              string     `json:"name"`
		CategoryCode      *string    `json:"category_code"`
		ServiceCode       *string    `json:"service_code"`
		Impact            string     `json:"impact"`
		Urgency           string     `json:"urgency"`
		ResultingPriority string     `json:"resulting_priority"`
		ResponseMinutes   int        `json:"response_minutes"`
		ResolveMinutes    int        `json:"resolve_minutes"`
		IsActive          bool       `json:"is_active"`
		Score             *float64   `json:"compliance_score"`
		LegacyScore       *float64   `json:"legacy_compliance_score"`
		CreatedBy         *int64     `json:"created_by"`
		UpdatedBy         *int64     `json:"updated_by"`
		CreatedByName     *string    `json:"created_by_name"`
		UpdatedByName     *string    `json:"updated_by_name"`
		CreatedAt         *time.Time `json:"created_at"`
		UpdatedAt         *time.Time `json:"updated_at"`
	}

	err := database.Pool.QueryRow(c.Request.Context(), query, id).Scan(
		&row.ID, &row.Name, &row.CategoryCode, &row.ServiceCode,
		&row.Impact, &row.Urgency, &row.ResultingPriority,
		&row.ResponseMinutes, &row.ResolveMinutes, &row.IsActive,
		&row.Score, &row.LegacyScore,
		&row.CreatedBy, &row.UpdatedBy,
		&row.CreatedByName, &row.UpdatedByName,
		&row.CreatedAt, &row.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(404, gin.H{"error": "SLA policy not found"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "query failed"})
		return
	}

	c.JSON(200, gin.H{"data": row})
}

// ============================================================
// SLA COMPLIANCE REPORT (DASHBOARD)
// ============================================================
func GetSLAComplianceReport(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT DATE_TRUNC('month',t.created_at) AS month,
		       COUNT(*) FILTER(WHERE t.sla_breached_at IS NOT NULL) AS breached,
		       COUNT(*) FILTER(WHERE t.status='Closed') AS closed,
		       COUNT(*) AS total
		  FROM tickets t
		 WHERE t.deleted_at IS NULL
		 GROUP BY 1 ORDER BY 1`)
	if err != nil {
		c.JSON(500, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Row struct {
		Month    time.Time `json:"month"`
		Breached int       `json:"breached"`
		Closed   int       `json:"closed"`
		Total    int       `json:"total"`
	}

	var list []Row

	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Month, &r.Breached, &r.Closed, &r.Total); err != nil {
			c.JSON(500, gin.H{"error": "scan failed"})
			return
		}
		list = append(list, r)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, gin.H{"error": "iteration error"})
		return
	}

	c.JSON(200, gin.H{"sla_compliance": list})
}
