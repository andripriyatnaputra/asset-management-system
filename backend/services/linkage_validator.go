package services

import (
	"context"
	"fmt"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// ============================================================
// 🔍 Linkage Check Utility — memastikan entitas masih aktif
// ============================================================
func CheckActiveLinkages(ctx context.Context, table string, id int64) ([]string, error) {
	queryMap := map[string]string{
		"assets": `
			SELECT 'contract' FROM assets WHERE id=$1 AND contract_id IS NOT NULL AND deleted_at IS NULL
			UNION
			SELECT 'license' FROM assets WHERE id=$1 AND license_id IS NOT NULL AND deleted_at IS NULL
			UNION
			SELECT 'budget' FROM assets WHERE id=$1 AND budget_id IS NOT NULL AND deleted_at IS NULL
			UNION
			SELECT 'cost_center' FROM assets WHERE id=$1 AND cost_center_id IS NOT NULL AND deleted_at IS NULL;
		`,
		"contracts": `
			SELECT 'asset' FROM assets WHERE contract_id=$1 AND deleted_at IS NULL LIMIT 1;
		`,
		"licenses": `
			SELECT 'asset' FROM assets WHERE license_id=$1 AND deleted_at IS NULL LIMIT 1;
		`,
		"budgets": `
			SELECT 'asset' FROM assets WHERE budget_id=$1 AND deleted_at IS NULL LIMIT 1;
		`,
	}

	sql, ok := queryMap[table]
	if !ok {
		return nil, fmt.Errorf("unsupported table for linkage check: %s", table)
	}

	rows, err := database.Pool.Query(ctx, sql, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []string
	for rows.Next() {
		var ref string
		_ = rows.Scan(&ref)
		links = append(links, ref)
	}

	return links, nil
}
