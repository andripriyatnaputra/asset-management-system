package services

import (
	"context"
	"fmt"
	"log"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// ============================================================
// 👥 Human-in-the-Loop Governance Review Service
// ============================================================

type ReviewInput struct {
	AssetID          int64   `json:"asset_id"`
	RiskIndex        float64 `json:"risk_index"`
	SystemNote       string  `json:"system_note"`
	ReviewerID       int64   `json:"reviewer_id"`
	ReviewerComment  string  `json:"reviewer_comment"`
	ReviewerDecision bool    `json:"reviewer_decision"`
}

// SaveFeedback menyimpan review manusia terhadap prediksi sistem
func SaveFeedback(ctx context.Context, input ReviewInput) error {
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO governance_review_feedback
			(asset_id, reviewer_id, risk_index, system_note, reviewer_comment, reviewer_decision)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, input.AssetID, input.ReviewerID, input.RiskIndex,
		input.SystemNote, input.ReviewerComment, input.ReviewerDecision)
	if err != nil {
		return fmt.Errorf("failed to insert review feedback: %w", err)
	}

	log.Printf("[GOV_REVIEW] Reviewer %d feedback for asset %d stored (decision=%v)",
		input.ReviewerID, input.AssetID, input.ReviewerDecision)

	// Optional: trigger recalibration every 50 feedback entries
	var count int
	_ = database.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM governance_review_feedback`).Scan(&count)
	if count%50 == 0 {
		go RunModelCalibration(ctx) // Tahap 11 auto recalibrate
	}
	return nil
}
