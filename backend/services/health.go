// File: backend/services/health.go
package services

import (
	"context"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// CalculateAssetHealth menghitung health score berdasarkan umur aset, audit, dan status
func CalculateAssetHealth(assetID int64) (float64, error) {
	var purchaseDate time.Time
	var usefulLifeMonths int64
	var lastAudit *time.Time
	var status string

	err := database.Pool.QueryRow(context.Background(), `
		SELECT 
			COALESCE(purchase_date, CURRENT_DATE),
			COALESCE(useful_life_months, 36),
			(SELECT MAX(created_at) FROM audit_logs WHERE entity_name = 'assets' AND entity_id = a.id),
			status
		FROM assets a
		WHERE id = $1 AND deleted_at IS NULL
	`, assetID).Scan(&purchaseDate, &usefulLifeMonths, &lastAudit, &status)
	if err != nil {
		return 50, err
	}

	// base score by age
	ageMonths := int(time.Since(purchaseDate).Hours() / (24 * 30))
	ageRatio := float64(ageMonths) / float64(usefulLifeMonths)
	if ageRatio > 1 {
		ageRatio = 1
	}
	score := 100 * (1 - ageRatio)

	// audit bonus: jika ada audit dalam 6 bulan terakhir
	if lastAudit != nil && time.Since(*lastAudit).Hours() < (24*30*6) {
		score += 5
	}

	// penalty untuk status non-normal
	switch status {
	case "Damaged":
		score -= 20
	case "Maintenance":
		score -= 10
	case "Disposed":
		score -= 50
	}

	// batas aman 0–100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// simpan ke DB
	_, _ = database.Pool.Exec(context.Background(),
		`UPDATE assets SET asset_health_score = $1 WHERE id = $2`, score, assetID)

	return score, nil
}
