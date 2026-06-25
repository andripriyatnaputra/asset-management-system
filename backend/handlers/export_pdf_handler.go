package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

// ── shared PDF helpers ───────────────────────────────────────────────────────

func newPDF(title string) *gofpdf.Fpdf {
	f := gofpdf.New("P", "mm", "A4", "")
	f.SetMargins(12, 14, 12)
	f.SetAutoPageBreak(true, 14)
	f.SetTitle(title, false)
	f.SetAuthor("IT Asset & Service Management System", false)
	f.AddPage()
	return f
}

// pdfHeader prints a branded page header then a thin rule.
func pdfHeader(f *gofpdf.Fpdf, title, subtitle string) {
	// dark navy bar
	f.SetFillColor(31, 78, 121)
	f.SetTextColor(255, 255, 255)
	f.SetFont("Arial", "B", 14)
	f.CellFormat(0, 10, title, "", 1, "C", true, 0, "")

	// subtitle + date
	f.SetFont("Arial", "", 9)
	f.SetFillColor(70, 130, 180)
	f.CellFormat(0, 6,
		fmt.Sprintf("%s  |  Dicetak: %s", subtitle, time.Now().Format("02 Jan 2006 15:04")),
		"", 1, "C", true, 0, "")

	// reset
	f.SetTextColor(0, 0, 0)
	f.Ln(3)
}

// pdfTableHeader prints one row of column headers with a blue-ish background.
func pdfTableHeader(f *gofpdf.Fpdf, cols []string, widths []float64) {
	f.SetFont("Arial", "B", 8)
	f.SetFillColor(41, 128, 185)
	f.SetTextColor(255, 255, 255)
	f.SetDrawColor(200, 200, 200)
	for i, col := range cols {
		f.CellFormat(widths[i], 7, col, "1", 0, "C", true, 0, "")
	}
	f.Ln(-1)
	f.SetTextColor(0, 0, 0)
	f.SetFont("Arial", "", 8)
}

// pdfRow prints one data row; even/odd rows alternate background.
func pdfRow(f *gofpdf.Fpdf, vals []string, widths []float64, even bool) {
	if even {
		f.SetFillColor(235, 245, 255)
	} else {
		f.SetFillColor(255, 255, 255)
	}
	for i, v := range vals {
		f.CellFormat(widths[i], 6, v, "1", 0, "L", true, 0, "")
	}
	f.Ln(-1)
}

// pdfFooter adds a simple page-number footer.
func pdfFooter(f *gofpdf.Fpdf) {
	f.SetFooterFunc(func() {
		f.SetY(-10)
		f.SetFont("Arial", "I", 7)
		f.SetTextColor(120, 120, 120)
		f.CellFormat(0, 5,
			fmt.Sprintf("Halaman %d  |  IT Asset & Service Management System", f.PageNo()),
			"", 0, "C", false, 0, "")
	})
}

// truncStr shortens a string for narrow columns.
func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// ── ExportAssetsPDF ──────────────────────────────────────────────────────────

// ExportAssetsPDF godoc
// @Summary Export daftar asset ke PDF
// @Tags Export
// @Produce application/pdf
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/assets.pdf [get]
func ExportAssetsPDF(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT a.id, a.name, a.asset_tag, a.serial_number,
		       COALESCE(at.name,''), a.status, a.lifecycle_stage,
		       COALESCE(e.full_name,''), COALESCE(l.name,''),
		       COALESCE(a.purchase_cost::text,'')
		FROM assets a
		LEFT JOIN asset_types at ON at.id = a.asset_type_id
		LEFT JOIN employees  e  ON e.id  = a.assigned_to
		LEFT JOIN locations  l  ON l.id  = a.location_id
		WHERE a.deleted_at IS NULL
		ORDER BY a.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := newPDF("Laporan Inventarisasi Aset IT")
	pdfFooter(f)
	pdfHeader(f, "LAPORAN INVENTARISASI ASET IT", "IT Asset & Service Management System")

	cols := []string{"ID", "Nama Aset", "Tag", "Tipe", "Status", "Stage", "Assigned To", "Lokasi", "Harga (Rp)"}
	widths := []float64{10, 42, 22, 22, 20, 18, 30, 22, 22}
	pdfTableHeader(f, cols, widths)

	even := false
	for rows.Next() {
		var id, name, tag, serial, assetType, status, stage, assignee, location, cost string
		if err := rows.Scan(&id, &name, &tag, &serial, &assetType, &status, &stage, &assignee, &location, &cost); err != nil {
			continue
		}
		pdfRow(f, []string{
			id,
			truncStr(name, 28),
			truncStr(tag, 14),
			truncStr(assetType, 14),
			truncStr(status, 13),
			truncStr(stage, 11),
			truncStr(assignee, 20),
			truncStr(location, 14),
			truncStr(cost, 14),
		}, widths, even)
		even = !even
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=assets_%s.pdf", time.Now().Format("20060102")))
	if err := f.Output(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menghasilkan PDF"})
	}
}

// ── ExportLicensesPDF ────────────────────────────────────────────────────────

// ExportLicensesPDF godoc
// @Summary Export laporan lisensi ke PDF (menyoroti yang akan expired)
// @Tags Export
// @Produce application/pdf
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/licenses.pdf [get]
func ExportLicensesPDF(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := database.Pool.Query(ctx, `
		SELECT l.id, l.name, COALESCE(l.vendor,''),
		       COALESCE(l.license_type,''), COALESCE(l.seats_total::text,'-'),
		       COALESCE(l.seats_used::text,'-'), COALESCE(l.status,''),
		       l.expiration_date,
		       CASE
		           WHEN l.expiration_date IS NULL THEN 'N/A'
		           WHEN l.expiration_date < now() THEN 'EXPIRED'
		           WHEN l.expiration_date < now() + INTERVAL '30 days' THEN 'SOON'
		           ELSE 'OK'
		       END AS expiry_flag
		FROM licenses l
		WHERE l.deleted_at IS NULL
		ORDER BY l.expiration_date NULLS LAST
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	f := newPDF("Laporan Lisensi Software")
	pdfFooter(f)
	pdfHeader(f, "LAPORAN LISENSI SOFTWARE", "IT Asset & Service Management System")

	// legend
	f.SetFont("Arial", "I", 8)
	f.SetTextColor(180, 0, 0)
	f.Cell(0, 5, "  Merah = EXPIRED   |  ")
	f.SetTextColor(200, 120, 0)
	f.Cell(0, 5, "  Oranye = Akan expired ≤30 hari   |  ")
	f.SetTextColor(0, 0, 0)
	f.Cell(0, 5, "  Hitam = OK")
	f.Ln(7)

	cols := []string{"ID", "Nama Lisensi", "Vendor", "Tipe", "Total", "Pakai", "Status", "Expiry", "Flag"}
	widths := []float64{10, 45, 28, 22, 12, 12, 18, 22, 14}
	pdfTableHeader(f, cols, widths)

	even := false
	for rows.Next() {
		var id, name, vendor, licType, total, used, status, flag string
		var expiryDate *time.Time
		if err := rows.Scan(&id, &name, &vendor, &licType, &total, &used, &status, &expiryDate, &flag); err != nil {
			continue
		}
		expiryStr := "-"
		if expiryDate != nil {
			expiryStr = expiryDate.Format("02/01/2006")
		}

		// colour-code expiry flag
		switch flag {
		case "EXPIRED":
			f.SetTextColor(180, 0, 0)
		case "SOON":
			f.SetTextColor(200, 100, 0)
		default:
			f.SetTextColor(0, 0, 0)
		}

		if even {
			f.SetFillColor(235, 245, 255)
		} else {
			f.SetFillColor(255, 255, 255)
		}
		vals := []string{id, truncStr(name, 30), truncStr(vendor, 18),
			truncStr(licType, 14), total, used, truncStr(status, 12), expiryStr, flag}
		for i, v := range vals {
			f.CellFormat(widths[i], 6, v, "1", 0, "L", true, 0, "")
		}
		f.Ln(-1)
		f.SetTextColor(0, 0, 0)
		even = !even
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=licenses_%s.pdf", time.Now().Format("20060102")))
	f.Output(c.Writer)
}

// ── ExportCompliancePDF ──────────────────────────────────────────────────────

// ExportCompliancePDF godoc
// @Summary Export laporan compliance framework ke PDF
// @Tags Export
// @Produce application/pdf
// @Success 200 {file} binary
// @Security BearerAuth
// @Router /export/compliance.pdf [get]
func ExportCompliancePDF(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Framework coverage summary
	fwRows, err := database.Pool.Query(ctx, `
		SELECT f.code, f.name,
		       COUNT(DISTINCT cc.id)                                                  AS total_ctrl,
		       COUNT(DISTINCT CASE WHEN ce.status='accepted' THEN cc.id END)         AS covered,
		       ROUND(
		           COUNT(DISTINCT CASE WHEN ce.status='accepted' THEN cc.id END)::NUMERIC /
		           NULLIF(COUNT(DISTINCT cc.id),0) * 100, 1
		       )                                                                       AS pct
		FROM compliance_frameworks f
		LEFT JOIN compliance_controls cc ON cc.framework_id = f.id AND cc.is_active
		LEFT JOIN compliance_evidence  ce ON ce.control_id  = cc.id
		WHERE f.is_active
		GROUP BY f.id, f.code, f.name
		ORDER BY f.code
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer fwRows.Close()

	type fwRow struct {
		Code, Name string
		Total, Covered int
		Pct *float64
	}
	var frameworks []fwRow
	for fwRows.Next() {
		var r fwRow
		fwRows.Scan(&r.Code, &r.Name, &r.Total, &r.Covered, &r.Pct)
		frameworks = append(frameworks, r)
	}
	fwRows.Close()

	// 2. Disposal compliance summary
	type dispCount struct{ Status string; Count int }
	dispRows, _ := database.Pool.Query(ctx, `
		SELECT compliance_status, COUNT(*) FROM v_asset_disposal_compliance
		GROUP BY compliance_status ORDER BY compliance_status
	`)
	var disposals []dispCount
	if dispRows != nil {
		defer dispRows.Close()
		for dispRows.Next() {
			var d dispCount
			dispRows.Scan(&d.Status, &d.Count)
			disposals = append(disposals, d)
		}
	}

	// ── build PDF ──
	f := newPDF("Laporan Compliance & Audit")
	pdfFooter(f)
	pdfHeader(f, "LAPORAN COMPLIANCE & AUDIT", "IT Asset & Service Management System")

	// ── Section 1: Framework Coverage ──
	f.SetFont("Arial", "B", 10)
	f.SetFillColor(52, 73, 94)
	f.SetTextColor(255, 255, 255)
	f.CellFormat(0, 7, "  1. Coverage Framework Compliance", "", 1, "L", true, 0, "")
	f.SetTextColor(0, 0, 0)
	f.Ln(2)

	cols := []string{"Kode", "Framework", "Total Controls", "Covered", "Coverage %", "Status"}
	widths := []float64{22, 68, 28, 22, 24, 22}
	pdfTableHeader(f, cols, widths)

	even := false
	for _, r := range frameworks {
		pctStr := "-"
		statusStr := "N/A"
		var textR, textG, textB int = 0, 0, 0

		if r.Pct != nil {
			pctStr = fmt.Sprintf("%.1f%%", *r.Pct)
			switch {
			case *r.Pct >= 80:
				statusStr = "Good"
				textR, textG, textB = 0, 128, 0
			case *r.Pct >= 50:
				statusStr = "Partial"
				textR, textG, textB = 180, 100, 0
			default:
				statusStr = "Low"
				textR, textG, textB = 180, 0, 0
			}
		}

		if even {
			f.SetFillColor(235, 245, 255)
		} else {
			f.SetFillColor(255, 255, 255)
		}

		rowData := []string{r.Code, truncStr(r.Name, 42), fmt.Sprintf("%d", r.Total),
			fmt.Sprintf("%d", r.Covered), pctStr, ""}
		for i, v := range rowData {
			if i == 5 { // status col — coloured
				f.SetTextColor(textR, textG, textB)
				f.CellFormat(widths[i], 6, statusStr, "1", 0, "C", true, 0, "")
				f.SetTextColor(0, 0, 0)
			} else {
				f.CellFormat(widths[i], 6, v, "1", 0, "L", true, 0, "")
			}
		}
		f.Ln(-1)
		even = !even
	}

	// ── Section 2: Disposal Compliance ──
	f.Ln(5)
	f.SetFont("Arial", "B", 10)
	f.SetFillColor(52, 73, 94)
	f.SetTextColor(255, 255, 255)
	f.CellFormat(0, 7, "  2. Ringkasan Disposal Compliance", "", 1, "L", true, 0, "")
	f.SetTextColor(0, 0, 0)
	f.Ln(2)

	if len(disposals) == 0 {
		f.SetFont("Arial", "I", 9)
		f.Cell(0, 6, "  Tidak ada data disposal compliance.")
		f.Ln(8)
	} else {
		dCols := []string{"Status Compliance", "Jumlah Aset"}
		dWidths := []float64{100, 40}
		pdfTableHeader(f, dCols, dWidths)
		even = false
		for _, d := range disposals {
			if even {
				f.SetFillColor(235, 245, 255)
			} else {
				f.SetFillColor(255, 255, 255)
			}
			f.CellFormat(dWidths[0], 6, d.Status, "1", 0, "L", true, 0, "")
			f.CellFormat(dWidths[1], 6, fmt.Sprintf("%d", d.Count), "1", 0, "C", true, 0, "")
			f.Ln(-1)
			even = !even
		}
	}

	// ── Section 3: Generated info ──
	f.Ln(5)
	f.SetFont("Arial", "I", 8)
	f.SetTextColor(120, 120, 120)
	f.Cell(0, 5, fmt.Sprintf("Laporan ini digenerate otomatis pada %s", time.Now().Format("02 January 2006 pukul 15:04 WIB")))

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=compliance_report_%s.pdf", time.Now().Format("20060102")))
	if err := f.Output(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menghasilkan PDF"})
	}
}
