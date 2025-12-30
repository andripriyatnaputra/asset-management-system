// File: backend/handlers/budget_handler.go
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// ============================================================
// 🔹 RecalculateBudgetTotals — sinkronisasi realisasi anggaran
// ============================================================
func RecalculateBudgetTotals(ctx context.Context, budgetID int64) error {
	// Hitung ulang total pengeluaran & update tabel budgets
	query := `
		WITH calc AS (
			SELECT 
				bt.budget_id,
				COALESCE(SUM(bt.amount), 0) AS total_spent
			FROM budget_transactions bt
			JOIN budgets b ON b.id = bt.budget_id
			WHERE bt.budget_id = $1
			  AND b.deleted_at IS NULL
			GROUP BY bt.budget_id
		)
		UPDATE budgets b
		   SET used_amount = c.total_spent,
		       updated_at  = NOW()
		  FROM calc c
		 WHERE b.id = c.budget_id;
	`

	if _, err := database.Pool.Exec(ctx, query, budgetID); err != nil {
		return fmt.Errorf("failed to recalc totals: %v", err)
	}

	// 🔹 Validasi overspend untuk alert governance (optional tapi disarankan)
	var total, used float64
	if err := database.Pool.QueryRow(ctx,
		`SELECT total_amount, used_amount FROM budgets WHERE id=$1`, budgetID).
		Scan(&total, &used); err == nil {
		if used > total {
			msg := fmt.Sprintf("⚠️ Budget %d overspent (%.2f / %.2f)", budgetID, used, total)
			services.BroadcastAlert(msg, "warning")
		}
	}

	return nil
}

// ============================================================
// 🔹 CREATE BUDGET (POST /budgets)
// ============================================================
func CreateBudget(c *gin.Context) {
	type CreateBudgetRequest struct {
		Name         string  `json:"name" binding:"required"`
		DepartmentID *int64  `json:"department_id"`
		StartDateStr string  `json:"start_date" binding:"required"`
		EndDateStr   string  `json:"end_date" binding:"required"`
		TotalAmount  float64 `json:"total_amount" binding:"required"`
		Category     *string `json:"category"`
		CostCenterID *int64  `json:"cost_center_id"`
		Currency     *string `json:"currency"`
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parseDate := func(s string) *time.Time {
		for _, layout := range []string{"2006-01-02", time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
		return nil
	}
	startDate, endDate := parseDate(req.StartDateStr), parseDate(req.EndDateStr)
	if startDate == nil || endDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	if endDate.Before(*startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date cannot be before start_date"})
		return
	}

	// Validasi cost_center_id (optional)
	if req.CostCenterID != nil {
		var exists bool
		if err := database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1)`, *req.CostCenterID).Scan(&exists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cost_center_id"})
			return
		}
	}

	// User pembuat
	var createdBy *int64
	if uid, ok := c.Get("user_id"); ok {
		if v, ok2 := uid.(int64); ok2 {
			createdBy = &v
		}
	}

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO budgets
		 (name, department_id, start_date, end_date, total_amount,
		  category, cost_center_id, currency, approved_by)
		VALUES ($1,$2,$3,$4,$5,
		        COALESCE($6,'CAPEX'),
		        $7,
		        COALESCE($8,'IDR'),
		        $9)
		RETURNING id`,
		req.Name, req.DepartmentID, startDate, endDate, req.TotalAmount,
		req.Category, req.CostCenterID, req.Currency, createdBy,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "budgets", id, "CREATE", req)
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Budget created successfully (Grade A++)"})
}

// GetAllBudgets retrieves all budgets
type BudgetInfo struct {
	models.Budget
	SpentAmount float64 `json:"spent_amount"`
}

// ============================================================
// 📊 GET ALL BUDGETS – dengan Health & Governance Score
// ============================================================
// ============================================================
// 📊 GetAllBudgets — daftar seluruh anggaran (Grade A++)
// ============================================================
func GetAllBudgets(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT 
			b.id,
			b.name,
			b.department_id,
			b.start_date,
			b.end_date,
			b.total_amount,
			COALESCE(SUM(bt.amount),0) AS spent,
			b.category,
			b.cost_center_id,
			cc.code AS cost_center_code,
			cc.name AS cost_center_name,
			b.currency,
			b.created_at
		FROM budgets b
		LEFT JOIN budget_transactions bt ON b.id = bt.budget_id
		LEFT JOIN cost_centers cc ON b.cost_center_id = cc.id
		WHERE b.deleted_at IS NULL
		GROUP BY 
			b.id, b.name, b.department_id, b.start_date, b.end_date,
			b.total_amount, b.category, b.cost_center_id,
			cc.code, cc.name, b.currency, b.created_at
		ORDER BY b.start_date DESC;
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type BudgetRow struct {
		ID           int64      `json:"id"`
		Name         string     `json:"name"`
		Total        float64    `json:"total_amount"`
		Spent        float64    `json:"spent_amount"`
		Category     *string    `json:"category"`
		Currency     *string    `json:"currency"`
		StartDate    *time.Time `json:"start_date"`
		EndDate      *time.Time `json:"end_date"`
		CostCenter   *string    `json:"cost_center"`
		Health       float64    `json:"health"`
		GovScore     float64    `json:"governance_score"`
		Utilization  float64    `json:"utilization_percent"`
		Overspent    bool       `json:"overspent"`
		AlertMessage *string    `json:"alert_message,omitempty"`
	}

	var list []BudgetRow

	for rows.Next() {
		var (
			id           int64
			name         string
			deptID       sql.NullInt64
			startDate    time.Time
			endDate      time.Time
			total        float64
			spent        float64
			category     sql.NullString
			costCenterID sql.NullInt64
			ccCode       sql.NullString
			ccName       sql.NullString
			currency     sql.NullString
			createdAt    time.Time
		)

		if err := rows.Scan(
			&id, &name, &deptID, &startDate, &endDate,
			&total, &spent, &category,
			&costCenterID, &ccCode, &ccName, &currency, &createdAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ccDisplay := "-"
		if ccCode.Valid || ccName.Valid {
			ccDisplay = fmt.Sprintf("%s — %s", ccCode.String, ccName.String)
		}

		r := BudgetRow{
			ID:          id,
			Name:        name,
			Total:       total,
			Spent:       spent,
			Category:    nullableStr(category),
			Currency:    nullableStr(currency),
			StartDate:   &startDate,
			EndDate:     &endDate,
			CostCenter:  &ccDisplay,
			Utilization: (spent / math.Max(total, 1)) * 100,
		}

		r.Health = math.Max(0, 100-r.Utilization)
		r.GovScore = governanceScore(true, true, true)
		r.Overspent = r.Spent > r.Total

		if r.Overspent {
			msg := fmt.Sprintf("Budget %s overspent (%.1f / %.1f)", r.Name, r.Spent, r.Total)
			services.BroadcastAlert(msg, "warning")
			r.AlertMessage = &msg
		}

		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
	middleware.LogAction(c, "budgets", 0, "LIST", nil)
}

// helper for nullable string
func nullableStr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// ============================================================
// 🔹 UPDATE BUDGET (PUT /budgets/:id)
// ============================================================
func UpdateBudget(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		TotalAmount  *float64 `json:"total_amount,omitempty"`
		EndDate      *string  `json:"end_date,omitempty"`
		CostCenterID *int64   `json:"cost_center_id,omitempty"`
		Currency     *string  `json:"currency,omitempty"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	set := []string{}
	args := []any{}
	i := 1

	if body.TotalAmount != nil {
		set = append(set, fmt.Sprintf("total_amount=$%d", i))
		args = append(args, *body.TotalAmount)
		i++
	}
	if body.EndDate != nil {
		var t time.Time
		var err error

		// ISO 8601 format
		t, err = time.Parse(time.RFC3339, *body.EndDate)
		if err != nil {
			// Fallback ke format pendek
			t, err = time.Parse("2006-01-02", *body.EndDate)
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}

		set = append(set, fmt.Sprintf("end_date=$%d", i))
		args = append(args, t)
		i++
	}
	if body.CostCenterID != nil {
		set = append(set, fmt.Sprintf("cost_center_id=$%d", i))
		args = append(args, *body.CostCenterID)
		i++
	}
	if body.Currency != nil {
		set = append(set, fmt.Sprintf("currency=$%d", i))
		args = append(args, *body.Currency)
		i++
	}

	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	query := fmt.Sprintf(`UPDATE budgets SET %s, updated_at=NOW() WHERE id=$%d`, strings.Join(set, ","), i)
	args = append(args, id)

	if _, err := database.Pool.Exec(c.Request.Context(), query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "budgets", mustAtoi64(id), "UPDATE", body)
	c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully"})
}

// ============================================================
// 🔹 DELETE BUDGET (soft delete + check transaksi)
// ============================================================
func DeleteBudget(c *gin.Context) {
	id := c.Param("id")
	var used float64

	links, err := services.CheckActiveLinkages(c.Request.Context(), "budgets", mustAtoi64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "linkage check failed"})
		return
	}
	if len(links) > 0 {
		middleware.LogAction(c, "budgets", mustAtoi64(id), "DELETE_BLOCKED", gin.H{"linked_assets": links})
		c.JSON(http.StatusForbidden, gin.H{
			"error": "cannot delete budget; assets still linked",
		})
		return
	}

	go services.BroadcastAlert(
		fmt.Sprintf("🛑 Delete attempt blocked — entity has active linkages (%v)", links),
		"warning",
	)

	if err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(SUM(amount),0) FROM budget_transactions WHERE budget_id=$1`, id).Scan(&used); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check fail"})
		return
	}
	if used > 0 {
		msg := fmt.Sprintf("Budget %s still has transactions (%.2f)", id, used)
		services.BroadcastAlert(msg, "warning")
		c.JSON(http.StatusConflict, gin.H{"error": msg})
		return
	}

	if _, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE budgets SET deleted_at=NOW() WHERE id=$1`, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "budgets", mustAtoi64(id), "DELETE", nil)
	c.JSON(http.StatusOK, gin.H{"message": "Budget deleted successfully"})
}

// ============================================================
// 🔹 CREATE BUDGET TRANSACTION (insert + validate)
// ============================================================
type BudgetTxInput struct {
	BudgetID      int64      `json:"budget_id" binding:"required"`
	EntityType    string     `json:"entity_type,omitempty"`
	EntityID      *int64     `json:"entity_id,omitempty"`
	Amount        float64    `json:"amount" binding:"required"`
	Category      *string    `json:"category,omitempty"`
	CostCenterID  *int64     `json:"cost_center_id,omitempty"`
	Currency      *string    `json:"currency,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	CreatedBy     *int64     `json:"created_by,omitempty"`
	ExchangeRate  *float64   `json:"exchange_rate,omitempty"`
	TaxAmount     *float64   `json:"tax_amount,omitempty"`
	TransactionAt *time.Time `json:"transaction_at,omitempty"`
}

func CreateBudgetTransaction(ctx context.Context, tx BudgetTxInput) error {
	query := `
	INSERT INTO budget_transactions
	  (budget_id, entity_type, entity_id, amount, currency, exchange_rate,
	   tax_amount, cost_center_id, category, transaction_date, notes, created_by, created_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,COALESCE($10,NOW()),$11,$12,NOW())`

	_, err := database.Pool.Exec(ctx, query,
		tx.BudgetID, tx.EntityType, tx.EntityID, tx.Amount,
		tx.Currency, tx.ExchangeRate, tx.TaxAmount,
		tx.CostCenterID, tx.Category, tx.TransactionAt,
		tx.Notes, tx.CreatedBy)

	if err != nil {
		fmt.Println("[WARN] gagal mencatat transaksi budget:", err)
		return err
	}

	return nil
}

// ============================================================
// 🔹 VALIDATE BEFORE TRANSACTION (periode & overspend guard)
// ============================================================
func ValidateBudgetBeforeTransaction(ctx context.Context, budgetID int64, amount float64, txDate time.Time) error {
	var startDate, endDate time.Time
	var totalAmount, spentAmount float64

	query := `
		SELECT b.start_date, b.end_date, b.total_amount,
		       COALESCE(SUM(bt.amount), 0)
		FROM budgets b
		LEFT JOIN budget_transactions bt ON bt.budget_id = b.id
		WHERE b.id = $1 AND b.deleted_at IS NULL
		GROUP BY b.start_date, b.end_date, b.total_amount
	`
	err := database.Pool.QueryRow(ctx, query, budgetID).Scan(&startDate, &endDate, &totalAmount, &spentAmount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("budget tidak ditemukan (id=%d)", budgetID)
		}
		return fmt.Errorf("gagal membaca data anggaran: %v", err)
	}

	if txDate.Before(startDate) || txDate.After(endDate) {
		return fmt.Errorf("tanggal transaksi di luar periode anggaran (%s - %s)",
			startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	}

	const tolerance = 0.05
	limit := totalAmount * (1 + tolerance)
	if spentAmount+amount > totalAmount && spentAmount+amount <= limit {
		return fmt.Errorf("warning: spending mendekati batas anggaran (used: %.2f, total: %.2f)", spentAmount+amount, totalAmount)
	}
	if spentAmount+amount > limit {
		return fmt.Errorf("overspend: total pengeluaran melebihi batas (used: %.2f, req: %.2f, total: %.2f)",
			spentAmount, amount, totalAmount)
	}

	return nil
}

// ============================================================
// 🔹 GET /budgets/:id/transactions — audit detail per budget
// ============================================================
func GetBudgetTransactions(c *gin.Context) {
	ctx := c.Request.Context()
	assetID := strings.TrimSpace(c.Query("asset_id"))

	var query string
	var args []any

	if assetID != "" {
		query = `
			SELECT 
				bt.id, bt.budget_id, bt.asset_id, bt.amount, bt.category,
				bt.currency, bt.transaction_date, bt.notes, bt.created_by
			FROM budget_transactions bt
			WHERE bt.asset_id = $1
			ORDER BY bt.transaction_date DESC;
		`
		args = append(args, assetID)
	} else {
		query = `
			SELECT 
				bt.id, bt.budget_id, bt.asset_id, bt.amount, bt.category,
				bt.currency, bt.transaction_date, bt.notes, bt.created_by
			FROM budget_transactions bt
			ORDER BY bt.transaction_date DESC LIMIT 50;
		`
	}

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Tx struct {
		ID            int64     `json:"id"`
		BudgetID      int64     `json:"budget_id"`
		AssetID       *int64    `json:"asset_id"`
		Amount        float64   `json:"amount"`
		Category      string    `json:"category"`
		Currency      string    `json:"currency"`
		TransactionAt time.Time `json:"transaction_date"`
		Notes         string    `json:"notes"`
		CreatedBy     *int64    `json:"created_by"`
	}

	var list []Tx
	for rows.Next() {
		var t Tx
		if err := rows.Scan(
			&t.ID, &t.BudgetID, &t.AssetID, &t.Amount,
			&t.Category, &t.Currency, &t.TransactionAt, &t.Notes, &t.CreatedBy,
		); err == nil {
			list = append(list, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": list,
		"count":        len(list),
	})
}

// ============================================================
// 🔹 GET /budgets/report — ringkasan realisasi per cost center
// ============================================================
func GetBudgetReport(c *gin.Context) {
	query := `
		SELECT 
			COALESCE(cc.code || ' — ' || cc.name, '-') AS cost_center,
			b.category,
			DATE_TRUNC('month', bt.transaction_date) AS periode,
			SUM(bt.amount) AS total_spent
		FROM budgets b
		JOIN budget_transactions bt ON bt.budget_id = b.id
		LEFT JOIN cost_centers cc ON b.cost_center_id = cc.id
		WHERE b.deleted_at IS NULL
		GROUP BY cc.code, cc.name, b.category, DATE_TRUNC('month', bt.transaction_date)
		ORDER BY cc.code, periode;
	`

	rows, err := database.Pool.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil laporan anggaran", "detail": err.Error()})
		return
	}
	defer rows.Close()

	type ReportRow struct {
		CostCenter string    `json:"cost_center"`
		Category   *string   `json:"category,omitempty"`
		Periode    time.Time `json:"periode"`
		TotalSpent float64   `json:"total_spent"`
	}

	var list []ReportRow
	for rows.Next() {
		var r ReportRow
		if err := rows.Scan(&r.CostCenter, &r.Category, &r.Periode, &r.TotalSpent); err == nil {
			list = append(list, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"report": list})
}

// DELETE /budgets/transactions/:id  — untuk reversal transaksi
func DeleteBudgetTransaction(c *gin.Context) {
	txID := c.Param("id")

	var budgetID int64
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT budget_id FROM budget_transactions WHERE id=$1`, txID).Scan(&budgetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
		return
	}

	_, err = database.Pool.Exec(c.Request.Context(),
		`DELETE FROM budget_transactions WHERE id=$1`, txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus transaksi"})
		return
	}

	_ = RecalculateBudgetTotals(c.Request.Context(), budgetID)
	middleware.LogAction(c, "budget_transactions", mustAtoi64(txID), "DELETE", nil)
	c.JSON(http.StatusOK, gin.H{"message": "Transaksi dihapus"})
}

// Ringkasan realisasi anggaran (CAPEX vs OPEX)
// GetBudgetSummary menampilkan ringkasan realisasi anggaran (CAPEX vs OPEX)
// sesuai ISO/IEC 19770-10:2025 (Financial Governance & Utilization)
func GetBudgetSummary(c *gin.Context) {
	query := `
	SELECT 
		b.id,
		b.name,
		b.category,
		b.currency,
		COALESCE(SUM(bt.amount),0) AS realized_amount,
		b.total_amount,
		(b.total_amount - COALESCE(SUM(bt.amount),0)) AS remaining,
		CASE 
			WHEN b.total_amount > 0 THEN ROUND((COALESCE(SUM(bt.amount),0) / b.total_amount) * 100, 2)
			ELSE 0 
		END AS utilization_percent,
		COUNT(DISTINCT CASE WHEN bt.entity_type='asset' THEN bt.entity_id END) AS asset_count,
		COUNT(DISTINCT CASE WHEN bt.entity_type='license' THEN bt.entity_id END) AS license_count,
		COUNT(DISTINCT CASE WHEN bt.entity_type='contract' THEN bt.entity_id END) AS contract_count
	FROM budgets b
	LEFT JOIN budget_transactions bt ON bt.budget_id = b.id
	WHERE b.deleted_at IS NULL
	GROUP BY b.id
	ORDER BY b.name;`

	rows, err := database.Pool.Query(c.Request.Context(), query)
	if err != nil {
		log.Printf("[BUDGET_SUMMARY_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil ringkasan anggaran"})
		return
	}
	defer rows.Close()

	type BudgetSummary struct {
		ID                 int64   `json:"id"`
		Name               string  `json:"name"`
		Category           *string `json:"category"`
		Currency           *string `json:"currency"`
		TotalAmount        float64 `json:"total_amount"`
		RealizedAmount     float64 `json:"realized_amount"`
		Remaining          float64 `json:"remaining"`
		UtilizationPercent float64 `json:"utilization_percent"`
		AssetCount         int     `json:"asset_count"`
		LicenseCount       int     `json:"license_count"`
		ContractCount      int     `json:"contract_count"`
	}

	var list []BudgetSummary
	for rows.Next() {
		var r BudgetSummary
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Category, &r.Currency,
			&r.RealizedAmount, &r.TotalAmount, &r.Remaining,
			&r.UtilizationPercent, &r.AssetCount, &r.LicenseCount, &r.ContractCount,
		); err == nil {
			list = append(list, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": list,
		"message": "Budget summary retrieved successfully (ISO/IEC 19770-10 compliant)",
	})
}

// GET /budgets/dashboard
func GetBudgetDashboard(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `SELECT * FROM v_budget_overview ORDER BY budget_name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch budget dashboard"})
		return
	}
	defer rows.Close()

	type rec struct {
		BudgetID        int64   `json:"budget_id"`
		BudgetName      string  `json:"budget_name"`
		Category        *string `json:"category"`
		Currency        *string `json:"currency"`
		TotalAmount     float64 `json:"total_amount"`
		RealizedAmount  float64 `json:"realized_amount"`
		RemainingAmount float64 `json:"remaining_amount"`
		RealizationPct  float64 `json:"realization_percent"`
		Status          string  `json:"status"`
	}
	var list []rec
	for rows.Next() {
		var r rec
		rows.Scan(&r.BudgetID, &r.BudgetName, &r.Category, &r.Currency,
			&r.TotalAmount, &r.RealizedAmount, &r.RemainingAmount, &r.RealizationPct, &r.Status)
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"dashboard": list})
}

func GetBudgetOverview(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT budget_id, budget_name, category, currency,
		        total_amount, realized_amount, remaining_amount,
		        realization_percent, status
		   FROM v_budget_overview
		   ORDER BY budget_name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch budget overview"})
		return
	}
	defer rows.Close()

	type Overview struct {
		BudgetID           int64   `json:"budget_id"`
		BudgetName         string  `json:"budget_name"`
		Category           *string `json:"category"`
		Currency           *string `json:"currency"`
		TotalAmount        float64 `json:"total_amount"`
		RealizedAmount     float64 `json:"realized_amount"`
		RemainingAmount    float64 `json:"remaining_amount"`
		RealizationPercent float64 `json:"realization_percent"`
		Status             string  `json:"status"`
	}
	var list []Overview
	for rows.Next() {
		var r Overview
		_ = rows.Scan(&r.BudgetID, &r.BudgetName, &r.Category, &r.Currency,
			&r.TotalAmount, &r.RealizedAmount, &r.RemainingAmount,
			&r.RealizationPercent, &r.Status)
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"overview": list})
}

func GetBudgetByID(c *gin.Context) {
	id := c.Param("id")

	var b struct {
		ID           int64      `json:"id"`
		Name         string     `json:"name"`
		DepartmentID *int64     `json:"department_id"`
		TotalAmount  float64    `json:"total_amount"`
		StartDate    *time.Time `json:"start_date"`
		EndDate      *time.Time `json:"end_date"`
		CostCenterID *int64     `json:"cost_center_id"`
		CostCenter   *string    `json:"cost_center"`
		Category     *string    `json:"category"`
		Currency     *string    `json:"currency"`
		CreatedAt    *time.Time `json:"created_at"`
	}

	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT 
			b.id,
			b.name,
			b.department_id,
			b.total_amount,
			b.start_date,
			b.end_date,
			b.cost_center_id,
			COALESCE(cc.code || ' — ' || cc.name, '-') AS cost_center,
			b.category,
			b.currency,
			b.created_at
		FROM budgets b
		LEFT JOIN cost_centers cc ON cc.id = b.cost_center_id
		WHERE b.id = $1 AND b.deleted_at IS NULL
	`, id).Scan(
		&b.ID, &b.Name, &b.DepartmentID, &b.TotalAmount,
		&b.StartDate, &b.EndDate, &b.CostCenterID, &b.CostCenter,
		&b.Category, &b.Currency, &b.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"budget": b})
}
