package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type AssetStatus string

const (
	StatusInStock     AssetStatus = "in_stock"
	StatusAssigned    AssetStatus = "assigned"
	StatusReturned    AssetStatus = "returned"
	StatusMaintenance AssetStatus = "maintenance"
	StatusRetired     AssetStatus = "retired"
	StatusDisposed    AssetStatus = "disposed"
)

// ================================================================
// 🔒 GOVERNANCE TRANSITION VALIDATOR
// ================================================================

// isTransitionAllowed menentukan apakah perpindahan status aset valid
// sesuai prinsip ISO/IEC 19770-10:2025 (Asset Lifecycle Governance)
// ============================================================
// 🔹 Aturan transisi status aset
// ============================================================
//
// Alur normal:
//
//	in_stock → assigned → returned → in_stock/maintenance → retired → disposed
//
// Dengan kelonggaran (opsional):
//
//	disposed → in_stock    → re-activate asset lama
//	maintenance ↔ in_stock → selesai servis bisa kembali aktif
func isTransitionAllowed(from, to AssetStatus) bool {
	if from == to {
		return true // tidak ada perubahan status
	}

	switch from {

	case StatusInStock:
		// stok bisa dipakai atau diperbaiki
		return to == StatusAssigned ||
			to == StatusMaintenance ||
			to == StatusRetired

	case StatusAssigned:
		// sedang dipakai — bisa dikembalikan, diservis, atau dipensiunkan
		return to == StatusInStock ||
			to == StatusReturned ||
			to == StatusMaintenance ||
			to == StatusRetired

	case StatusReturned:
		// setelah dikembalikan bisa diservis atau masuk stok
		return to == StatusInStock ||
			to == StatusMaintenance ||
			to == StatusRetired

	case StatusMaintenance:
		// setelah servis bisa kembali ke stok atau dipensiunkan
		return to == StatusInStock ||
			to == StatusRetired ||
			to == StatusDisposed

	case StatusRetired:
		// pensiun → buang / scrap atau diaktifkan kembali (optional)
		return to == StatusDisposed ||
			to == StatusInStock // <– boleh reaktivasi (opsional)

	case StatusDisposed:
		// aset scrap bisa dihidupkan lagi (optional reactivation)
		return to == StatusInStock

	default:
		return false
	}
}

// ================================================================
// 💰 DEPRECIATION METHODS
// ================================================================

// straight-line depreciation (garis lurus)
func straightLineMonthly(purchaseCost, salvage float64, usefulMonths int) float64 {
	if usefulMonths <= 0 {
		return 0
	}
	return (purchaseCost - salvage) / float64(usefulMonths)
}

// double-declining-balance depreciation
func doubleDecliningMonthly(purchaseCost, salvage float64, usefulMonths int, monthsElapsed int) float64 {
	if usefulMonths <= 0 {
		return 0
	}
	rate := (2.0 / float64(usefulMonths))
	bookValue := purchaseCost * math.Pow(1-rate, float64(monthsElapsed))
	monthly := bookValue * rate
	if monthly < 0 {
		monthly = 0
	}
	return monthly
}

// sum-of-years-digits depreciation
func sumOfYearsMonthly(purchaseCost, salvage float64, usefulMonths int, monthsElapsed int) float64 {
	if usefulMonths <= 0 {
		return 0
	}
	total := float64(usefulMonths * (usefulMonths + 1) / 2)
	remaining := float64(usefulMonths - monthsElapsed + 1)
	factor := remaining / total
	annual := (purchaseCost - salvage) * factor
	return annual / 12.0
}

// calcDepreciation menghitung nilai buku aset per tanggal tertentu
func calcDepreciation(purchaseCost, salvage float64, usefulMonths int,
	purchaseDate time.Time, asOf time.Time, method string) (monthly, accumulated, book float64) {

	if usefulMonths <= 0 {
		return 0, 0, purchaseCost
	}

	months := int((asOf.Sub(purchaseDate)).Hours() / (24 * 30))
	if months < 0 {
		months = 0
	}
	if months > usefulMonths {
		months = usefulMonths
	}

	switch method {
	case "double_declining":
		for m := 1; m <= months; m++ {
			monthly += doubleDecliningMonthly(purchaseCost, salvage, usefulMonths, m)
		}
		accumulated = monthly
	case "sum_of_years":
		for m := 1; m <= months; m++ {
			monthly += sumOfYearsMonthly(purchaseCost, salvage, usefulMonths, m)
		}
		accumulated = monthly
	default:
		monthly = straightLineMonthly(purchaseCost, salvage, usefulMonths)
		accumulated = monthly * float64(months)
	}

	book = purchaseCost - accumulated
	if book < salvage {
		book = salvage
	}
	return monthly, accumulated, book
}

// ================================================================
// 🧩 GOVERNANCE METRICS & HELPERS
// ================================================================

// governanceScore menghitung skor kepatuhan aset (0-100)
func governanceScore(hasContract, hasBudget, hasLifecycle bool) float64 {
	score := 0.0
	if hasContract {
		score += 25
	}
	if hasBudget {
		score += 25
	}
	if hasLifecycle {
		score += 25
	}
	return score
}

// calcAssetHealthScore (prediktif sederhana) – bisa dipakai di services.CalculateAssetHealth
func calcAssetHealthScore(ageMonths, usefulMonths int, condition string) float64 {
	if usefulMonths == 0 {
		return 100
	}
	ageRatio := float64(ageMonths) / float64(usefulMonths)
	conditionPenalty := map[string]float64{
		"excellent": 0.0,
		"good":      0.1,
		"fair":      0.25,
		"poor":      0.4,
	}
	penalty := conditionPenalty[condition]
	health := 100 * (1 - ageRatio - penalty)
	if health < 0 {
		health = 0
	}
	if health > 100 {
		health = 100
	}
	return math.Round(health*10) / 10 // satu desimal
}

// formatChangeDetail menghasilkan string audit yang konsisten
func formatChangeDetail(field, from, to any) string {
	return fmt.Sprintf("%v: %v → %v", field, from, to)
}

// writeAssetHistory — versi standar (non-transactional)
// digunakan untuk operasi tunggal (Create/Update/Dispose/Delete).
func writeAssetHistory(
	c *gin.Context,
	assetID int64,
	action string,
	from, to *string,
	note string,
	compliant bool,
	complianceNote string,
) {
	var actor *int64
	if v := getUserIDPtr(c); v != nil {
		actor = v
	}

	ctx := c.Request.Context()

	// ambil hash terakhir untuk menjaga chain integritas
	var prevHash *string
	_ = database.Pool.QueryRow(ctx,
		`SELECT hash FROM asset_history WHERE asset_id=$1 ORDER BY id DESC LIMIT 1`,
		assetID,
	).Scan(&prevHash)

	// bangun payload yang akan di-hash
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%v",
		assetID, action, ptrOr(from, ""), ptrOr(to, ""), note, time.Now(),
	)
	if prevHash != nil {
		data += *prevHash
	}
	h := sha256.Sum256([]byte(data))
	hash := hex.EncodeToString(h[:])

	_, err := database.Pool.Exec(ctx, `
		INSERT INTO asset_history (
			asset_id, action, detail, actor_employee_id,
			from_status, to_status, compliance_flag, compliance_note,
			hash, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())`,
		assetID, action, note, actor, from, to, compliant, complianceNote, hash,
	)
	if err != nil {
		log.Printf("[HISTORY_WRITE_ERR] asset_id=%d action=%s err=%v", assetID, action, err)
	}
}

// writeAssetHistoryTx — versi transactional (pgx.Tx)
// digunakan untuk operasi multi-tabel seperti AssignAsset / ReturnAsset.
func writeAssetHistoryTx(
	ctx context.Context,
	tx pgx.Tx,
	assetID int64,
	action string,
	from, to *string,
	note string,
	compliant bool,
	complianceNote string,
	actor *int64,
) {
	// ambil hash terakhir dalam transaksi
	var prevHash *string
	_ = tx.QueryRow(ctx,
		`SELECT hash FROM asset_history WHERE asset_id=$1 ORDER BY id DESC LIMIT 1`,
		assetID,
	).Scan(&prevHash)

	data := fmt.Sprintf("%d|%s|%s|%s|%s|%v",
		assetID, action, ptrOr(from, ""), ptrOr(to, ""), note, time.Now(),
	)
	if prevHash != nil {
		data += *prevHash
	}
	h := sha256.Sum256([]byte(data))
	hash := hex.EncodeToString(h[:])

	_, err := tx.Exec(ctx, `
		INSERT INTO asset_history (
			asset_id, action, detail, actor_employee_id,
			from_status, to_status, compliance_flag, compliance_note,
			hash, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())`,
		assetID, action, note, actor, from, to, compliant, complianceNote, hash,
	)
	if err != nil {
		log.Printf("[HISTORY_TX_ERR] asset_id=%d action=%s err=%v", assetID, action, err)
	}
}
