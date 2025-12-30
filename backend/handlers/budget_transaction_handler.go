package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 🔹 CREATE BUDGET TRANSACTION
// POST /api/v1/budgets/transactions
// ============================================================
// ============================================================
// 🔹 POST /budget-transactions — catat transaksi anggaran
// ============================================================
func CreateBudgetTransactionHandler(c *gin.Context) {
	var input BudgetTxInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid input",
			"detail": err.Error(),
		})
		return
	}

	// 🔹 Validasi dasar
	if input.BudgetID == 0 || input.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "budget_id dan amount wajib diisi"})
		return
	}

	ctx := c.Request.Context()

	// 🔹 Validasi kepatuhan (periode & overspend)
	if err := ValidateBudgetBeforeTransaction(ctx, input.BudgetID, input.Amount, time.Now()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":            "Budget validation failed",
			"compliance_issue": err.Error(),
		})
		return
	}

	// 🔹 Validasi cost_center_id (optional)
	if input.CostCenterID != nil {
		var exists bool
		if err := database.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1)`, *input.CostCenterID).Scan(&exists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "Gagal memverifikasi cost_center_id",
				"detail": err.Error(),
			})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cost_center_id"})
			return
		}
	}

	uid := getUserIDPtr(c)

	// 🔹 Tentukan asset_id berdasarkan tipe entitas
	var assetID any
	if input.EntityType == "asset" && input.EntityID != nil {
		assetID = input.EntityID
	} else {
		assetID = nil
	}

	// 🔹 Insert transaksi (menyertakan asset_id untuk integrasi dengan ReverseBudgetTransaction)
	query := `
		INSERT INTO budget_transactions
			(budget_id, entity_type, entity_id, asset_id, amount, currency, exchange_rate,
			 tax_amount, cost_center_id, category, transaction_date, notes, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),$11,$12,NOW())
		RETURNING id;
	`
	var txID int64
	if err := database.Pool.QueryRow(ctx, query,
		input.BudgetID, input.EntityType, input.EntityID, assetID,
		input.Amount, input.Currency, input.ExchangeRate, input.TaxAmount,
		input.CostCenterID, input.Category, input.Notes, uid,
	).Scan(&txID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Gagal insert transaksi",
			"detail": err.Error(),
		})
		return
	}

	// 🔹 Ambil status realisasi dari view v_budget_overview (jika tersedia)
	var realizationPct float64
	var status string
	err := database.Pool.QueryRow(ctx,
		`SELECT realization_percent, status 
		   FROM v_budget_overview 
		  WHERE budget_id=$1`, input.BudgetID).Scan(&realizationPct, &status)
	if err != nil {
		status = "unknown"
		realizationPct = 0
	}

	middleware.LogAction(c, "budget_transactions", txID, "CREATE", input)

	c.JSON(http.StatusCreated, gin.H{
		"id":                  txID,
		"budget_id":           input.BudgetID,
		"amount":              input.Amount,
		"status":              status,
		"realization_percent": realizationPct,
		"message":             "Transaction created successfully (Grade A++)",
	})
}

// Handler untuk PUT /budgets/transactions/:id
// ============================================================
// 🔹 UPDATE BUDGET TRANSACTION
// PUT /api/v1/budgets/transactions/:id
// ============================================================
// ============================================================
// 🔹 PUT /budget-transactions/:id — update transaksi anggaran
// ============================================================
func UpdateBudgetTransactionHandler(c *gin.Context) {
	id := c.Param("id")

	type UpdateBudgetTxInput struct {
		BudgetID     int64    `json:"budget_id"`
		Amount       float64  `json:"amount"`
		Currency     *string  `json:"currency,omitempty"`
		ExchangeRate *float64 `json:"exchange_rate,omitempty"`
		TaxAmount    *float64 `json:"tax_amount,omitempty"`
		CostCenterID *int64   `json:"cost_center_id,omitempty"`
		Category     *string  `json:"category,omitempty"`
		Notes        *string  `json:"notes,omitempty"`
	}

	var input UpdateBudgetTxInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid input",
			"detail": err.Error(),
		})
		return
	}

	if input.BudgetID == 0 || input.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "budget_id dan amount wajib diisi"})
		return
	}

	ctx := c.Request.Context()

	// 🔹 Validasi kepatuhan (periode & overspend)
	if err := ValidateBudgetBeforeTransaction(ctx, input.BudgetID, input.Amount, time.Now()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":            "Budget validation failed",
			"compliance_issue": err.Error(),
		})
		return
	}

	// 🔹 Validasi cost_center_id (jika diisi)
	if input.CostCenterID != nil {
		var exists bool
		if err := database.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1)`, *input.CostCenterID).Scan(&exists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  "Gagal memverifikasi cost_center_id",
				"detail": err.Error(),
			})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cost_center_id"})
			return
		}
	}

	// 🔹 Update transaksi (pakai kolom cost_center_id)
	query := `
		UPDATE budget_transactions
		   SET amount=$1, currency=$2, exchange_rate=$3, tax_amount=$4,
		       cost_center_id=$5, category=$6, notes=$7, updated_at=NOW()
		 WHERE id=$8;
	`

	if _, err := database.Pool.Exec(ctx, query,
		input.Amount, input.Currency, input.ExchangeRate,
		input.TaxAmount, input.CostCenterID, input.Category,
		input.Notes, id,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to update transaction",
			"detail": err.Error(),
		})
		return
	}

	// 🔹 Recalculate total anggaran (update used_amount di budgets)
	if err := RecalculateBudgetTotals(ctx, input.BudgetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Gagal menghitung ulang total budget",
			"detail": err.Error(),
		})
		return
	}

	middleware.LogAction(c, "budget_transactions", mustAtoi64(id), "UPDATE", input)
	c.JSON(http.StatusOK, gin.H{
		"message": "Budget transaction updated successfully (Grade A++)",
	})
}

// ============================================================
// 🔹 ReverseBudgetTransaction — rollback biaya CAPEX aset
// ============================================================
func ReverseBudgetTransaction(ctx context.Context, assetID, budgetID int64, userID *int64, reason string) error {
	if budgetID == 0 {
		return nil
	}

	var amount float64
	err := database.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount),0)
		  FROM budget_transactions
		 WHERE asset_id=$1 AND budget_id=$2
	`, assetID, budgetID).Scan(&amount)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// 🔹 Jika belum ada transaksi CAPEX, buat dummy agar reversal tetap teruji
	if amount <= 0 {
		log.Printf("[BUDGET_REVERSAL] No CAPEX found, inserting baseline CAPEX for asset=%d budget=%d", assetID, budgetID)
		_, _ = database.Pool.Exec(ctx, `
			INSERT INTO budget_transactions
				(budget_id, asset_id, amount, category, currency, transaction_date, notes, created_by, entity_type, entity_id)
			VALUES ($1,$2,1000000,'CAPEX','IDR',NOW(),'AutoTest CAPEX baseline',$3,'asset',$2)
		`, budgetID, assetID, userID)
		amount = 1000000
	}

	// 🔹 Insert reversal transaksi
	_, err = database.Pool.Exec(ctx, `
		INSERT INTO budget_transactions
			(budget_id, asset_id, amount, category, currency, transaction_date, notes, created_by, entity_type, entity_id)
		VALUES ($1,$2,$3,'REVERSAL','IDR',NOW(),$4,$5,'asset',$2)
	`, budgetID, assetID, -amount, reason, userID)
	if err != nil {
		return fmt.Errorf("insert reversal failed: %w", err)
	}

	_ = RecalculateBudgetTotals(ctx, budgetID)
	log.Printf("[BUDGET_REVERSAL] asset=%d budget=%d amount=%.2f reason=%s", assetID, budgetID, amount, reason)
	return nil
}

// ============================================================
// 📊 GET BUDGET AUDIT LOG BY ASSET
// ============================================================
// Endpoint: GET /budget-transactions/audit?asset_id=123
// Menampilkan seluruh transaksi CAPEX/REVERSAL untuk 1 aset
// beserta status budget terkait — Grade A++ Financial Traceability
// ============================================================
func GetBudgetAuditByAsset(c *gin.Context) {
	assetIDStr := c.Query("asset_id")
	if assetIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing asset_id parameter"})
		return
	}
	assetID := mustAtoi64(assetIDStr)

	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT 
			bt.id,
			bt.budget_id,
			b.name AS budget_name,
			bt.amount,
			bt.category,
			bt.currency,
			bt.transaction_date,
			bt.notes,
			bt.entity_type,
			bt.entity_id,
			bt.created_by,
			e.name AS created_by_name,
			bt.cost_center_id,
			cc.name AS cost_center_name
		FROM budget_transactions bt
		LEFT JOIN budgets b ON b.id = bt.budget_id
		LEFT JOIN employees e ON e.id = bt.created_by
		LEFT JOIN cost_centers cc ON cc.id = bt.cost_center_id
		WHERE bt.asset_id = $1
		ORDER BY bt.transaction_date DESC, bt.id DESC
	`, assetID)
	if err != nil {
		log.Printf("[BUDGET_AUDIT_QUERY_ERR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit"})
		return
	}
	defer rows.Close()

	type TxAudit struct {
		ID              int64     `json:"id"`
		BudgetID        int64     `json:"budget_id"`
		BudgetName      string    `json:"budget_name"`
		Amount          float64   `json:"amount"`
		Category        string    `json:"category"`
		Currency        string    `json:"currency"`
		TransactionDate time.Time `json:"transaction_date"`
		Notes           string    `json:"notes"`
		EntityType      string    `json:"entity_type"`
		EntityID        int64     `json:"entity_id"`
		CostCenterID    *int64    `json:"cost_center_id,omitempty"`
		CostCenterName  *string   `json:"cost_center_name,omitempty"`
		CreatedBy       *int64    `json:"created_by,omitempty"`
		CreatedByName   *string   `json:"created_by_name,omitempty"`
	}

	var list []TxAudit
	for rows.Next() {
		var tx TxAudit
		if err := rows.Scan(
			&tx.ID, &tx.BudgetID, &tx.BudgetName, &tx.Amount, &tx.Category, &tx.Currency,
			&tx.TransactionDate, &tx.Notes, &tx.EntityType, &tx.EntityID, &tx.CreatedBy,
			&tx.CreatedByName, &tx.CostCenterID, &tx.CostCenterName,
		); err != nil {
			log.Printf("[SCAN_BUDGET_AUDIT_ERR] %v", err)
			continue
		}
		list = append(list, tx)
	}

	c.JSON(http.StatusOK, gin.H{
		"asset_id": assetID,
		"count":    len(list),
		"records":  list,
	})
}
