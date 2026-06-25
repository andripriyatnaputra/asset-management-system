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

// ============================================================
// DR PLANS
// ============================================================

// GetAllDRPlans godoc
// @Summary List semua DR/BCP plans
// @Tags DR/BCP
// @Produce json
// @Param type query string false "Filter by plan type (DR|BCP|COOP)"
// @Param status query string false "Filter by status"
// @Param q query string false "Search by name"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} pagedResponse
// @Security BearerAuth
// @Router /dr/plans [get]
func GetAllDRPlans(c *gin.Context) {
	planType := c.Query("type")
	status := c.Query("status")
	q := c.Query("q")
	pg := getPagination(c)

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if planType != "" {
		where += fmt.Sprintf(" AND p.plan_type = $%d", idx)
		args = append(args, planType)
		idx++
	}
	if status != "" {
		where += fmt.Sprintf(" AND p.status = $%d", idx)
		args = append(args, status)
		idx++
	}
	if q != "" {
		where += fmt.Sprintf(" AND p.name ILIKE $%d", idx)
		args = append(args, "%"+q+"%")
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(*) FROM dr_plans p "+where, args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT p.id, p.name, p.description, p.plan_type, p.rto_hours, p.rpo_hours,
		       p.status, p.owner_id, p.last_tested_at, p.next_test_due,
		       p.created_by, p.created_at, p.updated_at,
		       o.name AS owner_name, cb.name AS created_by_name,
		       (SELECT count(*) FROM dr_plan_steps s WHERE s.plan_id = p.id) AS step_count
		FROM dr_plans p
		LEFT JOIN employees o  ON o.id = p.owner_id
		LEFT JOIN employees cb ON cb.id = p.created_by
		%s
		ORDER BY p.plan_type, p.name
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.DRPlan
	for rows.Next() {
		var p models.DRPlan
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.PlanType, &p.RTOHours, &p.RPOHours,
			&p.Status, &p.OwnerID, &p.LastTestedAt, &p.NextTestDue,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
			&p.OwnerName, &p.CreatedByName, &p.StepCount,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, p)
	}
	if list == nil {
		list = []models.DRPlan{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

// GetDRPlanByID godoc
// @Summary Detail DR/BCP plan beserta langkah-langkah
// @Tags DR/BCP
// @Produce json
// @Param id path int true "Plan ID"
// @Success 200 {object} models.DRPlan
// @Security BearerAuth
// @Router /dr/plans/{id} [get]
func GetDRPlanByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var p models.DRPlan
	err := database.Pool.QueryRow(c, `
		SELECT p.id, p.name, p.description, p.plan_type, p.rto_hours, p.rpo_hours,
		       p.status, p.owner_id, p.last_tested_at, p.next_test_due,
		       p.created_by, p.created_at, p.updated_at,
		       o.name, cb.name,
		       (SELECT count(*) FROM dr_plan_steps s WHERE s.plan_id = p.id)
		FROM dr_plans p
		LEFT JOIN employees o  ON o.id = p.owner_id
		LEFT JOIN employees cb ON cb.id = p.created_by
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.PlanType, &p.RTOHours, &p.RPOHours,
		&p.Status, &p.OwnerID, &p.LastTestedAt, &p.NextTestDue,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		&p.OwnerName, &p.CreatedByName, &p.StepCount,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DR plan tidak ditemukan"})
		return
	}

	// Load steps
	steps, _ := loadDRPlanSteps(c, id)
	p.Steps = steps

	c.JSON(http.StatusOK, p)
}

func loadDRPlanSteps(c *gin.Context, planID int64) ([]models.DRPlanStep, error) {
	rows, err := database.Pool.Query(c, `
		SELECT s.id, s.plan_id, s.step_order, s.title, s.description,
		       s.responsible, s.duration_minutes, s.is_critical, s.created_at,
		       e.name AS responsible_name
		FROM dr_plan_steps s
		LEFT JOIN employees e ON e.id = s.responsible
		WHERE s.plan_id = $1
		ORDER BY s.step_order
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []models.DRPlanStep
	for rows.Next() {
		var s models.DRPlanStep
		if err := rows.Scan(&s.ID, &s.PlanID, &s.StepOrder, &s.Title, &s.Description,
			&s.Responsible, &s.DurationMinutes, &s.IsCritical, &s.CreatedAt,
			&s.ResponsibleName); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, nil
}

// CreateDRPlan godoc
// @Summary Buat DR/BCP plan baru
// @Tags DR/BCP
// @Accept json
// @Produce json
// @Param body body models.DRPlan true "DR Plan"
// @Success 201 {object} models.DRPlan
// @Security BearerAuth
// @Router /dr/plans [post]
func CreateDRPlan(c *gin.Context) {
	actor := getActorID(c)
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description *string  `json:"description"`
		PlanType    string   `json:"plan_type"` // dr|bcp|contingency
		RTOHours    *float64 `json:"rto_hours"`
		RPOHours    *float64 `json:"rpo_hours"`
		OwnerID     *int64   `json:"owner_id"`
		NextTestDue *string  `json:"next_test_due"` // YYYY-MM-DD
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PlanType == "" {
		req.PlanType = "dr"
	}

	var nextTest *time.Time
	if req.NextTestDue != nil {
		t, err := time.Parse("2006-01-02", *req.NextTestDue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format next_test_due harus YYYY-MM-DD"})
			return
		}
		nextTest = &t
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO dr_plans (name, description, plan_type, rto_hours, rpo_hours, owner_id, next_test_due, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
	`, req.Name, req.Description, req.PlanType, req.RTOHours, req.RPOHours,
		req.OwnerID, nextTest, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "DR plan berhasil dibuat"})
}

func UpdateDRPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		RTOHours    *float64 `json:"rto_hours"`
		RPOHours    *float64 `json:"rpo_hours"`
		OwnerID     *int64   `json:"owner_id"`
		NextTestDue *string  `json:"next_test_due"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var nextTest *time.Time
	if req.NextTestDue != nil {
		t, err := time.Parse("2006-01-02", *req.NextTestDue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format next_test_due harus YYYY-MM-DD"})
			return
		}
		nextTest = &t
	}

	_, err := database.Pool.Exec(c, `
		UPDATE dr_plans SET
			name         = COALESCE($1, name),
			description  = COALESCE($2, description),
			rto_hours    = COALESCE($3, rto_hours),
			rpo_hours    = COALESCE($4, rpo_hours),
			owner_id     = COALESCE($5, owner_id),
			next_test_due = COALESCE($6, next_test_due),
			updated_at   = now()
		WHERE id = $7
	`, req.Name, req.Description, req.RTOHours, req.RPOHours, req.OwnerID, nextTest, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "DR plan berhasil diupdate"})
}

func ActivateDRPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `UPDATE dr_plans SET status='active', updated_at=now() WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "DR plan diaktifkan"})
}

func DeleteDRPlan(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM dr_plans WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "DR plan berhasil dihapus"})
}

// ============================================================
// DR PLAN STEPS
// ============================================================

func AddDRPlanStep(c *gin.Context) {
	planID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		StepOrder       int     `json:"step_order" binding:"required"`
		Title           string  `json:"title" binding:"required"`
		Description     *string `json:"description"`
		Responsible     *int64  `json:"responsible"`
		DurationMinutes *int    `json:"duration_minutes"`
		IsCritical      bool    `json:"is_critical"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO dr_plan_steps (plan_id, step_order, title, description, responsible, duration_minutes, is_critical)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
	`, planID, req.StepOrder, req.Title, req.Description,
		req.Responsible, req.DurationMinutes, req.IsCritical).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "langkah DR berhasil ditambahkan"})
}

func UpdateDRPlanStep(c *gin.Context) {
	stepID, _ := strconv.ParseInt(c.Param("step_id"), 10, 64)
	var req struct {
		StepOrder       *int    `json:"step_order"`
		Title           *string `json:"title"`
		Description     *string `json:"description"`
		Responsible     *int64  `json:"responsible"`
		DurationMinutes *int    `json:"duration_minutes"`
		IsCritical      *bool   `json:"is_critical"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE dr_plan_steps SET
			step_order       = COALESCE($1, step_order),
			title            = COALESCE($2, title),
			description      = COALESCE($3, description),
			responsible      = COALESCE($4, responsible),
			duration_minutes = COALESCE($5, duration_minutes),
			is_critical      = COALESCE($6, is_critical)
		WHERE id = $7
	`, req.StepOrder, req.Title, req.Description, req.Responsible, req.DurationMinutes, req.IsCritical, stepID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "langkah DR berhasil diupdate"})
}

func DeleteDRPlanStep(c *gin.Context) {
	stepID, _ := strconv.ParseInt(c.Param("step_id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM dr_plan_steps WHERE id = $1`, stepID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "langkah DR berhasil dihapus"})
}

// ============================================================
// DR TESTS
// ============================================================

// GetDRTests godoc
// @Summary List DR test records
// @Tags DR/BCP
// @Produce json
// @Param plan_id query int false "Filter by plan ID"
// @Param status query string false "Filter by status"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} pagedResponse
// @Security BearerAuth
// @Router /dr/tests [get]
func GetDRTests(c *gin.Context) {
	planID := c.Query("plan_id")
	statusFilter := c.Query("status")
	pg := getPagination(c)

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if planID != "" {
		where += fmt.Sprintf(" AND t.plan_id = $%d", idx)
		args = append(args, planID)
		idx++
	}
	if statusFilter != "" {
		where += fmt.Sprintf(" AND t.status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(*) FROM dr_tests t "+where, args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT t.id, t.plan_id, t.test_type, t.scheduled_at, t.started_at, t.completed_at,
		       t.status, t.rto_achieved_hours, t.rpo_achieved_hours, t.outcome,
		       t.notes, t.conducted_by, t.created_by, t.created_at, t.updated_at,
		       p.name AS plan_name, e.name AS conducted_by_name
		FROM dr_tests t
		JOIN dr_plans p       ON p.id = t.plan_id
		LEFT JOIN employees e ON e.id = t.conducted_by
		%s
		ORDER BY t.scheduled_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.DRTest
	for rows.Next() {
		var t models.DRTest
		if err := rows.Scan(
			&t.ID, &t.PlanID, &t.TestType, &t.ScheduledAt, &t.StartedAt, &t.CompletedAt,
			&t.Status, &t.RTOAchievedHours, &t.RPOAchievedHours, &t.Outcome,
			&t.Notes, &t.ConductedBy, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
			&t.PlanName, &t.ConductedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, t)
	}
	if list == nil {
		list = []models.DRTest{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

func GetDRTestByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var t models.DRTest
	err := database.Pool.QueryRow(c, `
		SELECT t.id, t.plan_id, t.test_type, t.scheduled_at, t.started_at, t.completed_at,
		       t.status, t.rto_achieved_hours, t.rpo_achieved_hours, t.outcome,
		       t.notes, t.conducted_by, t.created_by, t.created_at, t.updated_at,
		       p.name, e.name
		FROM dr_tests t
		JOIN dr_plans p      ON p.id = t.plan_id
		LEFT JOIN employees e ON e.id = t.conducted_by
		WHERE t.id = $1
	`, id).Scan(
		&t.ID, &t.PlanID, &t.TestType, &t.ScheduledAt, &t.StartedAt, &t.CompletedAt,
		&t.Status, &t.RTOAchievedHours, &t.RPOAchievedHours, &t.Outcome,
		&t.Notes, &t.ConductedBy, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.PlanName, &t.ConductedByName,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DR test tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func ScheduleDRTest(c *gin.Context) {
	planID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	var req struct {
		TestType    string `json:"test_type" binding:"required"` // tabletop|walkthrough|simulation|full_test
		ScheduledAt string `json:"scheduled_at" binding:"required"` // RFC3339 or YYYY-MM-DD
		Notes       *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		scheduledAt, err = time.Parse("2006-01-02", req.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format scheduled_at: RFC3339 atau YYYY-MM-DD"})
			return
		}
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO dr_tests (plan_id, test_type, scheduled_at, notes, created_by)
		VALUES ($1,$2,$3,$4,$5) RETURNING id
	`, planID, req.TestType, scheduledAt, req.Notes, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "DR test berhasil dijadwalkan"})
}

func StartDRTest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)

	_, err := database.Pool.Exec(c, `
		UPDATE dr_tests SET
			status       = 'in_progress',
			started_at   = now(),
			conducted_by = COALESCE(conducted_by, $1),
			updated_at   = now()
		WHERE id = $2 AND status = 'scheduled'
	`, actor, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "DR test dimulai"})
}

func CompleteDRTest(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		Outcome           string   `json:"outcome" binding:"required"` // passed|partial|failed
		RTOAchievedHours  *float64 `json:"rto_achieved_hours"`
		RPOAchievedHours  *float64 `json:"rpo_achieved_hours"`
		Notes             *string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	_, err := database.Pool.Exec(c, `
		UPDATE dr_tests SET
			status              = 'completed',
			completed_at        = $1,
			outcome             = $2,
			rto_achieved_hours  = $3,
			rpo_achieved_hours  = $4,
			notes               = COALESCE($5, notes),
			updated_at          = $1
		WHERE id = $6 AND status = 'in_progress'
	`, now, req.Outcome, req.RTOAchievedHours, req.RPOAchievedHours, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update last_tested_at di plan
	_, _ = database.Pool.Exec(c, `
		UPDATE dr_plans SET last_tested_at = $1, updated_at = $1
		WHERE id = (SELECT plan_id FROM dr_tests WHERE id = $2)
	`, now, id)

	c.JSON(http.StatusOK, gin.H{"message": "DR test selesai", "outcome": req.Outcome})
}

func RecordTestResult(c *gin.Context) {
	testID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		StepID                *int64  `json:"step_id"`
		Status                string  `json:"status" binding:"required"` // passed|failed|skipped|not_tested
		ActualDurationMinutes *int    `json:"actual_duration_minutes"`
		Notes                 *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO dr_test_results (test_id, step_id, status, actual_duration_minutes, notes)
		VALUES ($1,$2,$3,$4,$5) RETURNING id
	`, testID, req.StepID, req.Status, req.ActualDurationMinutes, req.Notes).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "hasil test berhasil dicatat"})
}

func GetTestResults(c *gin.Context) {
	testID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.Pool.Query(c, `
		SELECT r.id, r.test_id, r.step_id, r.status,
		       r.actual_duration_minutes, r.notes, r.created_at,
		       s.title AS step_title, s.step_order
		FROM dr_test_results r
		LEFT JOIN dr_plan_steps s ON s.id = r.step_id
		WHERE r.test_id = $1
		ORDER BY s.step_order NULLS LAST
	`, testID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.DRTestResult
	for rows.Next() {
		var r models.DRTestResult
		if err := rows.Scan(&r.ID, &r.TestID, &r.StepID, &r.Status,
			&r.ActualDurationMinutes, &r.Notes, &r.CreatedAt,
			&r.StepTitle, &r.StepOrder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, r)
	}
	if list == nil {
		list = []models.DRTestResult{}
	}
	c.JSON(http.StatusOK, list)
}
