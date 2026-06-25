package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

func getActorID(c *gin.Context) *int64 {
	v, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	switch id := v.(type) {
	case int64:
		return &id
	case float64:
		n := int64(id)
		return &n
	}
	return nil
}

// ============================================================
// PROBLEM MANAGEMENT
// ============================================================

// GetAllProblems godoc
// @Summary List semua problems
// @Tags Problem Management
// @Produce json
// @Param status query string false "Filter by status"
// @Param known_error query bool false "Filter known errors only"
// @Success 200 {array} models.ProblemInfo
// @Router /problems [get]
func GetAllProblems(c *gin.Context) {
	statusFilter := c.Query("status")
	priorityFilter := c.Query("priority")
	knownErrorFilter := c.Query("known_error")
	q := c.Query("q")
	pg := getPagination(c)

	where := "WHERE p.deleted_at IS NULL"
	args := []interface{}{}
	idx := 1

	if statusFilter != "" {
		where += fmt.Sprintf(" AND p.status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}
	if priorityFilter != "" {
		where += fmt.Sprintf(" AND p.priority = $%d", idx)
		args = append(args, priorityFilter)
		idx++
	}
	if knownErrorFilter == "true" {
		where += " AND p.known_error = true"
	}
	if q != "" {
		where += fmt.Sprintf(" AND (p.title ILIKE $%d OR p.description ILIKE $%d)", idx, idx)
		args = append(args, "%"+q+"%")
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c,
		"SELECT COUNT(DISTINCT p.id) FROM problems p LEFT JOIN problem_incidents pi ON pi.problem_id = p.id "+where,
		args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT
			p.id, p.title, p.description, p.status, p.priority, p.known_error,
			p.assigned_to, e.name AS assignee_name,
			p.related_asset_id, a.name AS related_asset_name,
			p.created_by, cb.name AS created_by_name,
			p.created_at, p.updated_at, p.resolved_at,
			COUNT(pi.ticket_id) AS incident_count
		FROM problems p
		LEFT JOIN employees e  ON e.id  = p.assigned_to
		LEFT JOIN assets a     ON a.id  = p.related_asset_id
		LEFT JOIN employees cb ON cb.id = p.created_by
		LEFT JOIN problem_incidents pi ON pi.problem_id = p.id
		%s
		GROUP BY p.id, e.name, a.name, cb.name
		ORDER BY p.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var problems []models.ProblemInfo
	for rows.Next() {
		var p models.ProblemInfo
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Description, &p.Status, &p.Priority, &p.KnownError,
			&p.AssignedTo, &p.AssigneeName,
			&p.RelatedAssetID, &p.RelatedAssetName,
			&p.CreatedBy, &p.CreatedByName,
			&p.CreatedAt, &p.UpdatedAt, &p.ResolvedAt,
			&p.IncidentCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		problems = append(problems, p)
	}
	if problems == nil {
		problems = []models.ProblemInfo{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: problems, Total: total, Page: pg.Page, Limit: pg.Limit})
}

// GetProblemByID godoc
// @Summary Detail problem beserta linked incidents
// @Tags Problem Management
// @Produce json
// @Param id path int true "Problem ID"
// @Success 200 {object} models.ProblemDetail
// @Router /problems/{id} [get]
func GetProblemByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var p models.ProblemDetail
	err := database.Pool.QueryRow(c, `
		SELECT
			p.id, p.title, p.description, p.status, p.priority, p.known_error,
			p.assigned_to, e.name AS assignee_name,
			p.related_asset_id, a.name AS related_asset_name,
			p.created_by, cb.name AS created_by_name,
			p.created_at, p.updated_at, p.resolved_at,
			0 AS incident_count,
			p.workaround, p.root_cause, p.permanent_solution
		FROM problems p
		LEFT JOIN employees e  ON e.id  = p.assigned_to
		LEFT JOIN assets a     ON a.id  = p.related_asset_id
		LEFT JOIN employees cb ON cb.id = p.created_by
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`, id).Scan(
		&p.ID, &p.Title, &p.Description, &p.Status, &p.Priority, &p.KnownError,
		&p.AssignedTo, &p.AssigneeName,
		&p.RelatedAssetID, &p.RelatedAssetName,
		&p.CreatedBy, &p.CreatedByName,
		&p.CreatedAt, &p.UpdatedAt, &p.ResolvedAt,
		&p.IncidentCount,
		&p.Workaround, &p.RootCause, &p.PermanentSolution,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "problem tidak ditemukan"})
		return
	}

	// Ambil linked incidents
	rows, _ := database.Pool.Query(c, `
		SELECT pi.id, pi.problem_id, pi.ticket_id, pi.linked_by, pi.linked_at, pi.notes,
		       t.subject, t.status, t.priority,
		       e.name AS linked_by_name
		FROM problem_incidents pi
		JOIN tickets t ON t.id = pi.ticket_id
		LEFT JOIN employees e ON e.id = pi.linked_by
		WHERE pi.problem_id = $1
		ORDER BY pi.linked_at DESC
	`, id)
	defer rows.Close()

	for rows.Next() {
		var pi models.ProblemIncident
		_ = rows.Scan(
			&pi.ID, &pi.ProblemID, &pi.TicketID, &pi.LinkedBy, &pi.LinkedAt, &pi.Notes,
			&pi.TicketSubject, &pi.TicketStatus, &pi.TicketPriority,
			&pi.LinkedByName,
		)
		p.LinkedIncidents = append(p.LinkedIncidents, pi)
	}
	if p.LinkedIncidents == nil {
		p.LinkedIncidents = []models.ProblemIncident{}
	}

	c.JSON(http.StatusOK, p)
}

// CreateProblem godoc
// @Summary Buat problem baru
// @Tags Problem Management
// @Accept json
// @Produce json
// @Success 201 {object} models.Problem
// @Router /problems [post]
func CreateProblem(c *gin.Context) {
	var req struct {
		Title          string  `json:"title" binding:"required"`
		Description    *string `json:"description"`
		Priority       string  `json:"priority"`
		AssignedTo     *int64  `json:"assigned_to"`
		RelatedAssetID *int64  `json:"related_asset_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Priority == "" {
		req.Priority = "Medium"
	}

	actor := getActorID(c)
	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO problems (title, description, priority, assigned_to, related_asset_id, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id
	`, req.Title, req.Description, req.Priority, req.AssignedTo, req.RelatedAssetID, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "problem berhasil dibuat"})
}

// UpdateProblem godoc
// @Summary Update problem (status, RCA, workaround, dll)
// @Tags Problem Management
// @Accept json
// @Produce json
// @Param id path int true "Problem ID"
// @Router /problems/{id} [put]
func UpdateProblem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		Title             *string `json:"title"`
		Description       *string `json:"description"`
		Status            *string `json:"status"`
		Priority          *string `json:"priority"`
		AssignedTo        *int64  `json:"assigned_to"`
		RootCause         *string `json:"root_cause"`
		Workaround        *string `json:"workaround"`
		KnownError        *bool   `json:"known_error"`
		PermanentSolution *string `json:"permanent_solution"`
		RelatedAssetID    *int64  `json:"related_asset_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var resolvedAt *time.Time
	if req.Status != nil && *req.Status == "Resolved" {
		now := time.Now()
		resolvedAt = &now
	}

	_, err := database.Pool.Exec(c, `
		UPDATE problems SET
			title              = COALESCE($1, title),
			description        = COALESCE($2, description),
			status             = COALESCE($3, status),
			priority           = COALESCE($4, priority),
			assigned_to        = COALESCE($5, assigned_to),
			root_cause         = COALESCE($6, root_cause),
			workaround         = COALESCE($7, workaround),
			known_error        = COALESCE($8, known_error),
			permanent_solution = COALESCE($9, permanent_solution),
			related_asset_id   = COALESCE($10, related_asset_id),
			resolved_at        = COALESCE($11, resolved_at),
			updated_by         = $12,
			updated_at         = now()
		WHERE id = $13 AND deleted_at IS NULL
	`, req.Title, req.Description, req.Status, req.Priority, req.AssignedTo,
		req.RootCause, req.Workaround, req.KnownError, req.PermanentSolution,
		req.RelatedAssetID, resolvedAt, actor, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "problem berhasil diupdate"})
}

// DeleteProblem godoc
// @Summary Soft-delete problem
// @Tags Problem Management
// @Param id path int true "Problem ID"
// @Router /problems/{id} [delete]
func DeleteProblem(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `UPDATE problems SET deleted_at = now() WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "problem dihapus"})
}

// ============================================================
// PROBLEM ↔ INCIDENT LINKING
// ============================================================

// LinkIncidentToProblem godoc
// @Summary Hubungkan ticket (incident) ke problem
// @Tags Problem Management
// @Param id path int true "Problem ID"
// @Router /problems/{id}/incidents [post]
func LinkIncidentToProblem(c *gin.Context) {
	problemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		TicketID int64   `json:"ticket_id" binding:"required"`
		Notes    *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		INSERT INTO problem_incidents (problem_id, ticket_id, linked_by, notes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (problem_id, ticket_id) DO NOTHING
	`, problemID, req.TicketID, actor, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Sync linked_problem_id di tabel tickets
	_, _ = database.Pool.Exec(c,
		`UPDATE tickets SET linked_problem_id = $1 WHERE id = $2`,
		problemID, req.TicketID,
	)

	c.JSON(http.StatusCreated, gin.H{"message": "incident berhasil dihubungkan ke problem"})
}

// UnlinkIncidentFromProblem godoc
// @Summary Putus relasi ticket dari problem
// @Tags Problem Management
// @Param id path int true "Problem ID"
// @Param ticket_id path int true "Ticket ID"
// @Router /problems/{id}/incidents/{ticket_id} [delete]
func UnlinkIncidentFromProblem(c *gin.Context) {
	problemID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticketID, _ := strconv.ParseInt(c.Param("ticket_id"), 10, 64)

	_, err := database.Pool.Exec(c,
		`DELETE FROM problem_incidents WHERE problem_id = $1 AND ticket_id = $2`,
		problemID, ticketID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "incident berhasil dilepas dari problem"})
}

// ============================================================
// POST-MORTEM / ROOT CAUSE ANALYSIS
// ============================================================

// CreatePostmortem godoc
// @Summary Buat post-mortem untuk incident (ticket)
// @Tags Post-Mortem
// @Accept json
// @Produce json
// @Param ticket_id path int true "Ticket ID"
// @Router /tickets/{ticket_id}/postmortem [post]
func CreatePostmortem(c *gin.Context) {
	ticketID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		ProblemID           *int64          `json:"problem_id"`
		Timeline            json.RawMessage `json:"timeline"`
		RootCause           *string         `json:"root_cause"`
		ContributingFactors *string         `json:"contributing_factors"`
		LessonsLearned      *string         `json:"lessons_learned"`
		ActionItems         json.RawMessage `json:"action_items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	timeline := req.Timeline
	if timeline == nil {
		timeline = json.RawMessage("[]")
	}
	actionItems := req.ActionItems
	if actionItems == nil {
		actionItems = json.RawMessage("[]")
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO incident_postmortems
			(ticket_id, problem_id, timeline, root_cause, contributing_factors,
			 lessons_learned, action_items, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ticket_id) DO UPDATE SET
			problem_id           = EXCLUDED.problem_id,
			timeline             = EXCLUDED.timeline,
			root_cause           = EXCLUDED.root_cause,
			contributing_factors = EXCLUDED.contributing_factors,
			lessons_learned      = EXCLUDED.lessons_learned,
			action_items         = EXCLUDED.action_items,
			updated_at           = now()
		RETURNING id
	`, ticketID, req.ProblemID, timeline, req.RootCause, req.ContributingFactors,
		req.LessonsLearned, actionItems, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "post-mortem berhasil disimpan"})
}

// GetPostmortem godoc
// @Summary Ambil post-mortem berdasarkan ticket
// @Tags Post-Mortem
// @Produce json
// @Param ticket_id path int true "Ticket ID"
// @Router /tickets/{ticket_id}/postmortem [get]
func GetPostmortem(c *gin.Context) {
	ticketID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var pm models.IncidentPostmortem
	err := database.Pool.QueryRow(c, `
		SELECT
			pm.id, pm.ticket_id, pm.problem_id,
			pm.timeline::text, pm.root_cause, pm.contributing_factors,
			pm.lessons_learned, pm.action_items::text,
			pm.reviewed_by, pm.reviewed_at,
			pm.created_by, pm.created_at, pm.updated_at,
			t.subject AS ticket_subject,
			rv.name AS reviewed_by_name,
			cb.name AS created_by_name
		FROM incident_postmortems pm
		JOIN tickets t       ON t.id  = pm.ticket_id
		LEFT JOIN employees rv ON rv.id = pm.reviewed_by
		LEFT JOIN employees cb ON cb.id = pm.created_by
		WHERE pm.ticket_id = $1
	`, ticketID).Scan(
		&pm.ID, &pm.TicketID, &pm.ProblemID,
		&pm.Timeline, &pm.RootCause, &pm.ContributingFactors,
		&pm.LessonsLearned, &pm.ActionItems,
		&pm.ReviewedBy, &pm.ReviewedAt,
		&pm.CreatedBy, &pm.CreatedAt, &pm.UpdatedAt,
		&pm.TicketSubject, &pm.ReviewedByName, &pm.CreatedByName,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post-mortem tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, pm)
}

// ReviewPostmortem godoc
// @Summary Tandai post-mortem sudah di-review
// @Tags Post-Mortem
// @Param ticket_id path int true "Ticket ID"
// @Router /tickets/{ticket_id}/postmortem/review [post]
func ReviewPostmortem(c *gin.Context) {
	ticketID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	now := time.Now()

	_, err := database.Pool.Exec(c, `
		UPDATE incident_postmortems
		SET reviewed_by = $1, reviewed_at = $2, updated_at = now()
		WHERE ticket_id = $3
	`, actor, now, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "post-mortem berhasil disetujui"})
}

// ============================================================
// ESCALATION RULES
// ============================================================

// GetEscalationRules godoc
// @Summary List semua aturan eskalasi
// @Tags Escalation
// @Produce json
// @Router /escalation-rules [get]
func GetEscalationRules(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT
			er.id, er.name, er.category_code, er.service_code, er.priority,
			er.trigger_after_minutes, er.action,
			er.escalate_to_role, er.escalate_to_employee, er.notify_emails,
			er.is_active, er.created_by, er.created_at, er.updated_at,
			e.name AS escalate_to_employee_name
		FROM escalation_rules er
		LEFT JOIN employees e ON e.id = er.escalate_to_employee
		ORDER BY er.priority, er.trigger_after_minutes
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []models.EscalationRule
	for rows.Next() {
		var r models.EscalationRule
		if err := rows.Scan(
			&r.ID, &r.Name, &r.CategoryCode, &r.ServiceCode, &r.Priority,
			&r.TriggerAfterMinutes, &r.Action,
			&r.EscalateToRole, &r.EscalateToEmployee, &r.NotifyEmails,
			&r.IsActive, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
			&r.EscalateToEmployeeName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rules = append(rules, r)
	}
	if rules == nil {
		rules = []models.EscalationRule{}
	}
	c.JSON(http.StatusOK, rules)
}

// CreateEscalationRule godoc
// @Summary Buat aturan eskalasi baru
// @Tags Escalation
// @Accept json
// @Produce json
// @Router /escalation-rules [post]
func CreateEscalationRule(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		Name                string  `json:"name" binding:"required"`
		CategoryCode        *string `json:"category_code"`
		ServiceCode         *string `json:"service_code"`
		Priority            string  `json:"priority" binding:"required"`
		TriggerAfterMinutes int     `json:"trigger_after_minutes" binding:"required,min=1"`
		Action              string  `json:"action" binding:"required"`
		EscalateToRole      *string `json:"escalate_to_role"`
		EscalateToEmployee  *int64  `json:"escalate_to_employee"`
		NotifyEmails        *string `json:"notify_emails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO escalation_rules
			(name, category_code, service_code, priority, trigger_after_minutes,
			 action, escalate_to_role, escalate_to_employee, notify_emails, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id
	`, req.Name, req.CategoryCode, req.ServiceCode, req.Priority, req.TriggerAfterMinutes,
		req.Action, req.EscalateToRole, req.EscalateToEmployee, req.NotifyEmails, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "escalation rule berhasil dibuat"})
}

// UpdateEscalationRule godoc
// @Summary Update aturan eskalasi
// @Tags Escalation
// @Param id path int true "Rule ID"
// @Router /escalation-rules/{id} [put]
func UpdateEscalationRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Name                *string `json:"name"`
		Priority            *string `json:"priority"`
		TriggerAfterMinutes *int    `json:"trigger_after_minutes"`
		Action              *string `json:"action"`
		EscalateToRole      *string `json:"escalate_to_role"`
		EscalateToEmployee  *int64  `json:"escalate_to_employee"`
		NotifyEmails        *string `json:"notify_emails"`
		IsActive            *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE escalation_rules SET
			name                 = COALESCE($1, name),
			priority             = COALESCE($2, priority),
			trigger_after_minutes = COALESCE($3, trigger_after_minutes),
			action               = COALESCE($4, action),
			escalate_to_role     = COALESCE($5, escalate_to_role),
			escalate_to_employee = COALESCE($6, escalate_to_employee),
			notify_emails        = COALESCE($7, notify_emails),
			is_active            = COALESCE($8, is_active),
			updated_at           = now()
		WHERE id = $9
	`, req.Name, req.Priority, req.TriggerAfterMinutes, req.Action,
		req.EscalateToRole, req.EscalateToEmployee, req.NotifyEmails, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "escalation rule berhasil diupdate"})
}

// DeleteEscalationRule godoc
// @Summary Hapus aturan eskalasi
// @Tags Escalation
// @Param id path int true "Rule ID"
// @Router /escalation-rules/{id} [delete]
func DeleteEscalationRule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM escalation_rules WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "escalation rule dihapus"})
}
