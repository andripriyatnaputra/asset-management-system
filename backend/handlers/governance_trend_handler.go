package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// GovernanceTrendPoint merepresentasikan satu titik data tren bulanan
type GovernanceTrendPoint struct {
	Month              string   `json:"month"` // e.g. "2025-09"
	AvgScore           *float64 `json:"avg_governance_score"`
	ComplianceRate     *float64 `json:"compliance_rate"`
	AvgHealthScore     *float64 `json:"avg_health_score"`
	TotalAssets        int64    `json:"total_assets"`
	CompliantAssets    int64    `json:"compliant_assets"`
	NonCompliantAssets int64    `json:"non_compliant_assets"`
}

// GetGovernanceTrend mengembalikan tren KPI governance 12 bulan terakhir
// @Summary Get governance trend (last 12 months)
// @Description Returns monthly average governance score & compliance rate
// @Tags Governance
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /governance/trend [get]
func GetGovernanceTrend(c *gin.Context) {
	ctx := context.Background()

	// 🔹 Buat atau update tabel ringkasan bulanan (opsional)
	_, _ = database.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS governance_trend (
			month TEXT PRIMARY KEY,
			avg_governance_score NUMERIC(5,2),
			compliance_rate NUMERIC(5,2),
			avg_health_score NUMERIC(5,2),
			total_assets INT,
			created_at TIMESTAMP DEFAULT now()
		)
	`)

	// 🔹 Ambil data langsung dari assets (agar selalu up-to-date)
	query := `
		SELECT 
			to_char(a.updated_month, 'YYYY-MM') AS month,
			ROUND(AVG(a.governance_score),2), AS avg_governance_score,
			ROUND(100.0 * SUM(CASE WHEN a.compliance_flag THEN 1 ELSE 0 END) / NULLIF(COUNT(a.id),0),2) AS compliance_rate,
			ROUND(AVG(a.asset_health_score),2) AS avg_health_score,
			COUNT(a.id) AS total_assets,
			SUM(CASE WHEN a.compliance_flag THEN 1 ELSE 0 END) AS compliant_assets,
			SUM(CASE WHEN NOT a.compliance_flag THEN 1 ELSE 0 END) AS non_compliant_assets
		FROM assets a
		WHERE a.deleted_at IS NULL
		GROUP BY month
		ORDER BY month ASC
		LIMIT 12
	`

	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("[GOVERNANCE_TREND_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch governance trend"})
		return
	}
	defer rows.Close()

	var points []GovernanceTrendPoint
	for rows.Next() {
		var p GovernanceTrendPoint
		if err := rows.Scan(
			&p.Month,
			&p.AvgScore,
			&p.ComplianceRate,
			&p.AvgHealthScore,
			&p.TotalAssets,
			&p.CompliantAssets,
			&p.NonCompliantAssets,
		); err == nil {
			points = append(points, p)
		}
	}

	// 🔹 Hitung ringkasan agregat 12 bulan terakhir
	var avg12, comp12 *float64
	_ = database.Pool.QueryRow(ctx, `
		SELECT 
			ROUND(AVG(governance_score),2),
			ROUND(100.0 * SUM(CASE WHEN compliance_flag THEN 1 ELSE 0 END)/NULLIF(COUNT(id),0),2)
		FROM assets 
		WHERE deleted_at IS NULL
		  AND updated_at >= (NOW() - INTERVAL '12 months')
	`).Scan(&avg12, &comp12)

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"period":               "last 12 months",
			"avg_governance_score": avg12,
			"avg_compliance_rate":  comp12,
			"total_points":         len(points),
		},
		"trend": points,
	})
}
