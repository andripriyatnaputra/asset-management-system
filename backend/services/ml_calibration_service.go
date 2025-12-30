package services

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// ============================================================
// 🤖 Machine Learning Calibration Service
// ============================================================
// Menganalisis data historis dan mengupdate bobot faktor risiko
// ============================================================

type MLCalibrationParams struct {
	GovernanceWeight float64 `json:"governance_weight"`
	ComplianceWeight float64 `json:"compliance_weight"`
	ReversalWeight   float64 `json:"reversal_weight"`
	TimeDecayFactor  float64 `json:"time_decay_factor"`
}

func RunModelCalibration(ctx context.Context) {
	log.Println("[ML_CALIBRATION] Starting calibration process...")

	// Ambil data historis 12 bulan terakhir
	rows, err := database.Pool.Query(ctx, `
		SELECT a.governance_score, a.compliance_flag,
		       COALESCE(r.rev_count,0) AS reversal_count,
		       EXTRACT(EPOCH FROM (NOW() - a.updated_at))/86400 AS age_days,
		       CASE WHEN t.status='Resolved' THEN 1 ELSE 0 END AS ticket_resolved
		  FROM assets a
		  LEFT JOIN (
			SELECT asset_id, COUNT(*) AS rev_count
			  FROM budget_transactions
			 WHERE category='REVERSAL'
			   AND transaction_date > NOW() - INTERVAL '12 months'
			 GROUP BY asset_id
		  ) r ON r.asset_id=a.id
		  LEFT JOIN tickets t ON t.related_asset_id=a.id
		 WHERE a.deleted_at IS NULL;
	`)
	if err != nil {
		log.Printf("[ML_CALIBRATION_ERR] %v", err)
		return
	}
	defer rows.Close()

	var total int
	var errSum float64
	var gW, cW, rW, tDecay float64 = 0.5, 0.3, 0.15, 0.05

	// 🔹 Adjust weights based on reviewer feedback
	var correct, incorrect int
	_ = database.Pool.QueryRow(ctx, `
	SELECT 
		COUNT(*) FILTER (WHERE reviewer_decision=true),
		COUNT(*) FILTER (WHERE reviewer_decision=false)
	FROM governance_review_feedback
	WHERE created_at > NOW() - INTERVAL '30 days'
`).Scan(&correct, &incorrect)

	adjust := float64(correct - incorrect)
	if adjust != 0 {
		gW = math.Max(0.1, gW+(adjust/500.0))
		cW = math.Max(0.1, cW+(adjust/800.0))
		rW = math.Max(0.05, rW+(adjust/1000.0))
		log.Printf("[ML_CALIBRATION] Adjusted weights from HITL feedback: Δ=%.3f", adjust)
	}

	for rows.Next() {
		var gScore float64
		var cFlag bool
		var revCount int
		var ageDays float64
		var ticketResolved int

		if err := rows.Scan(&gScore, &cFlag, &revCount, &ageDays, &ticketResolved); err != nil {
			continue
		}

		// Simulasi model sederhana: risk_pred = Σ(w_i * factor_i)
		riskPred := (100-gScore)*gW + (1.0-boolToFloat(cFlag))*cW + float64(revCount)*rW + (ageDays/180.0)*tDecay*100

		// Ground truth → high risk jika belum resolved dan governance rendah
		actualHigh := (gScore < 60 && ticketResolved == 0)
		predictedHigh := riskPred > 75

		if actualHigh == predictedHigh {
			correct++
		}

		errSum += math.Abs(riskPred - boolToFloat(actualHigh)*100)
		total++
	}

	if total == 0 {
		log.Println("[ML_CALIBRATION] no data samples found")
		return
	}

	avgError := errSum / float64(total)
	accuracy := float64(correct) / float64(total) * 100

	params := MLCalibrationParams{
		GovernanceWeight: gW,
		ComplianceWeight: cW,
		ReversalWeight:   rW,
		TimeDecayFactor:  tDecay,
	}

	jsonParams, _ := json.Marshal(params)

	_, err = database.Pool.Exec(ctx, `
		INSERT INTO ml_calibration_models (model_name, total_samples, avg_error, parameters)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (model_name)
		DO UPDATE SET last_trained_at=NOW(), total_samples=$2, avg_error=$3, parameters=$4
	`, "governance_risk_v1", total, avgError, string(jsonParams))
	if err != nil {
		log.Printf("[ML_CALIBRATION_SAVE_ERR] %v", err)
		return
	}

	log.Printf("[ML_CALIBRATION] Model updated: accuracy=%.1f%% avg_error=%.3f n=%d", accuracy, avgError, total)
}

func boolToFloat(v bool) float64 {
	if v {
		return 1
	}
	return 0
}
