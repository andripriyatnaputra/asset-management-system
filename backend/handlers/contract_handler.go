package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
)

//
// ===========================================================
// Contract Management Handler (ISO/IEC 19770-10:2025)
// ===========================================================
//

// ============================================================
// 📄 CREATE CONTRACT (Grade A ++)
// ============================================================
func CreateContract(c *gin.Context) {
	type Req struct {
		ContractNumber string   `json:"contract_number" binding:"required"`
		Vendor         *string  `json:"vendor,omitempty"`
		ContractType   *string  `json:"contract_type,omitempty"`
		StartDate      string   `json:"start_date" binding:"required"`
		EndDate        string   `json:"end_date,omitempty"`
		TotalValue     *float64 `json:"total_value,omitempty"`
		Currency       *string  `json:"currency,omitempty"`
		PaymentTerms   *string  `json:"payment_terms,omitempty"`
		ContactPerson  *string  `json:"contact_person,omitempty"`
		ContactEmail   *string  `json:"contact_email,omitempty"`
		AttachmentURL  *string  `json:"attachment_url,omitempty"`
		Notes          *string  `json:"notes,omitempty"`
		Status         *string  `json:"status,omitempty"`
		BudgetID       *int64   `json:"budget_id,omitempty"`
		CostCenterID   *int64   `json:"cost_center_id,omitempty"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔹 Validasi format tanggal start/end (mendukung date & RFC3339)
	parseDate := func(s string) *time.Time {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		for _, layout := range []string{"2006-01-02", time.RFC3339} {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
		return nil
	}
	start := parseDate(req.StartDate)
	if start == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date"})
		return
	}
	var end *time.Time
	if req.EndDate != "" {
		end = parseDate(req.EndDate)
		if end == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
			return
		}
		if end.Before(*start) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end_date cannot be before start_date"})
			return
		}
	}

	// 🔹 Default currency & status
	curr := "IDR"
	if req.Currency != nil && *req.Currency != "" {
		curr = *req.Currency
	}
	status := "active"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	// 🔹 Cek unique contract_number (soft-deleted tidak dihitung)
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM contracts WHERE contract_number=$1 AND deleted_at IS NULL)`,
		req.ContractNumber,
	).Scan(&exists)
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "contract_number already exists"})
		return
	}

	// 🔹 Validasi cost_center_id (jika diisi)
	if req.CostCenterID != nil {
		var ccExists bool
		_ = database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1)`, *req.CostCenterID,
		).Scan(&ccExists)
		if !ccExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cost_center not found"})
			return
		}
	}

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO contracts
		  (contract_number,vendor,contract_type,start_date,end_date,
		   total_value,currency,payment_terms,contact_person,contact_email,
		   attachment_url,notes,status,budget_id,cost_center_id,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW(),NOW())
		RETURNING id`,
		req.ContractNumber, req.Vendor, req.ContractType, start, end,
		req.TotalValue, curr, req.PaymentTerms, req.ContactPerson, req.ContactEmail,
		req.AttachmentURL, req.Notes, status, req.BudgetID, req.CostCenterID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 🔹 Auto-CAPEX transaction (jika punya budget & total_value)
	if req.BudgetID != nil && req.TotalValue != nil && *req.TotalValue > 0 {
		// ambil cost_center_id dari request atau fallback dari budget
		var ccIDPtr *int64
		if req.CostCenterID != nil && *req.CostCenterID > 0 {
			ccIDPtr = req.CostCenterID
		} else {
			_ = database.Pool.QueryRow(c.Request.Context(),
				`SELECT cost_center_id FROM budgets WHERE id=$1`, *req.BudgetID,
			).Scan(&ccIDPtr)
		}

		if err := ValidateBudgetBeforeTransaction(c, *req.BudgetID, *req.TotalValue, time.Now()); err == nil {
			note := fmt.Sprintf("Kontrak %s (CAPEX)", req.ContractNumber)
			currCode := ptrOr(req.Currency, "IDR")

			_ = CreateBudgetTransaction(c, BudgetTxInput{
				BudgetID:     *req.BudgetID,
				EntityType:   "contract",
				EntityID:     &id,
				Amount:       *req.TotalValue,
				Category:     strPtr("CAPEX"),
				Currency:     &currCode,
				CostCenterID: ccIDPtr,
				Notes:        note,
				CreatedBy:    getUserIDPtr(c),
			})
			_ = RecalculateBudgetTotals(c.Request.Context(), *req.BudgetID)
		}
	}

	middleware.LogAction(c, "contracts", id, "CREATE", req)
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Contract created (Grade A ++)"})
}

// ============================================================
// 📊 GET ALL CONTRACTS (+ health & governance)
// ============================================================
func GetAllContracts(c *gin.Context) {
	// optional: ?alert=1 untuk mengaktifkan broadcast alert expired
	alertFlag := c.Query("alert") == "1"

	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT id,contract_number,vendor,contract_type,start_date,end_date,
		       total_value,currency,status,budget_id,created_at
		  FROM contracts 
		  WHERE deleted_at IS NULL
		  ORDER BY start_date DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID             int64      `json:"id"`
		ContractNumber string     `json:"contract_number"`
		Vendor         *string    `json:"vendor,omitempty"`
		Type           *string    `json:"contract_type,omitempty"`
		StartDate      *time.Time `json:"start_date"`
		EndDate        *time.Time `json:"end_date"`
		TotalValue     *float64   `json:"total_value,omitempty"`
		Currency       *string    `json:"currency,omitempty"`
		Status         *string    `json:"status,omitempty"`
		BudgetID       *int64     `json:"budget_id,omitempty"`
		CreatedAt      *time.Time `json:"created_at,omitempty"`
		HealthScore    float64    `json:"contract_health_score"`
		GovScore       float64    `json:"governance_score"`
		Expired        bool       `json:"expired"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.ContractNumber, &r.Vendor, &r.Type, &r.StartDate,
			&r.EndDate, &r.TotalValue, &r.Currency, &r.Status, &r.BudgetID, &r.CreatedAt)

		// Durasi & health score
		if r.EndDate != nil && r.StartDate != nil {
			total := r.EndDate.Sub(*r.StartDate).Hours() / 24
			left := time.Until(*r.EndDate).Hours() / 24
			if total > 0 {
				r.HealthScore = math.Max(0, (left/total)*100)
			}
			r.Expired = left < 0
		}

		// Governance score
		r.GovScore = governanceScore(r.BudgetID != nil, true, r.EndDate != nil)

		// Alert kedaluwarsa hanya jika diminta (menghindari nondeterminism saat regression)
		if r.Expired && alertFlag {
			msg := fmt.Sprintf("Kontrak %s telah kedaluwarsa", r.ContractNumber)
			services.BroadcastAlert(msg, "warning")
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// 🔍 GET CONTRACT BY ID (+ health/governance)
// ============================================================
func GetContractByID(c *gin.Context) {
	id := c.Param("id")
	var r models.Contract
	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT id,contract_number,vendor,contract_type,start_date,end_date,
		       total_value,currency,payment_terms,contact_person,contact_email,
		       attachment_url,notes,status,budget_id,created_at,updated_at
		  FROM contracts WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&r.ID, &r.ContractNumber, &r.Vendor, &r.ContractType, &r.StartDate, &r.EndDate,
			&r.TotalValue, &r.Currency, &r.PaymentTerms, &r.ContactPerson, &r.ContactEmail,
			&r.AttachmentURL, &r.Notes, &r.Status, &r.BudgetID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var health float64
	if r.EndDate != nil && r.StartDate != nil {
		total := r.EndDate.Sub(*r.StartDate).Hours() / 24
		left := time.Until(*r.EndDate).Hours() / 24
		if total > 0 {
			health = math.Max(0, (left/total)*100)
		}
	}
	gov := governanceScore(r.BudgetID != nil, true, r.EndDate != nil)
	c.JSON(http.StatusOK, gin.H{"contract": r, "contract_health_score": health, "governance_score": gov})
}

// ============================================================
// ✏️ UPDATE CONTRACT (+ basic validation)
// ============================================================
func UpdateContract(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		EndDate *string  `json:"end_date,omitempty"`
		Total   *float64 `json:"total_value,omitempty"`
		Status  *string  `json:"status,omitempty"`
		Notes   *string  `json:"notes,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	set, args := []string{}, []any{}
	i := 1

	// end_date: dukung dua format umum
	if body.EndDate != nil {
		var parsed *time.Time
		for _, layout := range []string{"2006-01-02", time.RFC3339} {
			if t, err := time.Parse(layout, *body.EndDate); err == nil {
				parsed = &t
				break
			}
		}
		if parsed == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
			return
		}
		set = append(set, fmt.Sprintf("end_date=$%d", i))
		args = append(args, *parsed)
		i++
	}
	if body.Total != nil {
		set = append(set, fmt.Sprintf("total_value=$%d", i))
		args = append(args, *body.Total)
		i++
	}
	if body.Status != nil {
		set = append(set, fmt.Sprintf("status=$%d", i))
		args = append(args, *body.Status)
		i++
	}
	if body.Notes != nil {
		set = append(set, fmt.Sprintf("notes=$%d", i))
		args = append(args, *body.Notes)
		i++
	}
	if len(set) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields"})
		return
	}

	query := fmt.Sprintf(`UPDATE contracts SET %s,updated_at=NOW() WHERE id=$%d AND deleted_at IS NULL`, strings.Join(set, ","), i)
	args = append(args, id)
	_, err := database.Pool.Exec(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	middleware.LogAction(c, "contracts", mustAtoi64(id), "UPDATE", body)
	c.JSON(http.StatusOK, gin.H{"message": "Contract updated"})
}

// ============================================================
// 🗑 DELETE CONTRACT (soft + budget reversal & linkage guard)
// ============================================================
func DeleteContract(c *gin.Context) {
	idStr := c.Param("id")
	num := mustAtoi64(idStr)

	// 🔹 Cek apakah masih dipakai oleh asset atau license (hard block)
	var assetCount, licCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM assets WHERE contract_id=$1 AND deleted_at IS NULL`, num,
	).Scan(&assetCount)
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM licenses WHERE contract_id=$1 AND deleted_at IS NULL`, num,
	).Scan(&licCount)

	if assetCount > 0 || licCount > 0 {
		middleware.LogAction(c, "contracts", num, "DELETE_BLOCKED", gin.H{
			"assets_linked":   assetCount,
			"licenses_linked": licCount,
		})
		c.JSON(http.StatusForbidden, gin.H{
			"error":           "cannot delete contract; assets and/or licenses still linked",
			"assets_linked":   assetCount,
			"licenses_linked": licCount,
		})
		return
	}

	// Validate real existence BEFORE scanning detailed fields
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM contracts WHERE id=$1)`, num,
	).Scan(&exists)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "contract not found in database"})
		return
	}

	// 🔹 Ambil data untuk reversal
	var (
		budgetID     *int64
		total        *float64
		number       *string
		curr         *string
		costCenterID *int64
	)
	err := database.Pool.QueryRow(c.Request.Context(),
		`SELECT budget_id,total_value,contract_number,currency,cost_center_id 
		   FROM contracts
		  WHERE id=$1 AND deleted_at IS NULL`, num).
		Scan(&budgetID, &total, &number, &curr, &costCenterID)
	c.JSON(http.StatusNotFound, gin.H{
		"error": "contract not found or already deleted",
	})

	// 🔹 Soft delete
	_, err = database.Pool.Exec(c.Request.Context(),
		`UPDATE contracts SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, num)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "contracts", num, "DELETE", nil)

	// 🔹 Reverse CAPEX di budget (jika applicable)
	if budgetID != nil && total != nil && *total > 0 {
		note := fmt.Sprintf("Pembatalan kontrak %s (reversal CAPEX)", ptrOr(number, fmt.Sprintf("ID %d", num)))

		var ccIDPtr *int64
		if costCenterID != nil && *costCenterID > 0 {
			ccIDPtr = costCenterID
		}

		_ = CreateBudgetTransaction(c, BudgetTxInput{
			BudgetID:     *budgetID,
			EntityType:   "contract",
			EntityID:     &num,
			Amount:       -1 * *total,
			Category:     strPtr("CAPEX"),
			Currency:     curr,
			CostCenterID: ccIDPtr,
			Notes:        note,
			CreatedBy:    getUserIDPtr(c),
		})
		_ = RecalculateBudgetTotals(c.Request.Context(), *budgetID)

		// Alert dihapus dari flow reguler regression; kalaupun tetap ingin, bisa ditrigger dari UI saja.
		services.BroadcastAlert(note, "warning")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contract deleted"})
}

// ============================================================
// 🔗 GET LICENSES UNDER CONTRACT
// ============================================================
func GetLicensesByContract(c *gin.Context) {
	id := c.Param("id")
	rows, err := database.Pool.Query(context.Background(), `
		SELECT id,name,vendor,license_type,license_model,total_seats,cost,expiration_date,compliance_status
		  FROM licenses WHERE contract_id=$1 AND deleted_at IS NULL ORDER BY name`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Row struct {
		ID         int64      `json:"id"`
		Name       string     `json:"name"`
		Vendor     *string    `json:"vendor"`
		Type       *string    `json:"license_type"`
		Model      *string    `json:"license_model"`
		Seats      int        `json:"total_seats"`
		Cost       *float64   `json:"cost"`
		ExpireDate *time.Time `json:"expiration_date"`
		Status     *string    `json:"compliance_status"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.Name, &r.Vendor, &r.Type, &r.Model, &r.Seats, &r.Cost, &r.ExpireDate, &r.Status)
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"licenses": list})
}
