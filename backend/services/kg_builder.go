package services

import (
	"context"
	"fmt"
	"log"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

func BuildKnowledgeGraph(ctx context.Context) error {
	log.Println("[KG] building graph...")

	// 1) Departments
	if err := kgSyncDepartments(ctx); err != nil {
		return err
	}
	// 2) Cost Centers
	if err := kgSyncCostCenters(ctx); err != nil {
		return err
	}
	// 3) Budgets
	if err := kgSyncBudgets(ctx); err != nil {
		return err
	}
	// 4) Contracts
	if err := kgSyncContracts(ctx); err != nil {
		return err
	}
	// 5) Licenses
	if err := kgSyncLicenses(ctx); err != nil {
		return err
	}
	// 6) Assets (nodes + edges ke relasi)
	if err := kgSyncAssets(ctx); err != nil {
		return err
	}
	// 7) Tickets (node + edge ke asset)
	if err := kgSyncTickets(ctx); err != nil {
		return err
	}

	log.Println("[KG] build complete")
	return nil
}

func kgUpsertNode(ctx context.Context, typ string, id int64, label string, props string) (int64, error) {
	var nid int64
	err := database.Pool.QueryRow(ctx,
		`SELECT kg_upsert_node($1,$2,$3,$4::jsonb)`, typ, id, label, props).Scan(&nid)
	return nid, err
}

func kgUpsertEdge(ctx context.Context, src, dst int64, rel, props string, weight float64) error {
	_, err := database.Pool.Exec(ctx,
		`SELECT kg_upsert_edge($1,$2,$3,$4::jsonb,$5)`, src, dst, rel, props, weight)
	return err
}

// ======== Sync examples (lengkapi sesuai kebutuhan) ========

func kgSyncDepartments(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, COALESCE(name,'Department') FROM departments WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		if _, err := kgUpsertNode(ctx, "department", id, name, `{}`); err != nil {
			log.Printf("[KG] dept node err: %v", err)
		}
	}
	return nil
}

func kgSyncCostCenters(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, COALESCE(name,'CC'), department_id FROM cost_centers WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, deptID *int64
		var name string
		if err := rows.Scan(&id, &name, &deptID); err != nil {
			continue
		}
		nid, err := kgUpsertNode(ctx, "cost_center", *id, name, `{}`)
		if err != nil {
			continue
		}
		if deptID != nil {
			var deptNodeID int64
			_ = database.Pool.QueryRow(ctx, `SELECT id FROM kg_nodes WHERE entity_type='department' AND entity_id=$1`, *deptID).Scan(&deptNodeID)
			if deptNodeID != 0 {
				_ = kgUpsertEdge(ctx, deptNodeID, nid, "OWNS_CC", `{}`, 1)
			}
		}
	}
	return nil
}

func kgSyncBudgets(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, name, department_id, total_amount FROM budgets WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, deptID *int64
		var name string
		var total float64
		if err := rows.Scan(&id, &name, &deptID, &total); err != nil {
			continue
		}
		props := fmt.Sprintf(`{"total_amount": %.2f}`, total)
		bNode, err := kgUpsertNode(ctx, "budget", *id, name, props)
		if err != nil {
			continue
		}
		if deptID != nil {
			var deptNode int64
			_ = database.Pool.QueryRow(ctx, `SELECT id FROM kg_nodes WHERE entity_type='department' AND entity_id=$1`, *deptID).Scan(&deptNode)
			if deptNode != 0 {
				_ = kgUpsertEdge(ctx, deptNode, bNode, "OWNS_BUDGET", `{}`, 1)
			}
		}
	}
	return nil
}

func kgSyncContracts(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, COALESCE(contract_number,'Contract'), vendor FROM contracts WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label, vendor string
		if err := rows.Scan(&id, &label, &vendor); err != nil {
			continue
		}
		props := fmt.Sprintf(`{"vendor": %q}`, vendor)
		_, _ = kgUpsertNode(ctx, "contract", id, label, props)
	}
	return nil
}

func kgSyncLicenses(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, name, license_type, expiration_date FROM licenses WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, typ string
		var exp *string
		if err := rows.Scan(&id, &name, &typ, &exp); err != nil {
			continue
		}
		prop := fmt.Sprintf(`{"license_type": %q, "expiration": %q}`, typ, valOrStr(exp, ""))
		_, _ = kgUpsertNode(ctx, "license", id, name, prop)
	}
	return nil
}

func kgSyncAssets(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `
		SELECT id, name, governance_score, compliance_flag, lifecycle_stage, budget_id, contract_id, license_id, department_id, cost_center_id
		  FROM assets WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name string
		var g float64
		var c *bool
		var stage *string
		var bID, ctID, lID, dID, ccID *int64

		if err := rows.Scan(&id, &name, &g, &c, &stage, &bID, &ctID, &lID, &dID, &ccID); err != nil {
			continue
		}
		props := fmt.Sprintf(`{"governance_score": %.2f, "compliance_flag": %v, "lifecycle_stage": %q}`, g, valOrBool(c, false), valOrStr(stage, ""))
		aNode, err := kgUpsertNode(ctx, "asset", id, name, props)
		if err != nil {
			continue
		}

		// edges:
		if bID != nil {
			linkByEntity(ctx, aNode, "budget", *bID, "FUNDED_BY")
		}
		if ctID != nil {
			linkByEntity(ctx, aNode, "contract", *ctID, "COVERED_BY")
		}
		if lID != nil {
			linkByEntity(ctx, aNode, "license", *lID, "USES_LICENSE")
		}
		if dID != nil {
			linkByEntity(ctx, aNode, "department", *dID, "ASSET_OF")
		}
		if ccID != nil {
			linkByEntity(ctx, aNode, "cost_center", *ccID, "ASSET_OF_CC")
		}
	}
	return nil
}

func kgSyncTickets(ctx context.Context) error {
	rows, err := database.Pool.Query(ctx, `SELECT id, subject, status, related_asset_id, sla_breached_at FROM tickets WHERE deleted_at IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var subject, status string
		var assetID *int64
		var breach *string
		if err := rows.Scan(&id, &subject, &status, &assetID, &breach); err != nil {
			continue
		}
		props := fmt.Sprintf(`{"status": %q, "breached": %v}`, status, breach != nil)
		tNode, err := kgUpsertNode(ctx, "ticket", id, subject, props)
		if err != nil {
			continue
		}
		if assetID != nil {
			var aNode int64
			_ = database.Pool.QueryRow(ctx, `SELECT id FROM kg_nodes WHERE entity_type='asset' AND entity_id=$1`, *assetID).Scan(&aNode)
			if aNode != 0 {
				_ = kgUpsertEdge(ctx, aNode, tNode, "HAS_TICKET", `{}`, 1)
			}
		}
	}
	return nil
}

func linkByEntity(ctx context.Context, srcNode int64, typ string, entityID int64, rel string) {
	var dst int64
	_ = database.Pool.QueryRow(ctx, `SELECT id FROM kg_nodes WHERE entity_type=$1 AND entity_id=$2`, typ, entityID).Scan(&dst)
	if dst != 0 {
		_ = kgUpsertEdge(ctx, srcNode, dst, rel, `{}`, 1)
	}
}

func valOrStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}
func valOrBool(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}
