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

// generateCRNumber membuat nomor CR unik format CR-YYYY-NNNN
func generateCRNumber(c *gin.Context) (string, error) {
	var seq int64
	err := database.Pool.QueryRow(c, `SELECT nextval('cr_number_seq')`).Scan(&seq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("CR-%d-%04d", time.Now().Year(), seq), nil
}

// ============================================================
// CHANGE REQUEST CRUD
// ============================================================

// GetAllChangeRequests godoc
// @Summary List semua change requests
// @Tags Change Management
// @Produce json
// @Param status query string false "Filter by status"
// @Param type query string false "Filter by type (standard/normal/emergency)"
// @Success 200 {array} models.ChangeRequestInfo
// @Router /change-requests [get]
func GetAllChangeRequests(c *gin.Context) {
	statusFilter := c.Query("status")
	typeFilter := c.Query("type")
	q := c.Query("q")
	pg := getPagination(c)

	where := "WHERE cr.deleted_at IS NULL"
	args := []interface{}{}
	idx := 1

	if statusFilter != "" {
		where += fmt.Sprintf(" AND cr.status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}
	if typeFilter != "" {
		where += fmt.Sprintf(" AND cr.type = $%d", idx)
		args = append(args, typeFilter)
		idx++
	}
	if q != "" {
		where += fmt.Sprintf(" AND (cr.title ILIKE $%d OR cr.cr_number ILIKE $%d)", idx, idx)
		args = append(args, "%"+q+"%")
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(DISTINCT cr.id) FROM change_requests cr "+where, args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT
			cr.id, cr.cr_number, cr.title, cr.type, cr.status, cr.risk_level,
			cr.cab_required, cr.change_window_start, cr.change_window_end,
			cr.related_asset_id, a.name  AS related_asset_name,
			cr.related_ticket_id,
			cr.created_by,  cb.name AS created_by_name,
			cr.approved_by, ab.name AS approved_by_name,
			cr.submitted_at, cr.approved_at, cr.implemented_at,
			cr.created_at, cr.updated_at,
			COUNT(ct.id)                                          AS task_total,
			COUNT(ct.id) FILTER (WHERE ct.status = 'done')       AS task_done,
			COUNT(ca.id)                                          AS approval_total,
			COUNT(ca.id) FILTER (WHERE ca.decision = 'approved') AS approval_approved
		FROM change_requests cr
		LEFT JOIN assets    a  ON a.id  = cr.related_asset_id
		LEFT JOIN employees cb ON cb.id = cr.created_by
		LEFT JOIN employees ab ON ab.id = cr.approved_by
		LEFT JOIN change_tasks     ct ON ct.change_id = cr.id
		LEFT JOIN change_approvals ca ON ca.change_id = cr.id
		%s
		GROUP BY cr.id, a.name, cb.name, ab.name
		ORDER BY cr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ChangeRequestInfo
	for rows.Next() {
		var cr models.ChangeRequestInfo
		if err := rows.Scan(
			&cr.ID, &cr.CRNumber, &cr.Title, &cr.Type, &cr.Status, &cr.RiskLevel,
			&cr.CABRequired, &cr.ChangeWindowStart, &cr.ChangeWindowEnd,
			&cr.RelatedAssetID, &cr.RelatedAssetName,
			&cr.RelatedTicketID,
			&cr.CreatedBy, &cr.CreatedByName,
			&cr.ApprovedBy, &cr.ApprovedByName,
			&cr.SubmittedAt, &cr.ApprovedAt, &cr.ImplementedAt,
			&cr.CreatedAt, &cr.UpdatedAt,
			&cr.TaskTotal, &cr.TaskDone,
			&cr.ApprovalTotal, &cr.ApprovalApproved,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, cr)
	}
	if list == nil {
		list = []models.ChangeRequestInfo{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

// GetChangeRequestByID godoc
// @Summary Detail change request beserta approvals & tasks
// @Tags Change Management
// @Produce json
// @Param id path int true "Change Request ID"
// @Success 200 {object} models.ChangeRequestDetail
// @Router /change-requests/{id} [get]
func GetChangeRequestByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var cr models.ChangeRequestDetail
	err := database.Pool.QueryRow(c, `
		SELECT
			cr.id, cr.cr_number, cr.title, cr.type, cr.status, cr.risk_level,
			cr.cab_required, cr.change_window_start, cr.change_window_end,
			cr.related_asset_id, a.name  AS related_asset_name,
			cr.related_ticket_id,
			cr.created_by,  cb.name AS created_by_name,
			cr.approved_by, ab.name AS approved_by_name,
			cr.submitted_at, cr.approved_at, cr.implemented_at,
			cr.created_at, cr.updated_at,
			0, 0, 0, 0,
			cr.description, cr.impact_assessment, cr.rollback_plan
		FROM change_requests cr
		LEFT JOIN assets    a  ON a.id  = cr.related_asset_id
		LEFT JOIN employees cb ON cb.id = cr.created_by
		LEFT JOIN employees ab ON ab.id = cr.approved_by
		WHERE cr.id = $1 AND cr.deleted_at IS NULL
	`, id).Scan(
		&cr.ID, &cr.CRNumber, &cr.Title, &cr.Type, &cr.Status, &cr.RiskLevel,
		&cr.CABRequired, &cr.ChangeWindowStart, &cr.ChangeWindowEnd,
		&cr.RelatedAssetID, &cr.RelatedAssetName,
		&cr.RelatedTicketID,
		&cr.CreatedBy, &cr.CreatedByName,
		&cr.ApprovedBy, &cr.ApprovedByName,
		&cr.SubmittedAt, &cr.ApprovedAt, &cr.ImplementedAt,
		&cr.CreatedAt, &cr.UpdatedAt,
		&cr.TaskTotal, &cr.TaskDone, &cr.ApprovalTotal, &cr.ApprovalApproved,
		&cr.Description, &cr.ImpactAssessment, &cr.RollbackPlan,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "change request tidak ditemukan"})
		return
	}

	// Ambil approvals
	aRows, _ := database.Pool.Query(c, `
		SELECT ca.id, ca.change_id, ca.approver_id, ca.decision,
		       ca.comment, ca.decided_at, ca.created_at, e.name
		FROM change_approvals ca
		JOIN employees e ON e.id = ca.approver_id
		WHERE ca.change_id = $1
		ORDER BY ca.created_at
	`, id)
	defer aRows.Close()
	for aRows.Next() {
		var a models.ChangeApproval
		_ = aRows.Scan(&a.ID, &a.ChangeID, &a.ApproverID, &a.Decision,
			&a.Comment, &a.DecidedAt, &a.CreatedAt, &a.ApproverName)
		cr.Approvals = append(cr.Approvals, a)
	}
	if cr.Approvals == nil {
		cr.Approvals = []models.ChangeApproval{}
	}

	// Ambil tasks
	tRows, _ := database.Pool.Query(c, `
		SELECT ct.id, ct.change_id, ct.title, ct.description,
		       ct.status, ct.assigned_to, ct.seq_order,
		       ct.completed_at, ct.created_at, ct.updated_at,
		       e.name
		FROM change_tasks ct
		LEFT JOIN employees e ON e.id = ct.assigned_to
		WHERE ct.change_id = $1
		ORDER BY ct.seq_order, ct.id
	`, id)
	defer tRows.Close()
	for tRows.Next() {
		var t models.ChangeTask
		_ = tRows.Scan(&t.ID, &t.ChangeID, &t.Title, &t.Description,
			&t.Status, &t.AssignedTo, &t.SeqOrder,
			&t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
			&t.AssigneeName)
		cr.Tasks = append(cr.Tasks, t)
	}
	if cr.Tasks == nil {
		cr.Tasks = []models.ChangeTask{}
	}

	c.JSON(http.StatusOK, cr)
}

// CreateChangeRequest godoc
// @Summary Buat change request baru
// @Tags Change Management
// @Accept json
// @Produce json
// @Router /change-requests [post]
func CreateChangeRequest(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		Title             string     `json:"title" binding:"required"`
		Description       *string    `json:"description"`
		Type              string     `json:"type"`
		RiskLevel         string     `json:"risk_level"`
		ImpactAssessment  *string    `json:"impact_assessment"`
		RollbackPlan      *string    `json:"rollback_plan"`
		ChangeWindowStart *time.Time `json:"change_window_start"`
		ChangeWindowEnd   *time.Time `json:"change_window_end"`
		CABRequired       bool       `json:"cab_required"`
		RelatedAssetID    *int64     `json:"related_asset_id"`
		RelatedTicketID   *int64     `json:"related_ticket_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = "normal"
	}
	if req.RiskLevel == "" {
		req.RiskLevel = "medium"
	}

	crNumber, err := generateCRNumber(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat CR number"})
		return
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO change_requests
			(cr_number, title, description, type, risk_level,
			 impact_assessment, rollback_plan,
			 change_window_start, change_window_end, cab_required,
			 related_asset_id, related_ticket_id, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id
	`, crNumber, req.Title, req.Description, req.Type, req.RiskLevel,
		req.ImpactAssessment, req.RollbackPlan,
		req.ChangeWindowStart, req.ChangeWindowEnd, req.CABRequired,
		req.RelatedAssetID, req.RelatedTicketID, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "cr_number": crNumber, "message": "change request berhasil dibuat"})
}

// UpdateChangeRequest godoc
// @Summary Update detail change request
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id} [put]
func UpdateChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Title             *string    `json:"title"`
		Description       *string    `json:"description"`
		Type              *string    `json:"type"`
		RiskLevel         *string    `json:"risk_level"`
		ImpactAssessment  *string    `json:"impact_assessment"`
		RollbackPlan      *string    `json:"rollback_plan"`
		ChangeWindowStart *time.Time `json:"change_window_start"`
		ChangeWindowEnd   *time.Time `json:"change_window_end"`
		CABRequired       *bool      `json:"cab_required"`
		RelatedAssetID    *int64     `json:"related_asset_id"`
		RelatedTicketID   *int64     `json:"related_ticket_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			title              = COALESCE($1,  title),
			description        = COALESCE($2,  description),
			type               = COALESCE($3,  type),
			risk_level         = COALESCE($4,  risk_level),
			impact_assessment  = COALESCE($5,  impact_assessment),
			rollback_plan      = COALESCE($6,  rollback_plan),
			change_window_start = COALESCE($7, change_window_start),
			change_window_end  = COALESCE($8,  change_window_end),
			cab_required       = COALESCE($9,  cab_required),
			related_asset_id   = COALESCE($10, related_asset_id),
			related_ticket_id  = COALESCE($11, related_ticket_id),
			updated_at         = now()
		WHERE id = $12 AND deleted_at IS NULL
	`, req.Title, req.Description, req.Type, req.RiskLevel,
		req.ImpactAssessment, req.RollbackPlan,
		req.ChangeWindowStart, req.ChangeWindowEnd,
		req.CABRequired, req.RelatedAssetID, req.RelatedTicketID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request berhasil diupdate"})
}

// DeleteChangeRequest godoc
// @Summary Soft-delete change request
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id} [delete]
func DeleteChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c,
		`UPDATE change_requests SET deleted_at = now() WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request dihapus"})
}

// ============================================================
// WORKFLOW TRANSITIONS
// ============================================================

// SubmitChangeRequest godoc
// @Summary Submit CR dari draft → under_review / approved (standard)
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/submit [post]
func SubmitChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()

	// standard type → langsung approved, non-standard → under_review
	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status       = CASE WHEN type = 'standard' THEN 'approved' ELSE 'under_review' END,
			submitted_at = $1,
			updated_at   = now()
		WHERE id = $2 AND status = 'draft' AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request disubmit"})
}

// ApproveChangeRequest godoc
// @Summary Approve CR → status approved + catat approver
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/approve [post]
func ApproveChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status      = 'approved',
			approved_by = $1,
			approved_at = $2,
			updated_at  = now()
		WHERE id = $3 AND status IN ('under_review','submitted') AND deleted_at IS NULL
	`, actor, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request disetujui"})
}

// RejectChangeRequest godoc
// @Summary Tolak CR → status rejected
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/reject [post]
func RejectChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Reason *string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status     = 'rejected',
			updated_at = now()
		WHERE id = $1 AND status IN ('submitted','under_review') AND deleted_at IS NULL
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request ditolak"})
}

// ScheduleChangeRequest godoc
// @Summary Set jadwal implementasi → status scheduled
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/schedule [post]
func ScheduleChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		ChangeWindowStart time.Time `json:"change_window_start" binding:"required"`
		ChangeWindowEnd   time.Time `json:"change_window_end" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status              = 'scheduled',
			change_window_start = $1,
			change_window_end   = $2,
			updated_at          = now()
		WHERE id = $3 AND status = 'approved' AND deleted_at IS NULL
	`, req.ChangeWindowStart, req.ChangeWindowEnd, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request dijadwalkan"})
}

// ImplementChangeRequest godoc
// @Summary Tandai CR sedang diimplementasi → status implementing
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/implement [post]
func ImplementChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status         = 'implementing',
			implemented_by = $1,
			implemented_at = $2,
			updated_at     = now()
		WHERE id = $3 AND status IN ('approved','scheduled') AND deleted_at IS NULL
	`, actor, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "implementasi dimulai"})
}

// CompleteChangeRequest godoc
// @Summary Tandai CR selesai diimplementasi → status implemented
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/complete [post]
func CompleteChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status     = 'implemented',
			updated_at = now()
		WHERE id = $1 AND status = 'implementing' AND deleted_at IS NULL
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "implementasi selesai"})
}

// VerifyChangeRequest godoc
// @Summary Verifikasi hasil CR → status verified
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/verify [post]
func VerifyChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status      = 'verified',
			verified_at = $1,
			updated_at  = now()
		WHERE id = $2 AND status = 'implemented' AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request diverifikasi"})
}

// CloseChangeRequest godoc
// @Summary Tutup CR → status closed
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/close [post]
func CloseChangeRequest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE change_requests SET
			status     = 'closed',
			closed_at  = $1,
			updated_at = now()
		WHERE id = $2 AND status = 'verified' AND deleted_at IS NULL
	`, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "change request ditutup"})
}

// ============================================================
// CAB APPROVALS
// ============================================================

// AddCABApprover godoc
// @Summary Tambah anggota CAB untuk mereview CR
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/approvers [post]
func AddCABApprover(c *gin.Context) {
	changeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		ApproverID int64 `json:"approver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		INSERT INTO change_approvals (change_id, approver_id, decision)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (change_id, approver_id) DO NOTHING
	`, changeID, req.ApproverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "approver berhasil ditambahkan"})
}

// SubmitCABDecision godoc
// @Summary Submit keputusan CAB (approved/rejected/abstain)
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/approvers/decision [post]
func SubmitCABDecision(c *gin.Context) {
	changeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	now := time.Now()

	var req struct {
		Decision string  `json:"decision" binding:"required"`
		Comment  *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE change_approvals SET
			decision   = $1,
			comment    = $2,
			decided_at = $3
		WHERE change_id = $4 AND approver_id = $5
	`, req.Decision, req.Comment, now, changeID, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "keputusan CAB berhasil disimpan"})
}

// ============================================================
// CHANGE TASKS
// ============================================================

// AddChangeTask godoc
// @Summary Tambah sub-task implementasi ke CR
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Router /change-requests/{id}/tasks [post]
func AddChangeTask(c *gin.Context) {
	changeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description *string `json:"description"`
		AssignedTo  *int64  `json:"assigned_to"`
		SeqOrder    *int    `json:"seq_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	seqOrder := 1
	if req.SeqOrder != nil {
		seqOrder = *req.SeqOrder
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO change_tasks (change_id, title, description, assigned_to, seq_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, changeID, req.Title, req.Description, req.AssignedTo, seqOrder).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "task berhasil ditambahkan"})
}

// UpdateChangeTask godoc
// @Summary Update status / detail task
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Param task_id path int true "Task ID"
// @Router /change-requests/{id}/tasks/{task_id} [put]
func UpdateChangeTask(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("task_id"), 10, 64)
	changeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		AssignedTo  *int64  `json:"assigned_to"`
		SeqOrder    *int    `json:"seq_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var completedAt *time.Time
	if req.Status != nil && *req.Status == "done" {
		now := time.Now()
		completedAt = &now
	}

	_, err := database.Pool.Exec(c, `
		UPDATE change_tasks SET
			title        = COALESCE($1, title),
			description  = COALESCE($2, description),
			status       = COALESCE($3, status),
			assigned_to  = COALESCE($4, assigned_to),
			seq_order    = COALESCE($5, seq_order),
			completed_at = COALESCE($6, completed_at),
			updated_at   = now()
		WHERE id = $7 AND change_id = $8
	`, req.Title, req.Description, req.Status, req.AssignedTo, req.SeqOrder,
		completedAt, taskID, changeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "task berhasil diupdate"})
}

// DeleteChangeTask godoc
// @Summary Hapus task dari CR
// @Tags Change Management
// @Param id path int true "Change Request ID"
// @Param task_id path int true "Task ID"
// @Router /change-requests/{id}/tasks/{task_id} [delete]
func DeleteChangeTask(c *gin.Context) {
	taskID, _ := strconv.ParseInt(c.Param("task_id"), 10, 64)
	changeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := database.Pool.Exec(c,
		`DELETE FROM change_tasks WHERE id = $1 AND change_id = $2`, taskID, changeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "task dihapus"})
}
