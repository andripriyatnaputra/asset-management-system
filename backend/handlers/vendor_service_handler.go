package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// ============================================================
// VENDOR PERFORMANCE
// ============================================================

// GetAllVendorPerformance godoc
// @Summary List performa vendor
// @Tags Vendor & Service
// @Produce json
// @Param vendor query string false "Filter by vendor name"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} pagedResponse
// @Security BearerAuth
// @Router /vendors/performance [get]
func GetAllVendorPerformance(c *gin.Context) {
	vendor := c.Query("vendor")
	pg := getPagination(c)

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if vendor != "" {
		where += fmt.Sprintf(" AND vp.vendor_name ILIKE $%d", idx)
		args = append(args, "%"+vendor+"%")
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(*) FROM vendor_performance vp "+where, args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT vp.id, vp.vendor_name, vp.contract_id,
		       vp.period_start, vp.period_end,
		       vp.sla_compliance_pct, vp.avg_response_hours,
		       vp.total_tickets, vp.open_tickets, vp.critical_incidents,
		       vp.nps_score, vp.notes, vp.recorded_by, vp.created_at, vp.updated_at,
		       c.contract_number,
		       e.name AS recorded_by_name
		FROM vendor_performance vp
		LEFT JOIN contracts c  ON c.id = vp.contract_id
		LEFT JOIN employees e  ON e.id = vp.recorded_by
		%s
		ORDER BY vp.period_start DESC, vp.vendor_name
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.VendorPerformance
	for rows.Next() {
		var vp models.VendorPerformance
		if err := rows.Scan(
			&vp.ID, &vp.VendorName, &vp.ContractID,
			&vp.PeriodStart, &vp.PeriodEnd,
			&vp.SLACompliancePct, &vp.AvgResponseHours,
			&vp.TotalTickets, &vp.OpenTickets, &vp.CriticalIncidents,
			&vp.NPSScore, &vp.Notes, &vp.RecordedBy, &vp.CreatedAt, &vp.UpdatedAt,
			&vp.ContractNumber, &vp.RecordedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, vp)
	}
	if list == nil {
		list = []models.VendorPerformance{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

func GetVendorPerformanceByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var vp models.VendorPerformance
	err := database.Pool.QueryRow(c, `
		SELECT vp.id, vp.vendor_name, vp.contract_id,
		       vp.period_start, vp.period_end,
		       vp.sla_compliance_pct, vp.avg_response_hours,
		       vp.total_tickets, vp.open_tickets, vp.critical_incidents,
		       vp.nps_score, vp.notes, vp.recorded_by, vp.created_at, vp.updated_at,
		       c.contract_number,
		       e.name AS recorded_by_name
		FROM vendor_performance vp
		LEFT JOIN contracts c ON c.id = vp.contract_id
		LEFT JOIN employees e ON e.id = vp.recorded_by
		WHERE vp.id = $1
	`, id).Scan(
		&vp.ID, &vp.VendorName, &vp.ContractID,
		&vp.PeriodStart, &vp.PeriodEnd,
		&vp.SLACompliancePct, &vp.AvgResponseHours,
		&vp.TotalTickets, &vp.OpenTickets, &vp.CriticalIncidents,
		&vp.NPSScore, &vp.Notes, &vp.RecordedBy, &vp.CreatedAt, &vp.UpdatedAt,
		&vp.ContractNumber, &vp.RecordedByName,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vendor performance tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, vp)
}

func CreateVendorPerformance(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		VendorName        string   `json:"vendor_name" binding:"required"`
		ContractID        *int64   `json:"contract_id"`
		PeriodStart       string   `json:"period_start" binding:"required"` // YYYY-MM-DD
		PeriodEnd         string   `json:"period_end" binding:"required"`
		SLACompliancePct  *float64 `json:"sla_compliance_pct"`
		AvgResponseHours  *float64 `json:"avg_response_hours"`
		TotalTickets      int      `json:"total_tickets"`
		OpenTickets       int      `json:"open_tickets"`
		CriticalIncidents int      `json:"critical_incidents"`
		NPSScore          *int     `json:"nps_score"`
		Notes             *string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ps, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format period_start harus YYYY-MM-DD"})
		return
	}
	pe, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format period_end harus YYYY-MM-DD"})
		return
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO vendor_performance (
			vendor_name, contract_id, period_start, period_end,
			sla_compliance_pct, avg_response_hours,
			total_tickets, open_tickets, critical_incidents,
			nps_score, notes, recorded_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id
	`, req.VendorName, req.ContractID, ps, pe,
		req.SLACompliancePct, req.AvgResponseHours,
		req.TotalTickets, req.OpenTickets, req.CriticalIncidents,
		req.NPSScore, req.Notes, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "vendor performance berhasil dicatat"})
}

func UpdateVendorPerformance(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		SLACompliancePct  *float64 `json:"sla_compliance_pct"`
		AvgResponseHours  *float64 `json:"avg_response_hours"`
		TotalTickets      *int     `json:"total_tickets"`
		OpenTickets       *int     `json:"open_tickets"`
		CriticalIncidents *int     `json:"critical_incidents"`
		NPSScore          *int     `json:"nps_score"`
		Notes             *string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE vendor_performance SET
			sla_compliance_pct  = COALESCE($1, sla_compliance_pct),
			avg_response_hours  = COALESCE($2, avg_response_hours),
			total_tickets       = COALESCE($3, total_tickets),
			open_tickets        = COALESCE($4, open_tickets),
			critical_incidents  = COALESCE($5, critical_incidents),
			nps_score           = COALESCE($6, nps_score),
			notes               = COALESCE($7, notes),
			updated_at          = now()
		WHERE id = $8
	`, req.SLACompliancePct, req.AvgResponseHours,
		req.TotalTickets, req.OpenTickets, req.CriticalIncidents,
		req.NPSScore, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "vendor performance berhasil diupdate"})
}

func DeleteVendorPerformance(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM vendor_performance WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "vendor performance berhasil dihapus"})
}

// ============================================================
// SERVICE AVAILABILITY
// ============================================================

// GetServiceAvailability godoc
// @Summary List data availability layanan
// @Tags Vendor & Service
// @Produce json
// @Param service_code query string false "Filter by service code"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} pagedResponse
// @Security BearerAuth
// @Router /services/availability [get]
func GetServiceAvailability(c *gin.Context) {
	serviceCode := c.Query("service_code")
	pg := getPagination(c)

	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if serviceCode != "" {
		where += fmt.Sprintf(" AND sa.service_code = $%d", idx)
		args = append(args, serviceCode)
		idx++
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(*) FROM service_availability sa "+where, args...).Scan(&total)

	dataArgs := append(args, pg.Limit, pg.Offset)
	query := fmt.Sprintf(`
		SELECT sa.id, sa.service_code, sa.period_start, sa.period_end,
		       sa.downtime_minutes, sa.planned_downtime_minutes, sa.incident_count,
		       sa.availability_pct, sa.notes, sa.recorded_by, sa.created_at, sa.updated_at,
		       s.name AS service_name,
		       e.name AS recorded_by_name
		FROM service_availability sa
		JOIN services s       ON s.code = sa.service_code
		LEFT JOIN employees e ON e.id = sa.recorded_by
		%s
		ORDER BY sa.period_start DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.Pool.Query(c, query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ServiceAvailability
	for rows.Next() {
		var sa models.ServiceAvailability
		if err := rows.Scan(
			&sa.ID, &sa.ServiceCode, &sa.PeriodStart, &sa.PeriodEnd,
			&sa.DowntimeMinutes, &sa.PlannedDowntimeMinutes, &sa.IncidentCount,
			&sa.AvailabilityPct, &sa.Notes, &sa.RecordedBy, &sa.CreatedAt, &sa.UpdatedAt,
			&sa.ServiceName, &sa.RecordedByName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, sa)
	}
	if list == nil {
		list = []models.ServiceAvailability{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

func GetServiceAvailabilitySummary(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT sa.service_code, s.name,
		       ROUND(AVG(sa.availability_pct), 4)   AS avg_avail_pct,
		       SUM(sa.downtime_minutes)              AS total_downtime,
		       SUM(sa.incident_count)                AS total_incidents,
		       COUNT(*)                              AS period_count
		FROM service_availability sa
		JOIN services s ON s.code = sa.service_code
		GROUP BY sa.service_code, s.name
		ORDER BY avg_avail_pct ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.ServiceAvailabilitySummary
	for rows.Next() {
		var s models.ServiceAvailabilitySummary
		if err := rows.Scan(&s.ServiceCode, &s.ServiceName,
			&s.AvgAvailPct, &s.TotalDowntime, &s.TotalIncidents, &s.PeriodCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, s)
	}
	if list == nil {
		list = []models.ServiceAvailabilitySummary{}
	}
	c.JSON(http.StatusOK, list)
}

func RecordServiceAvailability(c *gin.Context) {
	actor := getActorID(c)

	var req struct {
		ServiceCode            string  `json:"service_code" binding:"required"`
		PeriodStart            string  `json:"period_start" binding:"required"` // RFC3339
		PeriodEnd              string  `json:"period_end" binding:"required"`
		DowntimeMinutes        int     `json:"downtime_minutes"`
		PlannedDowntimeMinutes int     `json:"planned_downtime_minutes"`
		IncidentCount          int     `json:"incident_count"`
		Notes                  *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ps, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		ps, err = time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format period_start: RFC3339 atau YYYY-MM-DD"})
			return
		}
	}
	pe, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		pe, err = time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format period_end: RFC3339 atau YYYY-MM-DD"})
			return
		}
	}

	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO service_availability
			(service_code, period_start, period_end, downtime_minutes,
			 planned_downtime_minutes, incident_count, notes, recorded_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, req.ServiceCode, ps, pe, req.DowntimeMinutes,
		req.PlannedDowntimeMinutes, req.IncidentCount, req.Notes, actor,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "service availability berhasil dicatat"})
}

func UpdateServiceAvailability(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req struct {
		DowntimeMinutes        *int    `json:"downtime_minutes"`
		PlannedDowntimeMinutes *int    `json:"planned_downtime_minutes"`
		IncidentCount          *int    `json:"incident_count"`
		Notes                  *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE service_availability SET
			downtime_minutes         = COALESCE($1, downtime_minutes),
			planned_downtime_minutes = COALESCE($2, planned_downtime_minutes),
			incident_count           = COALESCE($3, incident_count),
			notes                    = COALESCE($4, notes),
			updated_at               = now()
		WHERE id = $5
	`, req.DowntimeMinutes, req.PlannedDowntimeMinutes, req.IncidentCount, req.Notes, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service availability berhasil diupdate"})
}

func DeleteServiceAvailability(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM service_availability WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service availability berhasil dihapus"})
}
