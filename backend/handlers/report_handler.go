// File: backend/handlers/report_handler.go
package handlers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

type AssetReportRow struct {
	AssetName    string    `json:"asset_name"`
	AssetTag     string    `json:"asset_tag"`
	AssetType    string    `json:"asset_type"`
	EmployeeName string    `json:"employee_name"`
	EmployeeNIK  string    `json:"employee_nik"`
	AssignedAt   time.Time `json:"assigned_at"`
}

// GetAssetsByDepartmentReport generates a report of assets assigned to a department
func GetAssetsByDepartmentReport(c *gin.Context) {
	departmentIDStr := c.Query("department_id")
	if departmentIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "department_id is required"})
		return
	}
	departmentID, err := strconv.Atoi(departmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department_id"})
		return
	}

	query := `
			SELECT 
			a.name as asset_name,
			a.asset_tag,
			at.name as asset_type,
			e.name as employee_name,
			e.employee_nik,
			aa.assigned_at
		FROM asset_assignments aa
		JOIN assets a ON aa.asset_id = a.id
		JOIN asset_types at ON a.asset_type_id = at.id
		JOIN employees e ON aa.employee_id = e.id
		WHERE e.department_id = $1 
		AND aa.returned_at IS NULL 
		AND a.deleted_at IS NULL
		ORDER BY e.name, a.name`

	rows, err := database.Pool.Query(context.Background(), query, departmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch report data"})
		return
	}
	defer rows.Close()

	var results []AssetReportRow
	for rows.Next() {
		var row AssetReportRow
		if err := rows.Scan(
			&row.AssetName,
			&row.AssetTag,
			&row.AssetType,
			&row.EmployeeName,
			&row.EmployeeNIK,
			&row.AssignedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan report data"})
			return
		}
		results = append(results, row)
	}

	// Cek apakah ada permintaan untuk ekspor ke CSV
	if c.Query("export") == "csv" {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=report_assets_dept_%d_%s.csv", departmentID, time.Now().Format("2006-01-02")))
		c.Header("Content-Type", "text/csv")

		writer := csv.NewWriter(c.Writer)
		// Tulis header CSV
		writer.Write([]string{"Nama Karyawan", "NIK", "Nama Aset", "Tag Aset", "Tipe Aset", "Tanggal Diberikan"})
		for _, row := range results {
			writer.Write([]string{
				row.EmployeeName,
				row.EmployeeNIK,
				row.AssetName,
				row.AssetTag,
				row.AssetType,
				row.AssignedAt.Format("2006-01-02"),
			})
		}
		writer.Flush()
	} else {
		// Jika tidak, kirim sebagai JSON biasa
		c.JSON(http.StatusOK, results)
	}
}

func GetAssetsByEmployeeReport(c *gin.Context) {
	employeeIDStr := c.Query("employee_id")
	if employeeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee_id is required"})
		return
	}

	query := `
        SELECT 
            a.name as asset_name,
            a.asset_tag,
            at.name as asset_type,
            aa.assigned_at
        FROM asset_assignments aa
        JOIN assets a ON aa.asset_id = a.id
        JOIN asset_types at ON a.asset_type_id = at.id
        WHERE aa.employee_id = $1 AND aa.returned_at IS NULL AND a.deleted_at IS NULL
        ORDER BY aa.assigned_at DESC`

	rows, err := database.Pool.Query(context.Background(), query, employeeIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch report data"})
		return
	}
	defer rows.Close()

	// Kita gunakan map[string]interface{} agar lebih fleksibel
	var results []map[string]interface{}
	for rows.Next() {
		var row AssetReportRow // Kita bisa pakai ulang struct yang ada
		if err := rows.Scan(&row.AssetName, &row.AssetTag, &row.AssetType, &row.AssignedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan report data"})
			return
		}
		results = append(results, map[string]interface{}{
			"asset_name":  row.AssetName,
			"asset_tag":   row.AssetTag,
			"asset_type":  row.AssetType,
			"assigned_at": row.AssignedAt,
		})
	}

	c.JSON(http.StatusOK, results)
}

func GetTicketsByAssetTypeReport(c *gin.Context) {
	query := `
		SELECT 
			at.name as asset_type,
			COUNT(t.id) as ticket_count
		FROM tickets t
		JOIN assets a ON t.related_asset_id = a.id
		JOIN asset_types at ON a.asset_type_id = at.id
		WHERE t.deleted_at IS NULL
		GROUP BY at.name
		ORDER BY ticket_count DESC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch report data"})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var assetType string
		var ticketCount int
		if err := rows.Scan(&assetType, &ticketCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan report data"})
			return
		}
		results = append(results, map[string]interface{}{
			"asset_type":   assetType,
			"ticket_count": ticketCount,
		})
	}

	c.JSON(http.StatusOK, results)
}

// GET /api/v1/assets/compliance-summary
func GetComplianceSummaryReport(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT id, name, asset_tag, department_name, owner_department_name,
		        compliance_flag, compliance_note, updated_at
		   FROM compliance_summary
		   ORDER BY updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch compliance summary"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID                  int64      `json:"id"`
		Name                string     `json:"name"`
		AssetTag            string     `json:"asset_tag"`
		DepartmentName      *string    `json:"department_name"`
		OwnerDepartmentName *string    `json:"owner_department_name"`
		ComplianceFlag      *bool      `json:"compliance_flag"`
		ComplianceNote      *string    `json:"compliance_note"`
		UpdatedAt           *time.Time `json:"updated_at"`
	}

	var list []Row
	var compliant, nonCompliant, pending int64
	for rows.Next() {
		var r Row
		_ = rows.Scan(&r.ID, &r.Name, &r.AssetTag,
			&r.DepartmentName, &r.OwnerDepartmentName,
			&r.ComplianceFlag, &r.ComplianceNote, &r.UpdatedAt)
		list = append(list, r)
		switch {
		case r.ComplianceFlag == nil:
			pending++
		case *r.ComplianceFlag:
			compliant++
		default:
			nonCompliant++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"summary": gin.H{
			"compliant":      compliant,
			"non_compliant":  nonCompliant,
			"pending":        pending,
			"total":          compliant + nonCompliant + pending,
			"last_refreshed": time.Now(),
		},
	})
}

// GET /api/v1/assets/compliance-export
func ExportSystemComplianceCSV(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT name, asset_tag, status, lifecycle_stage, 
		       compliance_flag, compliance_note, updated_at
		FROM assets WHERE deleted_at IS NULL ORDER BY updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export CSV"})
		return
	}
	defer rows.Close()

	c.Header("Content-Disposition", fmt.Sprintf(
		"attachment; filename=compliance_report_%s.csv", time.Now().Format("2006-01-02")))
	c.Header("Content-Type", "text/csv; charset=utf-8")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"Asset Name", "Tag", "Status", "Lifecycle Stage", "Compliant", "Note", "Updated At"})
	for rows.Next() {
		var name, tag, status, lifecycle, note sql.NullString
		var compliant sql.NullBool
		var updated time.Time
		_ = rows.Scan(&name, &tag, &status, &lifecycle, &compliant, &note, &updated)
		writer.Write([]string{
			name.String,
			tag.String,
			status.String,
			lifecycle.String,
			strconv.FormatBool(compliant.Bool),
			note.String,
			updated.Format("2006-01-02 15:04"),
		})
	}
	writer.Flush()
}

// GET /api/v1/audit/security
func GetSecurityAuditLogs(c *gin.Context) {
	qUser := c.Query("user_id")
	qAction := c.Query("action")
	qStart := c.Query("start_date")
	qEnd := c.Query("end_date")

	query := `
		SELECT id, entity_name, action, actor_id, request_path, created_at
		  FROM v_security_audit
		 WHERE 1=1
	`

	var params []any
	paramIdx := 1

	// 🔹 Filter by user
	if qUser != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", paramIdx)
		params = append(params, qUser)
		paramIdx++
	}

	// 🔹 Filter by action
	if qAction != "" {
		query += fmt.Sprintf(" AND LOWER(action) = LOWER($%d)", paramIdx)
		params = append(params, qAction)
		paramIdx++
	}

	// 🔹 Filter by start / end date (parsial juga bisa)
	if qStart != "" && qEnd != "" {
		query += fmt.Sprintf(" AND created_at BETWEEN $%d AND $%d", paramIdx, paramIdx+1)
		params = append(params, qStart, qEnd)
		paramIdx += 2
	} else if qStart != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", paramIdx)
		params = append(params, qStart)
		paramIdx++
	} else if qEnd != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", paramIdx)
		params = append(params, qEnd)
		paramIdx++
	}

	query += " ORDER BY created_at DESC LIMIT 500"

	rows, err := database.Pool.Query(c.Request.Context(), query, params...)
	if err != nil {
		log.Printf("[ERROR][GetSecurityAuditLogs] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID          int64      `json:"id"`
		EntityName  string     `json:"entity_name"`
		Action      string     `json:"action"`
		ActorID     *int64     `json:"actor_id"`
		RequestPath string     `json:"request_path"`
		CreatedAt   *time.Time `json:"created_at"`
	}

	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.EntityName, &r.Action, &r.ActorID, &r.RequestPath, &r.CreatedAt); err != nil {
			log.Printf("[ERROR][GetSecurityAuditLogs] scan failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GET /api/v1/audit/security/meta
// File: backend/handlers/security_audit_handler.go
func GetSecurityAuditMeta(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT DISTINCT a.actor_id, COALESCE(e.name,'') AS actor_name, LOWER(a.action) AS action
		  FROM v_security_audit a
		  LEFT JOIN employees e ON e.id = a.actor_id
		 WHERE a.actor_id IS NOT NULL
		 ORDER BY a.actor_id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch meta"})
		return
	}
	defer rows.Close()

	type item struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Act  string `json:"action"`
	}
	actorsMap := map[int64]string{}
	actionsSet := map[string]struct{}{}

	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.Name, &it.Act); err != nil {
			continue
		}
		actorsMap[it.ID] = it.Name
		if it.Act != "" {
			actionsSet[it.Act] = struct{}{}
		}
	}
	actors := make([]map[string]any, 0, len(actorsMap))
	for id, name := range actorsMap {
		actors = append(actors, map[string]any{"id": id, "name": name})
	}
	actions := make([]string, 0, len(actionsSet))
	for a := range actionsSet {
		actions = append(actions, a)
	}

	c.JSON(http.StatusOK, gin.H{"actors": actors, "actions": actions})
}
