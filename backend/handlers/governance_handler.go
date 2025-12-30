package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// GovernanceScoreSummary merepresentasikan KPI per departemen
type GovernanceScoreSummary struct {
	DepartmentID   *int64    `json:"department_id"`
	DepartmentName *string   `json:"department_name"`
	AvgScore       *float64  `json:"avg_governance_score"`
	ComplianceRate *float64  `json:"compliance_rate"`
	AvgHealthScore *float64  `json:"avg_health_score"`
	AssetCount     int64     `json:"asset_count"`
	LastUpdated    time.Time `json:"last_updated"`
}

// GetGovernanceScoreSummary - KPI Dashboard API
// @Summary Get governance score summary
// @Description Returns average governance score and compliance rate per department
// @Tags Governance
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /governance/score-summary [get]
func GetGovernanceScoreSummary(c *gin.Context) {
	ctx := context.Background()
	query := `
		SELECT 
			d.id AS department_id,
			d.name AS department_name,
			ROUND(AVG(a.governance_score),2) AS avg_governance_score,
			ROUND(100.0 * SUM(CASE WHEN a.compliance_flag = TRUE THEN 1 ELSE 0 END) / NULLIF(COUNT(a.id),0),2) AS compliance_rate,
			ROUND(AVG(a.asset_health_score),2) AS avg_health_score,
			COUNT(a.id) AS asset_count,
			MAX(a.updated_at) AS last_updated
		FROM assets a
		LEFT JOIN departments d ON a.department_id = d.id
		WHERE a.deleted_at IS NULL
		GROUP BY d.id, d.name
		ORDER BY avg_governance_score ASC NULLS LAST
	`

	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		log.Printf("[GOVERNANCE_SUMMARY_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch governance summary"})
		return
	}
	defer rows.Close()

	var summaries []GovernanceScoreSummary
	for rows.Next() {
		var s GovernanceScoreSummary
		if err := rows.Scan(
			&s.DepartmentID,
			&s.DepartmentName,
			&s.AvgScore,
			&s.ComplianceRate,
			&s.AvgHealthScore,
			&s.AssetCount,
			&s.LastUpdated,
		); err == nil {
			summaries = append(summaries, s)
		}
	}

	// 🔹 Hitung rata-rata organisasi
	var orgAvgScore, orgCompliance *float64
	_ = database.Pool.QueryRow(ctx, `
		SELECT 
			ROUND(AVG(governance_score),2),
			ROUND(100.0 * SUM(CASE WHEN compliance_flag = TRUE THEN 1 ELSE 0 END) / NULLIF(COUNT(id),0),2)
		FROM assets WHERE deleted_at IS NULL
	`).Scan(&orgAvgScore, &orgCompliance)

	c.JSON(http.StatusOK, gin.H{
		"organization": gin.H{
			"avg_governance_score": orgAvgScore,
			"avg_compliance_rate":  orgCompliance,
			"total_departments":    len(summaries),
		},
		"departments": summaries,
	})
}
