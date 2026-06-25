package handlers

import (
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// COMPLIANCE FRAMEWORKS
// ============================================================

func GetAllFrameworks(c *gin.Context) {
	activeOnly := c.Query("active_only") == "true"

	query := `
		SELECT f.id, f.code, f.name, f.version, f.description, f.is_active, f.created_at, f.updated_at,
		       (SELECT count(*) FROM compliance_controls cc WHERE cc.framework_id = f.id AND cc.is_active) AS control_count
		FROM compliance_frameworks f
		WHERE 1=1
	`
	if activeOnly {
		query += " AND f.is_active = true"
	}
	query += " ORDER BY f.code"

	rows, err := database.Pool.Query(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ComplianceFramework
	for rows.Next() {
		var f models.ComplianceFramework
		if err := rows.Scan(&f.ID, &f.Code, &f.Name, &f.Version, &f.Description,
			&f.IsActive, &f.CreatedAt, &f.UpdatedAt, &f.ControlCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, f)
	}
	if list == nil {
		list = []models.ComplianceFramework{}
	}
	c.JSON(http.StatusOK, list)
}

func GetFrameworkByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var f models.ComplianceFramework
	err := database.Pool.QueryRow(c, `
		SELECT id, code, name, version, description, is_active, created_at, updated_at
		FROM compliance_frameworks WHERE id = $1
	`, id).Scan(&f.ID, &f.Code, &f.Name, &f.Version, &f.Description,
		&f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "framework tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, f)
}

func CreateFramework(c *gin.Context) {
	var req struct {
		Code        string  `json:"code" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Version     *string `json:"version"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO compliance_frameworks (code, name, version, description)
		VALUES ($1,$2,$3,$4) RETURNING id
	`, req.Code, req.Name, req.Version, req.Description).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "framework berhasil dibuat"})
}

func UpdateFramework(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name        *string `json:"name"`
		Version     *string `json:"version"`
		Description *string `json:"description"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE compliance_frameworks SET
			name        = COALESCE($1, name),
			version     = COALESCE($2, version),
			description = COALESCE($3, description),
			is_active   = COALESCE($4, is_active),
			updated_at  = now()
		WHERE id = $5
	`, req.Name, req.Version, req.Description, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "framework berhasil diupdate"})
}

func DeleteFramework(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM compliance_frameworks WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "framework berhasil dihapus"})
}

// ============================================================
// COMPLIANCE CONTROLS
// ============================================================

func GetControlsByFramework(c *gin.Context) {
	frameworkID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.Pool.Query(c, `
		SELECT cc.id, cc.framework_id, cc.control_code, cc.name, cc.description,
		       cc.category, cc.severity, cc.is_active, cc.created_at,
		       f.name AS framework_name,
		       (SELECT count(*) FROM compliance_evidence ce WHERE ce.control_id = cc.id AND ce.status = 'accepted') AS evidence_count
		FROM compliance_controls cc
		JOIN compliance_frameworks f ON f.id = cc.framework_id
		WHERE cc.framework_id = $1
		ORDER BY cc.control_code
	`, frameworkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ComplianceControl
	for rows.Next() {
		var ct models.ComplianceControl
		if err := rows.Scan(&ct.ID, &ct.FrameworkID, &ct.ControlCode, &ct.Name, &ct.Description,
			&ct.Category, &ct.Severity, &ct.IsActive, &ct.CreatedAt,
			&ct.FrameworkName, &ct.EvidenceCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, ct)
	}
	if list == nil {
		list = []models.ComplianceControl{}
	}
	c.JSON(http.StatusOK, list)
}

func CreateControl(c *gin.Context) {
	frameworkID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		ControlCode string  `json:"control_code" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
		Category    *string `json:"category"`
		Severity    *string `json:"severity"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO compliance_controls (framework_id, control_code, name, description, category, severity)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id
	`, frameworkID, req.ControlCode, req.Name, req.Description, req.Category, req.Severity).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "kontrol berhasil dibuat"})
}

func UpdateControl(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Category    *string `json:"category"`
		Severity    *string `json:"severity"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE compliance_controls SET
			name        = COALESCE($1, name),
			description = COALESCE($2, description),
			category    = COALESCE($3, category),
			severity    = COALESCE($4, severity),
			is_active   = COALESCE($5, is_active)
		WHERE id = $6
	`, req.Name, req.Description, req.Category, req.Severity, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "kontrol berhasil diupdate"})
}

func DeleteControl(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM compliance_controls WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "kontrol berhasil dihapus"})
}

// ============================================================
// COMPLIANCE EVIDENCE
// ============================================================

func GetEvidenceByControl(c *gin.Context) {
	controlID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	statusFilter := c.Query("status")

	query := `
		SELECT ce.id, ce.control_id, ce.entity_type, ce.entity_id,
		       ce.evidence_type, ce.title, ce.description, ce.file_url,
		       ce.status, ce.reviewed_by, ce.reviewed_at, ce.expires_at,
		       ce.submitted_by, ce.created_at, ce.updated_at,
		       cc.control_code, cc.name AS control_name,
		       rev.name AS reviewed_by_name,
		       sub.name AS submitted_by_name
		FROM compliance_evidence ce
		JOIN compliance_controls cc ON cc.id = ce.control_id
		LEFT JOIN employees rev ON rev.id = ce.reviewed_by
		LEFT JOIN employees sub ON sub.id = ce.submitted_by
		WHERE ce.control_id = $1
	`
	args := []interface{}{controlID}
	if statusFilter != "" {
		query += " AND ce.status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY ce.created_at DESC"

	rows, err := database.Pool.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ComplianceEvidence
	for rows.Next() {
		var ev models.ComplianceEvidence
		if err := rows.Scan(
			&ev.ID, &ev.ControlID, &ev.EntityType, &ev.EntityID,
			&ev.EvidenceType, &ev.Title, &ev.Description, &ev.FileURL,
			&ev.Status, &ev.ReviewedBy, &ev.ReviewedAt, &ev.ExpiresAt,
			&ev.SubmittedBy, &ev.CreatedAt, &ev.UpdatedAt,
			&ev.ControlCode, &ev.ControlName,
			&ev.ReviewedByName, &ev.SubmittedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, ev)
	}
	if list == nil {
		list = []models.ComplianceEvidence{}
	}
	c.JSON(http.StatusOK, list)
}

func AddEvidence(c *gin.Context) {
	controlID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		EntityType   string  `json:"entity_type" binding:"required"`
		EntityID     int64   `json:"entity_id" binding:"required"`
		EvidenceType string  `json:"evidence_type" binding:"required"`
		Title        string  `json:"title" binding:"required"`
		Description  *string `json:"description"`
		FileURL      *string `json:"file_url"`
		ExpiresAt    *string `json:"expires_at"` // RFC3339
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO compliance_evidence
			(control_id, entity_type, entity_id, evidence_type, title, description, file_url, expires_at, submitted_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id
	`, controlID, req.EntityType, req.EntityID, req.EvidenceType,
		req.Title, req.Description, req.FileURL, req.ExpiresAt, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "evidence berhasil ditambahkan"})
}

func ReviewEvidence(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		Status string  `json:"status" binding:"required"` // accepted|rejected|expired
		Notes  *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE compliance_evidence SET
			status      = $1,
			reviewed_by = $2,
			reviewed_at = now(),
			description = CASE WHEN $3::TEXT IS NOT NULL THEN $3 ELSE description END,
			updated_at  = now()
		WHERE id = $4
	`, req.Status, actor, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "evidence berhasil direview"})
}

func DeleteEvidence(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM compliance_evidence WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "evidence berhasil dihapus"})
}

// ============================================================
// DISPOSAL COMPLIANCE VIEW
// ============================================================

func GetDisposalCompliance(c *gin.Context) {
	statusFilter := c.Query("status") // compliant|data_wipe_pending|env_non_compliant|missing_record

	query := `
		SELECT asset_id, asset_name, asset_tag, lifecycle_stage,
		       disposal_record_id, disposal_method, data_wipe_completed,
		       environmental_compliant, certificate_number, date_disposed,
		       authorized_by, executed_by, compliance_status
		FROM v_asset_disposal_compliance
		WHERE 1=1
	`
	args := []interface{}{}
	if statusFilter != "" {
		query += " AND compliance_status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY compliance_status, asset_name"

	rows, err := database.Pool.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.AssetDisposalCompliance
	for rows.Next() {
		var d models.AssetDisposalCompliance
		if err := rows.Scan(
			&d.AssetID, &d.AssetName, &d.AssetTag, &d.LifecycleStage,
			&d.DisposalRecordID, &d.DisposalMethod, &d.DataWipeCompleted,
			&d.EnvironmentalCompliant, &d.CertificateNumber, &d.DateDisposed,
			&d.AuthorizedBy, &d.ExecutedBy, &d.ComplianceStatus,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, d)
	}
	if list == nil {
		list = []models.AssetDisposalCompliance{}
	}
	c.JSON(http.StatusOK, list)
}

// GetComplianceSummaryByFramework menghitung % evidence accepted per framework.
func GetComplianceSummaryByFramework(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT
			f.id, f.code, f.name,
			COUNT(DISTINCT cc.id) AS total_controls,
			COUNT(DISTINCT CASE WHEN ce.status = 'accepted' THEN cc.id END) AS covered_controls,
			ROUND(
				COUNT(DISTINCT CASE WHEN ce.status = 'accepted' THEN cc.id END)::NUMERIC /
				NULLIF(COUNT(DISTINCT cc.id), 0) * 100, 2
			) AS coverage_pct
		FROM compliance_frameworks f
		LEFT JOIN compliance_controls cc ON cc.framework_id = f.id AND cc.is_active
		LEFT JOIN compliance_evidence ce ON ce.control_id = cc.id
		WHERE f.is_active
		GROUP BY f.id, f.code, f.name
		ORDER BY f.code
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type FrameworkCoverage struct {
		ID              int64    `json:"id"`
		Code            string   `json:"code"`
		Name            string   `json:"name"`
		TotalControls   int      `json:"total_controls"`
		CoveredControls int      `json:"covered_controls"`
		CoveragePct     *float64 `json:"coverage_pct"`
	}

	var list []FrameworkCoverage
	for rows.Next() {
		var fc FrameworkCoverage
		if err := rows.Scan(&fc.ID, &fc.Code, &fc.Name, &fc.TotalControls, &fc.CoveredControls, &fc.CoveragePct); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, fc)
	}
	if list == nil {
		list = []FrameworkCoverage{}
	}
	c.JSON(http.StatusOK, list)
}
