package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
)

// ============================================================
// 🧩 Enforce Governance & Compliance (Universal Service)
// ============================================================
// Dapat dipanggil dari:
// - asset_handler.go → services.EnforceGovernanceAndCompliance(c.Request.Context(), id)
// - compliance_watcher.go → EnforceGovernanceAndCompliance(ctx, id)
func EnforceGovernanceAndCompliance(ctx context.Context, assetID int64) (bool, string) {
	var asset struct {
		ID               int64
		Name             string
		BudgetID         *int64
		ContractID       *int64
		LicenseID        *int64
		DepartmentID     *int64
		CostCenterID     *int64
		OwnershipType    *string
		AcquisitionType  *string
		Depreciation     *string
		InitialPrice     *float64
		AssetCriticality *string
		LifecycleStage   *string
	}

	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, budget_id, contract_id, license_id, department_id, cost_center_id,
		       ownership_type, acquisition_type, depreciation_method, initial_price,
		       asset_criticality, lifecycle_stage
		  FROM assets
		 WHERE id=$1 AND deleted_at IS NULL
	`, assetID).Scan(
		&asset.ID, &asset.Name,
		&asset.BudgetID, &asset.ContractID, &asset.LicenseID,
		&asset.DepartmentID, &asset.CostCenterID,
		&asset.OwnershipType, &asset.AcquisitionType, &asset.Depreciation,
		&asset.InitialPrice, &asset.AssetCriticality, &asset.LifecycleStage,
	)
	if err != nil {
		log.Printf("[ENFORCE_ERR] failed to load asset %d: %v", assetID, err)
		return false, "unverified"
	}

	// ============================================================
	// 🔹 Governance Score Calculation
	// ============================================================
	score := CalculateGovernanceScore(ctx, assetID)

	if asset.BudgetID != nil {
		score += 10
	}
	if asset.ContractID != nil {
		score += 10
	}
	if asset.LicenseID != nil {
		score += 5
	}
	if asset.DepartmentID != nil {
		score += 5
	}
	if asset.CostCenterID != nil {
		score += 5
	}
	if asset.OwnershipType != nil && *asset.OwnershipType == "company_owned" {
		score += 5
	}
	if asset.AcquisitionType != nil && *asset.AcquisitionType == "purchase" {
		score += 5
	}
	if asset.Depreciation != nil && *asset.Depreciation != "none" {
		score += 5
	}
	if asset.InitialPrice != nil && *asset.InitialPrice > 0 {
		score += 5
	}
	if score > 100 {
		score = 100
	}

	// ============================================================
	// 🔹 Compliance Logic
	// ============================================================
	isCompliant, note := ValidateAssetGovernance(models.Asset{
		ID:               asset.ID,
		Name:             asset.Name,
		BudgetID:         asset.BudgetID,
		ContractID:       asset.ContractID,
		LicenseID:        asset.LicenseID,
		LifecycleStage:   asset.LifecycleStage,
		AssetCriticality: asset.AssetCriticality,
	})
	if isCompliant && score < 70 {
		isCompliant = false
		note = "Governance score below threshold"
	}

	// ============================================================
	// 🔹 Update Asset Record
	// ============================================================
	_, err = database.Pool.Exec(ctx, `
		UPDATE assets
		   SET compliance_flag=$1,
		       compliance_note=$2,
		       governance_score=$3,
		       updated_at=NOW()
		 WHERE id=$4
	`, isCompliant, note, score, assetID)
	if err != nil {
		log.Printf("[ENFORCE_UPDATE_ERR] asset_id=%d err=%v", assetID, err)
	}

	// ============================================================
	// 🔹 Broadcast & Logging
	// ============================================================
	status := "info"
	if !isCompliant {
		status = "warning"
	}

	go BroadcastAlert(
		fmt.Sprintf("📊 Asset #%d (%s): %s (score %.1f)",
			asset.ID, asset.Name,
			ternary(isCompliant, "Compliant", "Non-Compliant"),
			score),
		status,
	)

	if !isCompliant {
		msg := fmt.Sprintf("Asset #%d (%s) non-compliant: %s",
			asset.ID, asset.Name, note)
		BroadcastAssetGovernanceAlert(ctx, asset.ID, asset.Name, msg, score)
	}

	return isCompliant, note
}

// ============================================================
// ⏱️ BACKGROUND WATCHER: Compliance & Financial Consistency
// ============================================================
func RunAssetComplianceWatcher(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	log.Println("[WATCHER] Asset Compliance Watcher started (interval 6h)")
	for {
		select {
		case <-ctx.Done():
			log.Println("[WATCHER] Stopping Compliance Watcher...")
			return
		case <-ticker.C:
			log.Println("[WATCHER] Running scheduled compliance & budget check...")
			RecheckAssetCompliance(ctx)
			ReconcileBudgetIntegrity(ctx)
		}
	}
}

func RecheckAssetCompliance(ctx context.Context) {
	rows, err := database.Pool.Query(ctx, `
		SELECT id 
		  FROM assets 
		 WHERE (compliance_flag IS NULL OR compliance_flag = false)
		   AND deleted_at IS NULL
	`)
	if err != nil {
		log.Printf("[WATCHER_ERR] Failed to query assets: %v", err)
		return
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		isCompliant, note := EnforceGovernanceAndCompliance(ctx, id)
		if !isCompliant {
			database.Pool.Exec(ctx, `
				INSERT INTO alerts (message, severity, category)
				VALUES ($1, 'warning', 'asset_compliance')
			`, note)
		}
		count++
	}
	log.Printf("[WATCHER] Compliance recheck done. %d assets processed.", count)
}

func ReconcileBudgetIntegrity(ctx context.Context) {
	rows, err := database.Pool.Query(ctx, `
		SELECT b.id, b.total_amount,
		       COALESCE(SUM(bt.amount),0) AS total_tx
		  FROM budgets b
		  LEFT JOIN budget_transactions bt ON bt.budget_id=b.id
		 WHERE b.deleted_at IS NULL
		 GROUP BY b.id, b.total_amount
		 HAVING ABS(COALESCE(SUM(bt.amount),0)) > b.total_amount * 0.05
	`)
	if err != nil {
		log.Printf("[WATCHER_ERR] Budget reconciliation failed: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var total, tx float64
		_ = rows.Scan(&id, &total, &tx)
		diff := tx - total
		note := "Budget mismatch: ID " + fmt.Sprint(id)
		log.Printf("[WATCHER] ⚠️ Budget %d inconsistent (diff %.2f)", id, diff)

		database.Pool.Exec(ctx, `
			INSERT INTO alerts (message, severity, category)
			VALUES ($1, 'critical', 'budget_integrity')
		`, note)
	}
}

// ternary — helper untuk ekspresi if singkat (seperti operator ? di bahasa lain)
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
