package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ExportAssetsExcel godoc
// @Summary Export daftar asset ke Excel
// @Tags Export
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/assets.xlsx [get]
func ExportAssetsExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT a.id, a.name, a.asset_tag, a.serial_number,
		       at.name AS asset_type,
		       a.status, a.lifecycle_stage, a.purchase_date, a.purchase_cost,
		       l.name AS location,
		       e.full_name AS assigned_to,
		       a.warranty_expiry, a.updated_at
		FROM assets a
		LEFT JOIN asset_types at ON at.id = a.asset_type_id
		LEFT JOIN locations l ON l.id = a.location_id
		LEFT JOIN employees e ON e.id = a.assigned_to
		WHERE a.deleted_at IS NULL
		ORDER BY a.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Assets"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "Nama", "Asset Tag", "Serial Number", "Tipe Asset",
		"Status", "Lifecycle Stage", "Tanggal Beli", "Harga Beli",
		"Lokasi", "Assigned To", "Warranty Expiry", "Last Updated",
	}
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
		f.SetColWidth(sheet, col, col, 18)
	}

	row := 2
	for rows.Next() {
		var (
			id, name, tag, serial, assetType, status, stage string
			purchaseDate, warrantyExpiry, updatedAt          *time.Time
			purchaseCost                                      *float64
			location, assignedTo                             *string
		)
		if err := rows.Scan(&id, &name, &tag, &serial, &assetType, &status, &stage,
			&purchaseDate, &purchaseCost, &location, &assignedTo,
			&warrantyExpiry, &updatedAt); err != nil {
			continue
		}
		vals := []interface{}{
			id, name, tag, serial, assetType, status, stage,
			nullableTime(purchaseDate), nullableFloat(purchaseCost),
			nullableString(location), nullableString(assignedTo),
			nullableTime(warrantyExpiry), nullableTime(updatedAt),
		}
		for i, v := range vals {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), v)
		}
		row++
	}

	f.SetActiveSheet(0)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=assets_%s.xlsx", time.Now().Format("20060102")))
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menulis file Excel"})
	}
}

// ExportLicensesExcel godoc
// @Summary Export daftar lisensi ke Excel
// @Tags Export
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/licenses.xlsx [get]
func ExportLicensesExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT l.id, l.name, l.vendor, l.product_key, l.license_type,
		       l.seats_total, l.seats_used, l.expiration_date, l.status,
		       l.updated_at
		FROM licenses l
		WHERE l.deleted_at IS NULL
		ORDER BY l.expiration_date NULLS LAST
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Licenses"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Nama", "Vendor", "Product Key", "Tipe", "Total Seat", "Terpakai", "Expiry Date", "Status", "Updated"}
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"2E75B6"}, Pattern: 1},
	})
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
		f.SetColWidth(sheet, col, col, 16)
	}

	row := 2
	for rows.Next() {
		var (
			id, name, vendor, key, licType, status string
			seatsTotal, seatsUsed                   *int
			expiryDate, updatedAt                   *time.Time
		)
		if err := rows.Scan(&id, &name, &vendor, &key, &licType,
			&seatsTotal, &seatsUsed, &expiryDate, &status, &updatedAt); err != nil {
			continue
		}
		vals := []interface{}{id, name, vendor, key, licType,
			nullableInt(seatsTotal), nullableInt(seatsUsed),
			nullableTime(expiryDate), status, nullableTime(updatedAt)}
		for i, v := range vals {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), v)
		}
		row++
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=licenses_%s.xlsx", time.Now().Format("20060102")))
	f.Write(c.Writer)
}

// ExportAuditLogsExcel godoc
// @Summary Export audit log ke Excel (maks 5000 baris terbaru)
// @Tags Export
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/audit-logs.xlsx [get]
func ExportAuditLogsExcel(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT al.id, al.entity_type, al.entity_id, al.action,
		       e.full_name AS changed_by,
		       al.changed_at
		FROM audit_logs al
		LEFT JOIN employees e ON e.id = al.changed_by
		ORDER BY al.changed_at DESC
		LIMIT 5000
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	sheet := "Audit Logs"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"ID", "Entitas", "Entity ID", "Aksi", "Dilakukan Oleh", "Waktu"}
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"375623"}, Pattern: 1},
	})
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		cell := fmt.Sprintf("%s1", col)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, style)
		f.SetColWidth(sheet, col, col, 20)
	}

	row := 2
	for rows.Next() {
		var (
			id, entityType, action string
			entityID               int64
			changedBy              *string
			changedAt              time.Time
		)
		if err := rows.Scan(&id, &entityType, &entityID, &action, &changedBy, &changedAt); err != nil {
			continue
		}
		vals := []interface{}{id, entityType, entityID, action, nullableString(changedBy), changedAt.Format("2006-01-02 15:04:05")}
		for i, v := range vals {
			col, _ := excelize.ColumnNumberToName(i + 1)
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, row), v)
		}
		row++
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=audit_logs_%s.xlsx", time.Now().Format("20060102")))
	f.Write(c.Writer)
}

// --- helpers ---

func nullableString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullableFloat(f *float64) interface{} {
	if f == nil {
		return ""
	}
	return *f
}

func nullableInt(i *int) interface{} {
	if i == nil {
		return ""
	}
	return *i
}

func nullableTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
