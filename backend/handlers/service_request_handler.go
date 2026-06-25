package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

func generateSRNumber(c *gin.Context) (string, error) {
	var seq int64
	err := database.Pool.QueryRow(c, `SELECT nextval('sr_number_seq')`).Scan(&seq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SR-%d-%04d", time.Now().Year(), seq), nil
}

// ============================================================
// SERVICE CATALOG
// ============================================================

// GetServiceCatalog godoc
// @Summary List service catalog yang aktif
// @Tags Service Catalog
// @Produce json
// @Success 200 {array} models.ServiceCatalog
// @Router /service-catalog [get]
func GetServiceCatalog(c *gin.Context) {
	activeOnly := c.Query("active") != "false"

	query := `
		SELECT sc.id, sc.code, sc.name, sc.category, sc.description,
		       sc.sla_policy_id, sc.approval_required, sc.fulfillment_sla_minutes,
		       sc.is_active, sc.created_by, sc.created_at, sc.updated_at, sc.deleted_at,
		       sp.name AS sla_policy_name,
		       e.name  AS created_by_name
		FROM service_catalog sc
		LEFT JOIN sla_policies sp ON sp.id = sc.sla_policy_id
		LEFT JOIN employees    e  ON e.id  = sc.created_by
		WHERE sc.deleted_at IS NULL
	`
	if activeOnly {
		query += " AND sc.is_active = true"
	}
	query += " ORDER BY sc.category, sc.name"

	rows, err := database.Pool.Query(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ServiceCatalog
	for rows.Next() {
		var sc models.ServiceCatalog
		if err := rows.Scan(
			&sc.ID, &sc.Code, &sc.Name, &sc.Category, &sc.Description,
			&sc.SLAPolicyID, &sc.ApprovalRequired, &sc.FulfillmentSLAMinutes,
			&sc.IsActive, &sc.CreatedBy, &sc.CreatedAt, &sc.UpdatedAt, &sc.DeletedAt,
			&sc.SLAPolicyName, &sc.CreatedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, sc)
	}
	if list == nil {
		list = []models.ServiceCatalog{}
	}
	c.JSON(http.StatusOK, list)
}

// CreateServiceCatalogItem godoc
// @Summary Tambah item baru ke service catalog
// @Tags Service Catalog
// @Accept json
// @Produce json
// @Router /service-catalog [post]
func CreateServiceCatalogItem(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		Code                  string  `json:"code" binding:"required"`
		Name                  string  `json:"name" binding:"required"`
		Category              *string `json:"category"`
		Description           *string `json:"description"`
		SLAPolicyID           *int64  `json:"sla_policy_id"`
		ApprovalRequired      bool    `json:"approval_required"`
		FulfillmentSLAMinutes *int    `json:"fulfillment_sla_minutes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO service_catalog
			(code, name, category, description, sla_policy_id,
			 approval_required, fulfillment_sla_minutes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, req.Code, req.Name, req.Category, req.Description, req.SLAPolicyID,
		req.ApprovalRequired, req.FulfillmentSLAMinutes, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "item catalog berhasil ditambahkan"})
}

// UpdateServiceCatalogItem godoc
// @Summary Update item service catalog
// @Tags Service Catalog
// @Param id path int true "Catalog Item ID"
// @Router /service-catalog/{id} [put]
func UpdateServiceCatalogItem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Name                  *string `json:"name"`
		Category              *string `json:"category"`
		Description           *string `json:"description"`
		SLAPolicyID           *int64  `json:"sla_policy_id"`
		ApprovalRequired      *bool   `json:"approval_required"`
		FulfillmentSLAMinutes *int    `json:"fulfillment_sla_minutes"`
		IsActive              *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE service_catalog SET
			name                   = COALESCE($1, name),
			category               = COALESCE($2, category),
			description            = COALESCE($3, description),
			sla_policy_id          = COALESCE($4, sla_policy_id),
			approval_required      = COALESCE($5, approval_required),
			fulfillment_sla_minutes = COALESCE($6, fulfillment_sla_minutes),
			is_active              = COALESCE($7, is_active),
			updated_at             = now()
		WHERE id = $8 AND deleted_at IS NULL
	`, req.Name, req.Category, req.Description, req.SLAPolicyID,
		req.ApprovalRequired, req.FulfillmentSLAMinutes, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "item catalog berhasil diupdate"})
}

// DeleteServiceCatalogItem godoc
// @Summary Soft-delete item service catalog
// @Tags Service Catalog
// @Param id path int true "Catalog Item ID"
// @Router /service-catalog/{id} [delete]
func DeleteServiceCatalogItem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c,
		`UPDATE service_catalog SET deleted_at = now(), is_active = false WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "item catalog dihapus"})
}

// ============================================================
// SERVICE REQUESTS
// ============================================================

// GetAllServiceRequests godoc
// @Summary List semua service requests
// @Tags Service Request
// @Produce json
// @Param status query string false "Filter by status"
// @Param my query bool false "Hanya request milik sendiri"
// @Success 200 {array} models.ServiceRequestInfo
// @Router /service-requests [get]
func GetAllServiceRequests(c *gin.Context) {
	statusFilter := c.Query("status")
	myOnly := c.Query("my") == "true"
	q := c.Query("q")
	actor := getActorID(c)
	pg := getPagination(c)

	where := "WHERE sr.deleted_at IS NULL"
	args := []interface{}{}
	idx := 1

	if statusFilter != "" {
		where += fmt.Sprintf(" AND sr.status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}
	if myOnly && actor != nil {
		where += fmt.Sprintf(" AND sr.requested_by = $%d", idx)
		args = append(args, *actor)
		idx++
	}
	if q != "" {
		where += fmt.Sprintf(" AND (sr.subject ILIKE $%d OR sr.sr_number ILIKE $%d)", idx, idx)
		args = append(args, "%"+q+"%")
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c,
		"SELECT COUNT(DISTINCT sr.id) FROM service_requests sr JOIN service_catalog sc ON sc.id=sr.service_catalog_id "+where,
		args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT
			sr.id, sr.sr_number,
			sr.service_catalog_id, sc.name AS catalog_name, sc.code AS catalog_code,
			sr.subject, sr.status, sr.priority,
			sr.requested_by, rb.name AS requested_by_name,
			sr.assigned_to, ab.name AS assigned_to_name,
			sr.department_id, d.name AS department_name,
			sr.related_asset_id, a.name AS related_asset_name,
			sc.approval_required,
			sr.fulfilled_at, sr.created_at, sr.updated_at,
			COUNT(aw.id) FILTER (WHERE aw.status = 'pending') AS pending_approvals
		FROM service_requests sr
		JOIN service_catalog sc ON sc.id = sr.service_catalog_id
		JOIN employees rb       ON rb.id = sr.requested_by
		LEFT JOIN employees ab  ON ab.id = sr.assigned_to
		LEFT JOIN departments d ON d.id  = sr.department_id
		LEFT JOIN assets a      ON a.id  = sr.related_asset_id
		LEFT JOIN approval_workflows aw
		       ON aw.entity_type = 'service_request' AND aw.entity_id = sr.id
		%s
		GROUP BY sr.id, sc.name, sc.code, sc.approval_required,
		         rb.name, ab.name, d.name, a.name
		ORDER BY sr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ServiceRequestInfo
	for rows.Next() {
		var sr models.ServiceRequestInfo
		if err := rows.Scan(
			&sr.ID, &sr.SRNumber,
			&sr.ServiceCatalogID, &sr.ServiceCatalogName, &sr.ServiceCatalogCode,
			&sr.Subject, &sr.Status, &sr.Priority,
			&sr.RequestedBy, &sr.RequestedByName,
			&sr.AssignedTo, &sr.AssignedToName,
			&sr.DepartmentID, &sr.DepartmentName,
			&sr.RelatedAssetID, &sr.RelatedAssetName,
			&sr.ApprovalRequired,
			&sr.FulfilledAt, &sr.CreatedAt, &sr.UpdatedAt,
			&sr.PendingApprovals,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, sr)
	}
	if list == nil {
		list = []models.ServiceRequestInfo{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

// GetServiceRequestByID godoc
// @Summary Detail service request beserta approval history
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id} [get]
func GetServiceRequestByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var sr models.ServiceRequestDetail
	err := database.Pool.QueryRow(c, `
		SELECT
			sr.id, sr.sr_number,
			sr.service_catalog_id, sc.name, sc.code,
			sr.subject, sr.status, sr.priority,
			sr.requested_by, rb.name,
			sr.assigned_to, ab.name,
			sr.department_id, d.name,
			sr.related_asset_id, a.name,
			sc.approval_required,
			sr.fulfilled_at, sr.created_at, sr.updated_at,
			0,
			sr.description, sr.notes
		FROM service_requests sr
		JOIN service_catalog sc ON sc.id = sr.service_catalog_id
		JOIN employees rb       ON rb.id = sr.requested_by
		LEFT JOIN employees ab  ON ab.id = sr.assigned_to
		LEFT JOIN departments d ON d.id  = sr.department_id
		LEFT JOIN assets a      ON a.id  = sr.related_asset_id
		WHERE sr.id = $1 AND sr.deleted_at IS NULL
	`, id).Scan(
		&sr.ID, &sr.SRNumber,
		&sr.ServiceCatalogID, &sr.ServiceCatalogName, &sr.ServiceCatalogCode,
		&sr.Subject, &sr.Status, &sr.Priority,
		&sr.RequestedBy, &sr.RequestedByName,
		&sr.AssignedTo, &sr.AssignedToName,
		&sr.DepartmentID, &sr.DepartmentName,
		&sr.RelatedAssetID, &sr.RelatedAssetName,
		&sr.ApprovalRequired,
		&sr.FulfilledAt, &sr.CreatedAt, &sr.UpdatedAt,
		&sr.PendingApprovals,
		&sr.Description, &sr.Notes,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "service request tidak ditemukan"})
		return
	}

	// Ambil approval history
	rows, _ := database.Pool.Query(c, `
		SELECT aw.id, aw.entity_type, aw.entity_id, aw.level,
		       aw.approver_id, aw.status, aw.comment, aw.decided_at, aw.created_at,
		       e.name AS approver_name
		FROM approval_workflows aw
		JOIN employees e ON e.id = aw.approver_id
		WHERE aw.entity_type = 'service_request' AND aw.entity_id = $1
		ORDER BY aw.level, aw.created_at
	`, id)
	defer rows.Close()

	for rows.Next() {
		var aw models.ApprovalWorkflow
		_ = rows.Scan(
			&aw.ID, &aw.EntityType, &aw.EntityID, &aw.Level,
			&aw.ApproverID, &aw.Status, &aw.Comment, &aw.DecidedAt, &aw.CreatedAt,
			&aw.ApproverName,
		)
		sr.Approvals = append(sr.Approvals, aw)
	}
	if sr.Approvals == nil {
		sr.Approvals = []models.ApprovalWorkflow{}
	}

	c.JSON(http.StatusOK, sr)
}

// CreateServiceRequest godoc
// @Summary Buat service request baru
// @Tags Service Request
// @Accept json
// @Produce json
// @Router /service-requests [post]
func CreateServiceRequest(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		ServiceCatalogID int64   `json:"service_catalog_id" binding:"required"`
		Subject          string  `json:"subject" binding:"required"`
		Description      *string `json:"description"`
		Priority         string  `json:"priority"`
		DepartmentID     *int64  `json:"department_id"`
		RelatedAssetID   *int64  `json:"related_asset_id"`
		Notes            *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority == "" {
		req.Priority = "Medium"
	}

	srNumber, err := generateSRNumber(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat SR number"})
		return
	}

	// Tentukan initial status: jika approval_required → pending_approval, else submitted
	var initialStatus string
	var approvalRequired bool
	_ = database.Pool.QueryRow(c,
		`SELECT approval_required FROM service_catalog WHERE id = $1`,
		req.ServiceCatalogID,
	).Scan(&approvalRequired)

	if approvalRequired {
		initialStatus = "pending_approval"
	} else {
		initialStatus = "submitted"
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO service_requests
			(sr_number, service_catalog_id, subject, description,
			 status, priority, requested_by, department_id, related_asset_id, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`, srNumber, req.ServiceCatalogID, req.Subject, req.Description,
		initialStatus, req.Priority, actor, req.DepartmentID, req.RelatedAssetID, req.Notes,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":        id,
		"sr_number": srNumber,
		"status":    initialStatus,
		"message":   "service request berhasil dibuat",
	})
}

// UpdateServiceRequest godoc
// @Summary Update service request (assignee, notes, dll)
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id} [put]
func UpdateServiceRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		AssignedTo     *int64  `json:"assigned_to"`
		Priority       *string `json:"priority"`
		Notes          *string `json:"notes"`
		RelatedAssetID *int64  `json:"related_asset_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE service_requests SET
			assigned_to     = COALESCE($1, assigned_to),
			priority        = COALESCE($2, priority),
			notes           = COALESCE($3, notes),
			related_asset_id = COALESCE($4, related_asset_id),
			updated_at      = now()
		WHERE id = $5 AND deleted_at IS NULL
	`, req.AssignedTo, req.Priority, req.Notes, req.RelatedAssetID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service request berhasil diupdate"})
}

// FulfillServiceRequest godoc
// @Summary Tandai service request selesai dipenuhi
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id}/fulfill [post]
func FulfillServiceRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE service_requests SET
			status       = 'completed',
			fulfilled_at = $1,
			closed_at    = $1,
			updated_at   = now()
		WHERE id = $2
		  AND status IN ('approved','in_fulfillment')
		  AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service request selesai dipenuhi"})
}

// StartFulfillmentServiceRequest godoc
// @Summary Mulai proses pemenuhan SR → in_fulfillment
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id}/start [post]
func StartFulfillmentServiceRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := database.Pool.Exec(c, `
		UPDATE service_requests SET
			status     = 'in_fulfillment',
			updated_at = now()
		WHERE id = $1 AND status = 'approved' AND deleted_at IS NULL
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pemenuhan SR dimulai"})
}

// CancelServiceRequest godoc
// @Summary Batalkan service request
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id}/cancel [post]
func CancelServiceRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := database.Pool.Exec(c, `
		UPDATE service_requests SET
			status     = 'cancelled',
			closed_at  = now(),
			updated_at = now()
		WHERE id = $1
		  AND status NOT IN ('completed','cancelled','rejected')
		  AND deleted_at IS NULL
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service request dibatalkan"})
}

// DeleteServiceRequest godoc
// @Summary Soft-delete service request
// @Tags Service Request
// @Param id path int true "Service Request ID"
// @Router /service-requests/{id} [delete]
func DeleteServiceRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c,
		`UPDATE service_requests SET deleted_at = now() WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service request dihapus"})
}

// ============================================================
// APPROVAL WORKFLOWS (generic — service_request & change_request)
// ============================================================

// AddApprover godoc
// @Summary Tambah approver ke workflow (service_request / change_request)
// @Tags Approval
// @Router /approvals [post]
func AddApprover(c *gin.Context) {
	var req struct {
		EntityType string `json:"entity_type" binding:"required"`
		EntityID   int64  `json:"entity_id" binding:"required"`
		Level      int    `json:"level"`
		ApproverID int64  `json:"approver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Level == 0 {
		req.Level = 1
	}

	_, err := database.Pool.Exec(c, `
		INSERT INTO approval_workflows (entity_type, entity_id, level, approver_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (entity_type, entity_id, level, approver_id) DO NOTHING
	`, req.EntityType, req.EntityID, req.Level, req.ApproverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "approver berhasil ditambahkan"})
}

// SubmitApprovalDecision godoc
// @Summary Submit keputusan approval (approved/rejected/skipped)
// @Tags Approval
// @Router /approvals/decision [post]
func SubmitApprovalDecision(c *gin.Context) {
	actor := getActorID(c)
	now := time.Now()

	var req struct {
		EntityType string  `json:"entity_type" binding:"required"`
		EntityID   int64   `json:"entity_id" binding:"required"`
		Decision   string  `json:"decision" binding:"required"`
		Comment    *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE approval_workflows SET
			status     = $1,
			comment    = $2,
			decided_at = $3
		WHERE entity_type = $4 AND entity_id = $5 AND approver_id = $6
		  AND status = 'pending'
	`, req.Decision, req.Comment, now, req.EntityType, req.EntityID, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Jika service_request dan semua level sudah approved → update SR status
	if req.EntityType == "service_request" && req.Decision == "approved" {
		var pendingCount int
		_ = database.Pool.QueryRow(c, `
			SELECT COUNT(*) FROM approval_workflows
			WHERE entity_type = 'service_request' AND entity_id = $1 AND status = 'pending'
		`, req.EntityID).Scan(&pendingCount)

		if pendingCount == 0 {
			_, _ = database.Pool.Exec(c, `
				UPDATE service_requests SET status = 'approved', updated_at = now()
				WHERE id = $1 AND status = 'pending_approval'
			`, req.EntityID)
		}
	}

	// Jika ada satu reject → SR/CR langsung rejected
	if req.Decision == "rejected" {
		switch req.EntityType {
		case "service_request":
			_, _ = database.Pool.Exec(c, `
				UPDATE service_requests SET status = 'rejected', updated_at = now()
				WHERE id = $1
			`, req.EntityID)
		case "change_request":
			_, _ = database.Pool.Exec(c, `
				UPDATE change_requests SET status = 'rejected', updated_at = now()
				WHERE id = $1
			`, req.EntityID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "keputusan approval berhasil disimpan"})
}

// GetApprovalsByEntity godoc
// @Summary List approval history untuk suatu entitas
// @Tags Approval
// @Param entity_type query string true "service_request atau change_request"
// @Param entity_id query int true "ID entitas"
// @Router /approvals [get]
func GetApprovalsByEntity(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityID, _ := strconv.ParseInt(c.Query("entity_id"), 10, 64)

	if entityType == "" || entityID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type dan entity_id wajib diisi"})
		return
	}

	rows, err := database.Pool.Query(c, `
		SELECT aw.id, aw.entity_type, aw.entity_id, aw.level,
		       aw.approver_id, aw.status, aw.comment, aw.decided_at, aw.created_at,
		       e.name AS approver_name
		FROM approval_workflows aw
		JOIN employees e ON e.id = aw.approver_id
		WHERE aw.entity_type = $1 AND aw.entity_id = $2
		ORDER BY aw.level, aw.created_at
	`, entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ApprovalWorkflow
	for rows.Next() {
		var aw models.ApprovalWorkflow
		if err := rows.Scan(
			&aw.ID, &aw.EntityType, &aw.EntityID, &aw.Level,
			&aw.ApproverID, &aw.Status, &aw.Comment, &aw.DecidedAt, &aw.CreatedAt,
			&aw.ApproverName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, aw)
	}
	if list == nil {
		list = []models.ApprovalWorkflow{}
	}
	c.JSON(http.StatusOK, list)
}
