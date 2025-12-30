package services

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// ============================================================
// 🔮 Predictive Governance & Risk Forecast Service
// ============================================================
// Menghitung indeks risiko aset berdasarkan trend skor governance,
// compliance history, dan reversal frequency.
// ============================================================
type AssetRiskForecast struct {
	AssetID         int64     `json:"asset_id"`
	AssetName       string    `json:"asset_name"`
	GovernanceScore float64   `json:"governance_score"`
	ComplianceFlag  bool      `json:"compliance_flag"`
	ReversalCount   int       `json:"reversal_count"`
	LastUpdated     time.Time `json:"last_updated"`
	RiskIndex       float64   `json:"risk_index"`
	PredictionNote  string    `json:"prediction_note"`
}

func ComputeAssetRiskForecast(ctx context.Context) ([]AssetRiskForecast, error) {
	params := FetchCalibrationParams(ctx)

	rows, err := database.Pool.Query(ctx, `
		WITH reversal_count AS (
			SELECT asset_id, COUNT(*) AS count
			  FROM budget_transactions
			 WHERE category='REVERSAL'
			 GROUP BY asset_id
		)
		SELECT a.id, a.name, a.governance_score, a.compliance_flag,
		       COALESCE(r.count,0) AS reversal_count,
		       a.updated_at
		  FROM assets a
		  LEFT JOIN reversal_count r ON r.asset_id = a.id
		 WHERE a.deleted_at IS NULL;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AssetRiskForecast
	for rows.Next() {
		var item AssetRiskForecast
		if err := rows.Scan(
			&item.AssetID, &item.AssetName, &item.GovernanceScore,
			&item.ComplianceFlag, &item.ReversalCount, &item.LastUpdated,
		); err != nil {
			log.Printf("[SCAN_RISK_ERR] %v", err)
			continue
		}

		item.RiskIndex, item.PredictionNote = calculateRiskWithParams(item, params)
		results = append(results, item)
	}

	return results, nil
}

func calculateRiskWithParams(a AssetRiskForecast, params MLCalibrationParams) (float64, string) {
	risk := (100 - a.GovernanceScore) * params.GovernanceWeight
	if !a.ComplianceFlag {
		risk += 100 * params.ComplianceWeight
	}
	risk += float64(a.ReversalCount) * params.ReversalWeight
	risk += (time.Since(a.LastUpdated).Hours() / (24 * 180)) * params.TimeDecayFactor * 100

	// Additional static safety layer (optional tuning)
	if a.GovernanceScore < 60 {
		risk += (60 - a.GovernanceScore) * 1.2
	}
	if !a.ComplianceFlag {
		risk += 20
	}
	risk += float64(a.ReversalCount) * 5

	ageDays := time.Since(a.LastUpdated).Hours() / 24
	if ageDays > 90 {
		risk += 10
	} else if ageDays > 180 {
		risk += 20
	}
	if risk > 100 {
		risk = 100
	}

	note := "Healthy"
	switch {
	case risk >= 75:
		note = "Critical — high risk of non-compliance"
	case risk >= 50:
		note = "Warning — moderate risk"
	case risk >= 30:
		note = "Caution — early signs of degradation"
	}

	return math.Round(risk*10) / 10, note
}

func FetchCalibrationParams(ctx context.Context) MLCalibrationParams {
	var jsonParams []byte
	var params MLCalibrationParams
	err := database.Pool.QueryRow(ctx, `
		SELECT parameters FROM ml_calibration_models
		 WHERE model_name='governance_risk_v1'
		 ORDER BY last_trained_at DESC LIMIT 1
	`).Scan(&jsonParams)
	if err == nil {
		_ = json.Unmarshal(jsonParams, &params)
	}
	return params
}
