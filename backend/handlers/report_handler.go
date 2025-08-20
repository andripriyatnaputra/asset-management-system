// File: backend/handlers/report_handler.go
package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

type AssetReportRow struct {
	AssetName    string `json:"asset_name"`
	AssetTag     string `json:"asset_tag"`
	AssetType    string `json:"asset_type"`
	EmployeeName string `json:"employee_name"`
	EmployeeNIK  string `json:"employee_nik"`
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
			e.employee_nik
		FROM asset_assignments aa
		JOIN assets a ON aa.asset_id = a.id
		JOIN asset_types at ON a.asset_type_id = at.id
		JOIN employees e ON aa.employee_id = e.id
		WHERE e.department_id = $1 AND aa.returned_at IS NULL AND a.deleted_at IS NULL
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
		if err := rows.Scan(&row.AssetName, &row.AssetTag, &row.AssetType, &row.EmployeeName, &row.EmployeeNIK); err != nil {
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
		writer.Write([]string{"Nama Karyawan", "NIK", "Nama Aset", "Tag Aset", "Tipe Aset"})
		// Tulis data
		for _, row := range results {
			writer.Write([]string{row.EmployeeName, row.EmployeeNIK, row.AssetName, row.AssetTag, row.AssetType})
		}
		writer.Flush()
	} else {
		// Jika tidak, kirim sebagai JSON biasa
		c.JSON(http.StatusOK, results)
	}
}
