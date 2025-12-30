package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type simpleRow struct {
	ID   int64
	Name string
}

// ----------------------------------------------------
// DEBUG HELPERS
// ----------------------------------------------------

func ExecQ(ctx context.Context, tx pgx.Tx, sql string, args ...any) error {
	_, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		log.Printf("❌ SQL ERROR\nQUERY:\n%s\nARGS: %#v\nERR: %v\n", sql, args, err)
	}
	return err
}

// ----------------------------------------------------
// MAIN
// ----------------------------------------------------

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://admin:secret@localhost:5432/asset_db?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("cannot connect db: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("cannot begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	log.Println("🧹 Resetting seed tables ...")

	if err := ExecQ(ctx, tx, `
        TRUNCATE TABLE 
            ticket_comments,
            ticket_attachments,
            tickets,
            budget_transactions,
            software_installations,
            assets,
            asset_assignments,
            licenses,
            employee_trainings,
            contracts,
            budgets,
            locations,
            cost_centers,
            sla_policies,
            alerts,
            audit_logs
        RESTART IDENTITY CASCADE;
    `); err != nil {
		log.Fatalf("truncate failed: %v", err)
	}

	depts, err := loadDepartments(ctx, tx)
	if err != nil {
		log.Fatalf("loadDepartments failed: %v", err)
	}
	emps, err := loadEmployees(ctx, tx)
	if err != nil {
		log.Fatalf("loadEmployees failed: %v", err)
	}
	assetTypes, err := loadAssetTypes(ctx, tx)
	if err != nil {
		log.Fatalf("loadAssetTypes failed: %v", err)
	}

	emps, depts, err = ensureMinimumData(ctx, tx, emps, depts)
	if err != nil {
		log.Fatalf("ensureMinimumData failed: %v", err)
	}

	log.Printf("Found: %d departments, %d employees, %d assetTypes", len(depts), len(emps), len(assetTypes))

	costCenters, err := seedCostCenters(ctx, tx)
	if err != nil {
		log.Fatalf("seedCostCenters failed: %v", err)
	}

	locations, err := seedLocations(ctx, tx, emps[0].ID)
	if err != nil {
		log.Fatalf("seedLocations failed: %v", err)
	}

	budgetIDs, err := seedBudgets(ctx, tx, depts, costCenters)
	if err != nil {
		log.Fatalf("seedBudgets failed: %v", err)
	}

	contractIDs, err := seedContracts(ctx, tx, budgetIDs, costCenters)
	if err != nil {
		log.Fatalf("seedContracts failed: %v", err)
	}

	licenseIDs, err := seedLicenses(ctx, tx, budgetIDs, contractIDs, emps[0].ID)
	if err != nil {
		log.Fatalf("seedLicenses failed: %v", err)
	}

	assetIDs, err := seedAssets(ctx, tx, depts, emps, assetTypes, budgetIDs, contractIDs, licenseIDs, locations, costCenters)
	if err != nil {
		log.Fatalf("seedAssets failed: %v", err)
	}

	if _, err := seedTicketCategories(ctx, tx); err != nil {
		log.Fatalf("seedTicketCategories failed: %v", err)
	}
	if _, err := seedServices(ctx, tx); err != nil {
		log.Fatalf("seedServices failed: %v", err)
	}

	slaIDs, err := seedSLAPolicies(ctx, tx, emps[0].ID)
	if err != nil {
		log.Fatalf("seedSLAPolicies failed: %v", err)
	}

	if err := seedEmployeeTrainings(ctx, tx, emps); err != nil {
		log.Fatalf("seedEmployeeTrainings failed: %v", err)
	}

	if err := seedBudgetTransactions(ctx, tx, budgetIDs, costCenters, assetIDs, contractIDs, licenseIDs, emps[0].ID); err != nil {
		log.Fatalf("seedBudgetTransactions failed: %v", err)
	}

	if err := seedTickets(ctx, tx, emps, assetIDs, slaIDs); err != nil {
		log.Fatalf("seedTickets failed: %v", err)
	}

	if err := seedAlerts(ctx, tx, emps, assetIDs); err != nil {
		log.Fatalf("seedAlerts failed: %v", err)
	}

	if err := seedAuditLogs(ctx, tx, emps); err != nil {
		log.Fatalf("seedAuditLogs failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("❌ commit failed: %v", err)
	}

	log.Println("🎉 Seeder completed successfully.")
}

// ----------------------------------------------------
// LOADERS
// ----------------------------------------------------

func loadDepartments(ctx context.Context, tx pgx.Tx) ([]simpleRow, error) {
	rows, err := tx.Query(ctx, `SELECT id, name FROM departments WHERE deleted_at IS NULL OR deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []simpleRow
	for rows.Next() {
		var r simpleRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func loadEmployees(ctx context.Context, tx pgx.Tx) ([]simpleRow, error) {
	rows, err := tx.Query(ctx, `SELECT id, name FROM employees WHERE deleted_at IS NULL OR deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []simpleRow
	for rows.Next() {
		var r simpleRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func loadAssetTypes(ctx context.Context, tx pgx.Tx) ([]simpleRow, error) {
	rows, err := tx.Query(ctx, `SELECT id, name FROM asset_types ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []simpleRow
	for rows.Next() {
		var r simpleRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

// ----------------------------------------------------
// FALLBACK ROOT DATA
// ----------------------------------------------------

func ensureMinimumData(ctx context.Context, tx pgx.Tx, emps, depts []simpleRow) ([]simpleRow, []simpleRow, error) {

	if len(depts) == 0 {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO departments (name, created_at, updated_at)
            VALUES ('IT Operations', NOW(), NOW())
            RETURNING id`,
		).Scan(&id)
		if err != nil {
			return nil, nil, fmt.Errorf("fallback dept failed: %w", err)
		}
		log.Println("⚠️ Created fallback department 'IT Operations'")
		depts = append(depts, simpleRow{ID: id, Name: "IT Operations"})
	}

	if len(emps) == 0 {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO employees
            (employee_nik, name, email, password_hash, role, department_id,
             created_at, updated_at)
            VALUES
            ('SEED-0001','Seeder Admin','seed.admin@local',
             '$2a$10$abcdefghijklmnopqrstuv1234567890abcdEFGH',
             'super_admin', NULL, NOW(), NOW())
            RETURNING id`,
		).Scan(&id)
		if err != nil {
			return nil, nil, fmt.Errorf("fallback employee failed: %w", err)
		}
		log.Println("⚠️ Created fallback employee 'Seeder Admin'")
		emps = append(emps, simpleRow{ID: id, Name: "Seeder Admin"})
	}

	return emps, depts, nil
}

// ----------------------------------------------------
// SEEDERS
// ----------------------------------------------------

func seedCostCenters(ctx context.Context, tx pgx.Tx) ([]int64, error) {
	type C struct{ code, name string }
	rows := []C{
		{"NOC-JKT", "Network Operations Center"},
		{"ITSM-SUP", "ITSM Support Team"},
		{"DC-HLM", "Data Center Halim"},
	}

	var ids []int64
	for _, r := range rows {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO cost_centers (code,name,created_at,updated_at)
            VALUES ($1,$2,NOW(),NOW()) RETURNING id`,
			r.code, r.name,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedCostCenters row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded cost centers")
	return ids, nil
}

func seedLocations(ctx context.Context, tx pgx.Tx, userID int64) ([]int64, error) {
	type L struct {
		site, building, room, desc string
	}

	rows := []L{
		{"Jakarta HQ", "Tower A", "NOC-R1", "Main NOC"},
		{"Jakarta DC", "DC1", "Rack-22", "Core infra"},
		{"Bandung POP", "POP-BDG", "Rack-3", "POP West Java"},
	}

	var ids []int64
	for _, r := range rows {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO locations
            (site,building,room,description,status,created_by,updated_by,
             created_at,updated_at)
            VALUES ($1,$2,$3,$4,'active',$5,$5,NOW(),NOW())
            RETURNING id`,
			r.site, r.building, r.room, r.desc, userID,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedLocations row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded locations")
	return ids, nil
}

func seedBudgets(ctx context.Context, tx pgx.Tx, depts []simpleRow, ccs []int64) ([]int64, error) {
	type B struct {
		name  string
		total float64
		cat   string
	}

	rows := []B{
		{"FY2025 CAPEX Network", 5_000_000_000, "CAPEX"},
		{"FY2025 OPEX ITSM", 1_500_000_000, "OPEX"},
		{"FY2025 OPEX DC", 2_000_000_000, "OPEX"},
	}

	var ids []int64
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.Local)

	for i, r := range rows {
		dept := depts[i%len(depts)]
		cc := ccs[i%len(ccs)]

		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO budgets
            (name, department_id, start_date, end_date,
             total_amount, used_amount, category,
             cost_center_id, currency,
             created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5,0,$6,$7,'IDR',NOW(),NOW())
            RETURNING id`,
			r.name, dept.ID, start, end, r.total, r.cat, cc,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedBudgets row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded budgets")
	return ids, nil
}

func seedContracts(ctx context.Context, tx pgx.Tx, budgets []int64, ccs []int64) ([]int64, error) {
	type C struct {
		num, vendor, ctype string
		val                float64
	}

	rows := []C{
		{"CNT-NET-001", "Cisco Indonesia", "Maintenance", 1_800_000_000},
		{"CNT-ITSM-002", "ServiceNow Partner", "SaaS", 900_000_000},
	}

	var ids []int64
	for i, r := range rows {
		start := time.Date(2025, 1, 15+10*i, 0, 0, 0, 0, time.Local)
		end := start.AddDate(1, 0, 0)

		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO contracts
            (contract_number, vendor, contract_type,
             start_date, end_date, total_value,
             currency, status, budget_id, cost_center_id,
             created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,'IDR','active',$7,$8,
                    NOW(),NOW())
            RETURNING id`,
			r.num, r.vendor, r.ctype,
			start, end, r.val,
			budgets[i%len(budgets)], ccs[i%len(ccs)],
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedContracts row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded contracts")
	return ids, nil
}

func seedLicenses(ctx context.Context, tx pgx.Tx, budgets []int64, contracts []int64, createdBy int64) ([]int64, error) {
	type L struct {
		name   string
		key    string
		seats  int
		vendor string
		cat    string
	}

	rows := []L{
		{"ITSM Platform", "ITSM-KEY-001", 25, "ServiceNow", "ITSM"},
		{"NMS Enterprise", "NMS-KEY-002", 50, "SolarWinds", "Monitoring"},
	}

	var ids []int64
	for i, r := range rows {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO licenses
            (name, license_key, total_seats, vendor,
             category, budget_id, contract_id,
             created_by, updated_by, created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8,NOW(),NOW())
            RETURNING id`,
			r.name, r.key, r.seats, r.vendor,
			r.cat, budgets[i%len(budgets)], contracts[i%len(contracts)],
			createdBy,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedLicenses row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded licenses")
	return ids, nil
}

func seedAssets(
	ctx context.Context, tx pgx.Tx,
	depts []simpleRow, emps []simpleRow, types []simpleRow,
	budgets []int64, contracts []int64, licenses []int64,
	locs []int64, ccs []int64,
) ([]int64, error) {

	type A struct {
		name, tag, status, cond, critical string
	}

	rows := []A{
		{"Core Router", "AST-NET-001", "in_use", "good", "critical"},
		{"Firewall DC", "AST-NET-002", "in_use", "good", "high"},
		{"ITSM App Server", "AST-APP-003", "in_use", "good", "high"},
		{"NOC Laptop", "AST-EU-004", "in_stock", "good", "medium"},
	}

	var ids []int64
	for i, r := range rows {
		dept := depts[i%len(depts)]
		loc := locs[i%len(locs)]
		tID := types[i%len(types)].ID
		bID := budgets[i%len(budgets)]
		cID := contracts[i%len(contracts)]
		lID := licenses[i%len(licenses)]
		ccID := ccs[i%len(ccs)]

		purchaseDate := time.Now().AddDate(-1, 0, -i)

		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO assets
            (name, asset_tag, status, department_id, location_id,
             asset_type_id, purchase_date, purchase_cost,
             depreciation_method, ownership_type, acquisition_type,
             asset_condition, lifecycle_stage, asset_criticality,
             budget_id, contract_id, license_id, cost_center_id,
             created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5,
                    $6,$7,$8,
                    'straight_line','company_owned','purchase',
                    $9,'in_use',$10,
                    $11,$12,$13,$14,
                    NOW(),NOW())
            RETURNING id`,
			r.name, r.tag, r.status, dept.ID, loc,
			tID, purchaseDate, 300_000_000+int64(i)*50_000_000,
			r.cond, r.critical,
			bID, cID, lID, ccID,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedAssets row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)

		if r.status == "in_use" {
			if err := ExecQ(ctx, tx, `
                INSERT INTO asset_assignments (asset_id, employee_id, assigned_at)
                VALUES ($1,$2,NOW())`,
				id, emps[i%len(emps)].ID,
			); err != nil {
				return nil, err
			}
		}
	}

	log.Println("✓ seeded assets")
	return ids, nil
}

func seedServices(ctx context.Context, tx pgx.Tx) ([]string, error) {
	rows := []struct {
		code string
		name string
	}{
		{"NET", "Network Service"},
		{"APP", "Application Service"},
		{"SYS", "System Infrastructure"},
	}

	var codes []string

	for _, r := range rows {
		if err := ExecQ(ctx, tx, `
            INSERT INTO services (code, name)
            VALUES ($1,$2)
            ON CONFLICT (code) DO NOTHING`, r.code, r.name); err != nil {
			return nil, err
		}
		codes = append(codes, r.code)
	}

	log.Println("✓ seeded services")
	return codes, nil
}

func seedTicketCategories(ctx context.Context, tx pgx.Tx) ([]string, error) {
	rows := []struct {
		code string
		name string
	}{
		{"INC", "Incident"},
		{"REQ", "Service Request"},
		{"PRB", "Problem"},
	}

	var codes []string

	for _, r := range rows {
		if err := ExecQ(ctx, tx, `
            INSERT INTO ticket_categories (code, name)
            VALUES ($1,$2)
            ON CONFLICT (code) DO NOTHING`, r.code, r.name); err != nil {
			return nil, err
		}
		codes = append(codes, r.code)
	}

	log.Println("✓ seeded ticket_categories")
	return codes, nil
}

func seedSLAPolicies(ctx context.Context, tx pgx.Tx, createdBy int64) ([]int64, error) {
	type S struct {
		name   string
		impact string
		urg    string
		prio   string
		cat    *string
		svc    *string
		resp   int
		res    int
	}

	inc := "INC"
	req := "REQ"
	net := "NET"
	app := "APP"

	rows := []S{
		{"P1 - Network Down", "High", "High", "Critical", &inc, &net, 30, 120},
		{"P2 - Degraded Svc", "Medium", "High", "High", &inc, &net, 60, 240},
		{"P3 - ITSM Request", "Low", "Low", "Medium", &req, &app, 240, 1440},
		{"Default SLA", "Medium", "Medium", "Medium", nil, nil, 120, 480},
	}

	var ids []int64
	for _, r := range rows {
		var id int64
		err := tx.QueryRow(ctx, `
            INSERT INTO sla_policies
            (name, category_code, service_code,
             impact, urgency, resulting_priority,
             response_minutes, resolve_minutes,
             created_by, updated_by, created_at, updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
                    $9,$9,NOW(),NOW())
            RETURNING id`,
			r.name, r.cat, r.svc,
			r.impact, r.urg, r.prio,
			r.resp, r.res,
			createdBy,
		).Scan(&id)
		if err != nil {
			log.Printf("❌ seedSLAPolicies row failed: %v", err)
			return nil, err
		}
		ids = append(ids, id)
	}

	log.Println("✓ seeded SLA policies")
	return ids, nil
}

func seedEmployeeTrainings(ctx context.Context, tx pgx.Tx, emps []simpleRow) error {
	for _, e := range emps {
		if err := ExecQ(ctx, tx, `
            INSERT INTO employee_trainings
            (employee_id, training_name, certificate_url, completed_at, created_at)
            VALUES ($1,$2,$3,NOW(),NOW())`,
			e.ID, "ISO/IEC 19770-10 Awareness", "https://example.com/cert.pdf",
		); err != nil {
			return err
		}
	}
	log.Println("✓ seeded employee trainings")
	return nil
}

func seedBudgetTransactions(
	ctx context.Context, tx pgx.Tx,
	budgets []int64, ccs []int64,
	assets []int64, contracts []int64, licenses []int64,
	createdBy int64,
) error {

	type T struct {
		amt float64
		cat string
	}

	rows := []T{
		{750_000_000, "CAPEX"},
		{150_000_000, "OPEX"},
		{200_000_000, "OPEX"},
	}

	for i, r := range rows {
		bID := budgets[i%len(budgets)]
		cc := ccs[i%len(ccs)]

		var assetID, contractID, licenseID *int64

		if r.cat == "CAPEX" && len(contracts) > 0 {
			tmp := contracts[i%len(contracts)]
			contractID = &tmp
		}
		if r.cat == "OPEX" && len(assets) > 0 {
			tmp := assets[i%len(assets)]
			assetID = &tmp
		}
		if i == 2 && len(licenses) > 0 {
			tmp := licenses[i%len(licenses)]
			licenseID = &tmp
		}

		if err := ExecQ(ctx, tx, `
            INSERT INTO budget_transactions
            (budget_id, contract_id, license_id, asset_id,
             amount, currency, category, cost_center_id,
             created_by, created_at)
            VALUES ($1,$2,$3,$4,$5,'IDR',$6,$7,$8,NOW())`,
			bID, contractID, licenseID, assetID,
			r.amt, r.cat, cc, createdBy,
		); err != nil {
			return err
		}
	}

	log.Println("✓ seeded budget transactions")
	return nil
}

func seedTickets(
	ctx context.Context, tx pgx.Tx,
	emps []simpleRow, assets []int64, slaIDs []int64,
) error {

	type T struct {
		subj string
		desc string
		imp  string
		urg  string
		prio string
	}

	rows := []T{
		{"Core link down", "Backbone Jakarta-Bandung down", "High", "High", "Critical"},
		{"Portal slow", "ITSM portal slow", "Medium", "High", "High"},
		{"VPN request", "Install VPN for engineer", "Low", "Low", "Medium"},
	}

	now := time.Now()
	for i, r := range rows {
		reporter := emps[0]

		var assetID *int64
		if len(assets) > 0 {
			tmp := assets[i%len(assets)]
			assetID = &tmp
		}
		slaID := slaIDs[i%len(slaIDs)]

		if err := ExecQ(ctx, tx, `
            INSERT INTO tickets
            (subject,description,
             impact,urgency,priority,
             created_by_employee_id,
             related_asset_id,
             sla_policy_id,
             response_due_at, sla_due_at,
             compliance_flag, compliance_score,
             created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,
                    $6,$7,$8,$9,$10,
                    TRUE,100,NOW(),NOW())`,
			r.subj, r.desc,
			r.imp, r.urg, r.prio,
			reporter.ID, assetID, slaID,
			now.Add(30*time.Minute),
			now.Add(120*time.Minute),
		); err != nil {
			return err
		}
	}

	log.Println("✓ seeded tickets")
	return nil
}

func seedAlerts(ctx context.Context, tx pgx.Tx, emps []simpleRow, assets []int64) error {
	msgs := []string{
		"Core Link Down",
		"Contract Expiry Warning",
		"High CPU Router",
		"SLA Breach Portal",
	}

	for i, msg := range msgs {
		var assetID *int64
		if len(assets) > 0 && i < len(assets) {
			tmp := assets[i]
			assetID = &tmp
		}

		if err := ExecQ(ctx, tx, `
            INSERT INTO alerts
            (message, severity, category, acknowledged, acknowledged_by,
             asset_id, created_at)
            VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
			msg, "warning", "system", false, nil, assetID,
		); err != nil {
			return err
		}
	}

	log.Println("✓ seeded alerts")
	return nil
}

func seedAuditLogs(ctx context.Context, tx pgx.Tx, emps []simpleRow) error {
	actor := emps[0]

	entries := []struct {
		ent, act, changes, path string
	}{
		{"assets", "CREATE", `{"name":"Core Router"}`, "/api/v1/assets"},
		{"contracts", "UPDATE", `{"end_date":"2025-12-31"}`, "/api/v1/contracts/1"},
		{"licenses", "DELETE", `{"deleted":true}`, "/api/v1/licenses/1"},
		{"alerts", "ACK", `{"ack_by":1}`, "/api/v1/alerts/2/ack"},
		{"tickets", "CREATE", `{"subject":"Core link down"}`, "/api/v1/tickets"},
	}

	for _, e := range entries {
		if err := ExecQ(ctx, tx, `
            INSERT INTO audit_logs
            (entity_name, entity_id, action, actor_id,
             changes, ip_address, user_agent, request_path,
             created_at)
            VALUES ($1,NULL,$2,$3,$4,
                    '10.10.20.15','SeederBot/1.0',$5,
                    NOW())`,
			e.ent, e.act, actor.ID,
			e.changes, e.path,
		); err != nil {
			return err
		}
	}

	log.Println("✓ seeded audit logs")
	return nil
}
