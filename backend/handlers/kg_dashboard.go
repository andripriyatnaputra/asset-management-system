package handlers

import (
	"log"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// Ringkasan Knowledge Graph untuk dashboard
type KGSummary struct {
	OrphanedAssets            int64 `json:"orphaned_assets"`
	AssetsWithBreachedTickets int64 `json:"assets_with_breached_tickets"`
	HighRiskContracts         int64 `json:"high_risk_contracts"`
	TotalNodes                int64 `json:"total_nodes"`
	TotalEdges                int64 `json:"total_edges"`
}

// GET /dashboard/kg-summary
func GetKGSummary(c *gin.Context) {
	// ✅ Sama pola auth seperti GetDashboardStats
	roleVal, roleExists := c.Get("role")
	_, userExists := c.Get("user_id")
	if !roleExists || !userExists || roleVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ctx := c.Request.Context()
	var summary KGSummary

	// 1) Orphaned assets (punya node asset tapi tidak punya relasi governance utama)
	if err := database.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT a.id
			FROM assets a
			LEFT JOIN kg_nodes n 
			       ON n.entity_type = 'asset' AND n.entity_id = a.id
			LEFT JOIN kg_edges e 
			       ON e.src_node_id = n.id 
			      AND e.rel_type IN ('FUNDED_BY','COVERED_BY','USES_LICENSE')
			WHERE a.deleted_at IS NULL
			GROUP BY a.id
			HAVING COUNT(e.id) = 0
		) sub;
	`).Scan(&summary.OrphanedAssets); err != nil {
		log.Printf("[WARN] KGSummary orphaned_assets query: %v", err)
	}

	// 2) Assets dengan tiket yang breach / non-compliant (pakai tickets yang sudah ada di dashboard SLA)
	if err := database.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.related_asset_id)
		FROM tickets t
		WHERE t.deleted_at IS NULL
		  AND t.related_asset_id IS NOT NULL
		  AND (t.breach_flag = TRUE OR t.compliance_flag = FALSE);
	`).Scan(&summary.AssetsWithBreachedTickets); err != nil {
		log.Printf("[WARN] KGSummary assets_with_breached_tickets query: %v", err)
	}

	// 3) High exposure contracts: kontrak yang meng-cover banyak aset (>=5 aset)
	if err := database.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT e.dst_node_id AS contract_node_id,
			       COUNT(*)       AS asset_count
			FROM kg_edges e
			JOIN kg_nodes n 
			  ON n.id = e.src_node_id
			 AND n.entity_type = 'asset'
			WHERE e.rel_type = 'COVERED_BY'
			GROUP BY e.dst_node_id
			HAVING COUNT(*) >= 5
		) sub;
	`).Scan(&summary.HighRiskContracts); err != nil {
		log.Printf("[WARN] KGSummary high_risk_contracts query: %v", err)
	}

	// 4) Total nodes
	if err := database.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM kg_nodes;
	`).Scan(&summary.TotalNodes); err != nil {
		log.Printf("[WARN] KGSummary total_nodes query: %v", err)
	}

	// 5) Total edges
	if err := database.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM kg_edges;
	`).Scan(&summary.TotalEdges); err != nil {
		log.Printf("[WARN] KGSummary total_edges query: %v", err)
	}

	c.JSON(http.StatusOK, summary)
}
