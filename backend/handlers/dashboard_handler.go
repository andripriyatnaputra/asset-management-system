package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
)

type StatCard struct {
	Title string `json:"title"`
	Value int64  `json:"value"`
}

type RecentActivity struct {
	AssetName    string `json:"asset_name"`
	EmployeeName string `json:"employee_name"`
	AssignedAt   string `json:"assigned_at"`
}

type ChartData struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// ============================================================
// 🔹 Extended asset metrics per department (for dashboard)
// ============================================================
type Metric struct {
	Name           string  `json:"name"`
	AvgHealth      float64 `json:"avg_health"`
	AvgGovernance  float64 `json:"avg_governance"`
	ComplianceRate float64 `json:"compliance_rate"`
	TotalAssets    int64   `json:"total_assets"`
}

type DashboardStats struct {
	StatCards          []StatCard                   `json:"stat_cards"`
	RecentActivity     []RecentActivity             `json:"recent_activity"`
	AssetsByType       []ChartData                  `json:"assets_by_type"`
	EmployeesByDept    []ChartData                  `json:"employees_by_dept"`
	AssetMetricsByDept []Metric                     `json:"asset_metrics_by_dept"`
	PredictiveRisk     []services.AssetRiskForecast `json:"predictive_risk"`
	Compliance         []ChartData                  `json:"compliance"`
}

// ============================================================
// MAIN HANDLER: GET /dashboard/stats
// ============================================================
func GetDashboardStats(c *gin.Context) {
	var stats DashboardStats

	roleVal, roleExists := c.Get("role")
	_, userExists := c.Get("user_id")
	if !roleExists || !userExists || roleVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 1️⃣ Stat cards
	if cards, err := getStatCards(c); err == nil {
		stats.StatCards = cards
	} else {
		log.Printf("[WARN] getStatCards: %v", err)
	}

	// 2️⃣ Alert summary as additional stat cards
	if alertCards, err := getAlertStatsSummary(c); err == nil {
		stats.StatCards = append(stats.StatCards, alertCards...)
	} else {
		log.Printf("[WARN] getAlertStatsSummary: %v", err)
	}

	// 3️⃣ Recent activities
	if recent, err := getRecentActivities(c); err == nil {
		stats.RecentActivity = recent
	} else {
		log.Printf("[WARN] getRecentActivities: %v", err)
	}

	// 4️⃣ Assets by type
	if byType, err := getAssetsByType(c); err == nil {
		stats.AssetsByType = byType
	} else {
		log.Printf("[WARN] getAssetsByType: %v", err)
	}

	// 5️⃣ Employees by department
	if byDept, err := getEmployeesByDept(c); err == nil {
		stats.EmployeesByDept = byDept
	} else {
		log.Printf("[WARN] getEmployeesByDept: %v", err)
	}

	// 6️⃣ Extended metrics per department
	if metrics, err := getAssetMetricsByDept(c); err == nil {
		stats.AssetMetricsByDept = metrics
	} else {
		log.Printf("[WARN] getAssetMetricsByDept: %v", err)
	}

	// 7️⃣ Predictive risk (services)
	if riskData, err := services.ComputeAssetRiskForecast(c.Request.Context()); err == nil {
		stats.PredictiveRisk = riskData
	} else {
		log.Printf("[WARN] ComputeAssetRiskForecast: %v", err)
	}

	// 8️⃣ Governance / compliance distribution
	if governance, err := getGovernanceCompliance(c); err == nil {
		stats.Compliance = governance
		log.Printf("[DEBUG] Compliance data count = %d", len(stats.Compliance))
	} else {
		log.Printf("[WARN] getGovernanceCompliance: %v", err)
	}

	c.JSON(http.StatusOK, stats)
}

// ============================================================
// 1️⃣ Stat Cards
// ============================================================
func getStatCards(c *gin.Context) ([]StatCard, error) {
	cards := []StatCard{
		{Title: "Total Aset"},
		{Title: "Aset Dipinjam"},
		{Title: "Aset Tersedia"},
		{Title: "Total Karyawan"},
	}
	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL),0),
			COALESCE((SELECT COUNT(*) FROM assets WHERE LOWER(status) IN ('assigned','in_use') AND deleted_at IS NULL),0),
			COALESCE((SELECT COUNT(*) FROM assets WHERE LOWER(status) IN ('in_stock','available') AND deleted_at IS NULL),0),
			COALESCE((SELECT COUNT(*) FROM employees WHERE deleted_at IS NULL),0)
	`
	err := database.Pool.QueryRow(c.Request.Context(), query).
		Scan(&cards[0].Value, &cards[1].Value, &cards[2].Value, &cards[3].Value)
	if err != nil {
		return nil, fmt.Errorf("stat cards query failed: %w", err)
	}
	return cards, nil
}

// ============================================================
// 2️⃣ Recent Activities
// ============================================================
func getRecentActivities(c *gin.Context) ([]RecentActivity, error) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT 
			COALESCE(a.name,'-'),
			COALESCE(e.name,'-'),
			COALESCE(to_char(aa.assigned_at,'YYYY-MM-DD'),'')
		FROM asset_assignments aa
		LEFT JOIN assets a ON aa.asset_id=a.id AND a.deleted_at IS NULL
		LEFT JOIN employees e ON aa.employee_id=e.id AND e.deleted_at IS NULL
		WHERE aa.returned_at IS NULL
		ORDER BY aa.assigned_at DESC
		LIMIT 10;
	`)
	if err != nil {
		return nil, fmt.Errorf("recent activities query failed: %w", err)
	}
	defer rows.Close()

	var list []RecentActivity
	for rows.Next() {
		var r RecentActivity
		if err := rows.Scan(&r.AssetName, &r.EmployeeName, &r.AssignedAt); err != nil {
			log.Printf("[WARN] scan recent activity: %v", err)
			continue
		}
		list = append(list, r)
	}
	return list, nil
}

// ============================================================
// 3️⃣ Assets by Type
// ============================================================
func getAssetsByType(c *gin.Context) ([]ChartData, error) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT COALESCE(at.name,'Tidak Diketahui') AS name,
		       COUNT(a.id)::bigint AS value
		FROM asset_types at
		LEFT JOIN assets a ON a.asset_type_id=at.id AND a.deleted_at IS NULL
		GROUP BY at.name
		ORDER BY value DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("assets by type query failed: %w", err)
	}
	defer rows.Close()

	var data []ChartData
	for rows.Next() {
		var d ChartData
		var v int64
		if err := rows.Scan(&d.Name, &v); err != nil {
			log.Printf("[WARN] scan assets by type: %v", err)
			continue
		}
		d.Value = float64(v)
		data = append(data, d)
	}
	return data, nil
}

// ============================================================
// 4️⃣ Employees by Department
// ============================================================
func getEmployeesByDept(c *gin.Context) ([]ChartData, error) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT COALESCE(d.name,'Tidak Diketahui'),
		       COUNT(e.id)::bigint
		FROM departments d
		LEFT JOIN employees e ON e.department_id=d.id AND e.deleted_at IS NULL
		GROUP BY d.name ORDER BY 2 DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("employees by dept query failed: %w", err)
	}
	defer rows.Close()

	var list []ChartData
	for rows.Next() {
		var d ChartData
		var v int64
		if err := rows.Scan(&d.Name, &v); err != nil {
			log.Printf("[WARN] scan employees by dept: %v", err)
			continue
		}
		d.Value = float64(v)
		list = append(list, d)
	}
	return list, nil
}

// ============================================================
// 5️⃣ Asset Metrics per Department
// ============================================================
func getAssetMetricsByDept(c *gin.Context) ([]Metric, error) {
	rows, err := database.Pool.Query(c.Request.Context(), `
        SELECT COALESCE(d.name,'Tidak Diketahui') AS name,
               ROUND(AVG(COALESCE(a.asset_health_score,0))::numeric,1) AS avg_health,
               ROUND(AVG(COALESCE(a.governance_score,0))::numeric,1)   AS avg_governance,
               ROUND(100.0 * SUM(CASE WHEN a.compliance_flag THEN 1 ELSE 0 END) / NULLIF(COUNT(a.id),0),1) AS compliance_rate,
               COUNT(a.id) AS total_assets
          FROM departments d
          LEFT JOIN assets a ON a.department_id=d.id AND a.deleted_at IS NULL
         GROUP BY d.name
         ORDER BY avg_health DESC;
    `)
	if err != nil {
		return nil, fmt.Errorf("asset metrics query failed: %w", err)
	}
	defer rows.Close()

	var list []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.Name, &m.AvgHealth, &m.AvgGovernance, &m.ComplianceRate, &m.TotalAssets); err != nil {
			log.Printf("[WARN] scan asset metrics: %v", err)
			continue
		}
		list = append(list, m)
	}
	return list, nil
}

// ============================================================
// 6️⃣ Governance Compliance Summary (Compliant / Partially / Non / Pending)
// ============================================================
func getGovernanceCompliance(c *gin.Context) ([]ChartData, error) {
	ctx := c.Request.Context()
	rows, err := database.Pool.Query(ctx, `
		SELECT 
			CASE 
				WHEN total_compliance_index >= 80 THEN 'Compliant'
				WHEN total_compliance_index BETWEEN 50 AND 79 THEN 'Partially Compliant'
				WHEN total_compliance_index BETWEEN 1 AND 49 THEN 'Non-Compliant'
				ELSE 'Pending'
			END AS compliance_category,
			COUNT(*)::bigint AS total_departments
		FROM compliance_summary
		GROUP BY 1
		ORDER BY 1;
	`)
	if err != nil {
		return nil, fmt.Errorf("governance compliance query failed: %w", err)
	}
	defer rows.Close()

	var list []ChartData
	for rows.Next() {
		var d ChartData
		var v int64
		if err := rows.Scan(&d.Name, &v); err != nil {
			log.Printf("[WARN] scan governance compliance: %v", err)
			continue
		}
		d.Value = float64(v)
		list = append(list, d)
	}

	return list, nil
}

// ============================================================
// 7️⃣ SLA Dashboard: GET /dashboard/sla
// ============================================================
func GetSLADashboard(c *gin.Context) {
	var open, breached, resolved int64
	var avgMTTR, avgMTTA *float64
	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT
		  COUNT(*) FILTER (WHERE status!='Closed'),
		  COUNT(*) FILTER (WHERE status!='Closed' AND sla_due_at<NOW()),
		  COUNT(*) FILTER (WHERE status='Closed'),
		  ROUND(AVG(EXTRACT(EPOCH FROM (resolved_at-created_at))/60),2),
		  ROUND(AVG(EXTRACT(EPOCH FROM (response_due_at-created_at))/60),2)
		FROM tickets WHERE deleted_at IS NULL;
	`).Scan(&open, &breached, &resolved, &avgMTTR, &avgMTTA)
	if err != nil {
		log.Printf("[ERROR] GetSLADashboard: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load SLA dashboard"})
		return
	}
	total := open + resolved
	comp := 0.0
	if total > 0 {
		comp = (float64(total-breached) / float64(total)) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"open_tickets":        open,
		"breached_tickets":    breached,
		"resolved_tickets":    resolved,
		"sla_compliance_rate": comp,
		"avg_mttr_minutes":    avgMTTR,
		"avg_mtta_minutes":    avgMTTA,
	})
}

// ============================================================
// 8️⃣ Health–SLA Trend (12 bulan terakhir): GET /dashboard/trend-health-sla
// ============================================================
func GetHealthSLATrend(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		WITH monthly_health AS (
			SELECT 
				TO_CHAR(a.updated_at, 'YYYY-MM') AS period,
				ROUND(AVG(COALESCE(a.asset_health_score, 0))::numeric, 1) AS avg_health
			FROM assets a
			WHERE a.deleted_at IS NULL 
			  AND a.updated_at > NOW() - INTERVAL '12 months'
			GROUP BY 1
		),
		monthly_sla AS (
			SELECT 
				TO_CHAR(t.created_at, 'YYYY-MM') AS period,
				COUNT(*) FILTER (WHERE t.breach_flag = true OR t.compliance_flag = false) AS sla_breach_count
			FROM tickets t
			WHERE t.deleted_at IS NULL 
			  AND t.created_at > NOW() - INTERVAL '12 months'
			GROUP BY 1
		)
		SELECT 
			COALESCE(h.period, s.period) AS period,
			COALESCE(h.avg_health, 0) AS avg_health,
			COALESCE(s.sla_breach_count, 0) AS sla_breach_count
		FROM monthly_health h
		FULL JOIN monthly_sla s ON h.period = s.period
		ORDER BY period ASC;
	`)
	if err != nil {
		log.Printf("[ERROR] GetHealthSLATrend: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trend"})
		return
	}
	defer rows.Close()

	type Trend struct {
		Period         string  `json:"period"`
		AvgHealth      float64 `json:"avg_health"`
		SLABreachCount float64 `json:"sla_breach_count"`
	}
	var list []Trend
	for rows.Next() {
		var t Trend
		if err := rows.Scan(&t.Period, &t.AvgHealth, &t.SLABreachCount); err != nil {
			log.Printf("[WARN] scan trend: %v", err)
			continue
		}
		list = append(list, t)
	}

	c.JSON(http.StatusOK, gin.H{"trend": list})
}

// ============================================================
// 9️⃣ Health–SLA Forecast (Linear Regression): GET /dashboard/forecast-health-sla
// ============================================================
func GetHealthSLAForecast(c *gin.Context) {
	// Reuse same aggregation as trend
	rows, err := database.Pool.Query(c.Request.Context(), `
		WITH monthly_health AS (
			SELECT 
				TO_CHAR(a.updated_at, 'YYYY-MM') AS period,
				ROUND(AVG(COALESCE(a.asset_health_score, 0))::numeric, 1) AS avg_health
			FROM assets a
			WHERE a.deleted_at IS NULL 
			  AND a.updated_at > NOW() - INTERVAL '12 months'
			GROUP BY 1
		),
		monthly_sla AS (
			SELECT 
				TO_CHAR(t.created_at, 'YYYY-MM') AS period,
				COUNT(*) FILTER (WHERE t.status!='Resolved' AND t.sla_due_at<NOW()) AS sla_breach_count
			FROM tickets t
			WHERE t.deleted_at IS NULL 
			  AND t.created_at > NOW() - INTERVAL '12 months'
			GROUP BY 1
		)
		SELECT 
			COALESCE(h.period, s.period) AS period,
			COALESCE(h.avg_health, 0) AS avg_health,
			COALESCE(s.sla_breach_count, 0) AS sla_breach_count
		FROM monthly_health h
		FULL JOIN monthly_sla s ON h.period = s.period
		ORDER BY period ASC;
	`)
	if err != nil {
		log.Printf("[ERROR] GetHealthSLAForecast: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch forecast"})
		return
	}
	defer rows.Close()

	type P struct {
		Period string
		H      float64
		B      float64
	}
	var pts []P
	for rows.Next() {
		var p P
		if err := rows.Scan(&p.Period, &p.H, &p.B); err != nil {
			log.Printf("[WARN] scan forecast point: %v", err)
			continue
		}
		pts = append(pts, p)
	}
	if len(pts) < 2 {
		c.JSON(http.StatusOK, gin.H{"note": "not enough data", "sample_size": len(pts)})
		return
	}

	calc := func(x, y []float64) (a, b float64) {
		n := float64(len(x))
		var sx, sy, sxy, sx2 float64
		for i := range x {
			sx += x[i]
			sy += y[i]
			sxy += x[i] * y[i]
			sx2 += x[i] * x[i]
		}
		den := n*sx2 - sx*sx
		if den == 0 {
			return 0, y[len(y)-1]
		}
		a = (n*sxy - sx*sy) / den
		b = (sy - a*sx) / n
		return
	}

	var xs, hs, bs []float64
	for i := range pts {
		xs = append(xs, float64(i))
		hs = append(hs, pts[i].H)
		bs = append(bs, pts[i].B)
	}
	aH, bH := calc(xs, hs)
	aB, bB := calc(xs, bs)
	next := float64(len(xs))
	pH := aH*next + bH
	pB := aB*next + bB
	if pH < 0 {
		pH = 0
	} else if pH > 100 {
		pH = 100
	}
	if pB < 0 {
		pB = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"predicted_health_next_month":     math.Round(pH*10) / 10,
		"predicted_sla_breach_next_month": math.Round(pB),
		"sample_size":                     len(xs),
	})
}

// ============================================================
// 🔟 Realtime state ping
// ============================================================
func GetRealtimeState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Realtime WebSocket active"})
}

// ============================================================
// GET /dashboard/alert-stats
// ============================================================
func GetAlertStats(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`SELECT severity, COUNT(*) AS count FROM alerts GROUP BY severity ORDER BY severity`,
	)
	if err != nil {
		log.Printf("[ERROR] GetAlertStats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load alert stats"})
		return
	}
	defer rows.Close()

	type Item struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var list []Item
	for rows.Next() {
		var i Item
		if err := rows.Scan(&i.Severity, &i.Count); err != nil {
			log.Printf("[WARN] scan alert stats: %v", err)
			continue
		}
		list = append(list, i)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// GET /dashboard/alert-trends
// ============================================================
func GetAlertTrends(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD') AS day,
			severity,
			COUNT(*) AS count
		FROM alerts
		GROUP BY day, severity
		ORDER BY day ASC
		`,
	)
	if err != nil {
		log.Printf("[ERROR] GetAlertTrends: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query alert trends"})
		return
	}
	defer rows.Close()

	type Trend struct {
		Day      string `json:"day"`
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var list []Trend
	for rows.Next() {
		var t Trend
		if err := rows.Scan(&t.Day, &t.Severity, &t.Count); err != nil {
			log.Printf("[WARN] scan alert trend: %v", err)
			continue
		}
		list = append(list, t)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// GET /dashboard/health-heatmap
// ============================================================
func GetDepartmentHealthHeatmap(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`
		SELECT
			d.name AS department,
			ROUND(AVG(COALESCE(a.asset_health_score, 0))::numeric, 1) AS avg_health,
			COUNT(al.id) AS alert_count
		FROM departments d
		LEFT JOIN assets a ON a.department_id = d.id AND a.deleted_at IS NULL
		LEFT JOIN alerts al ON al.asset_id = a.id
		GROUP BY d.name
		ORDER BY d.name
		`,
	)
	if err != nil {
		log.Printf("[ERROR] GetDepartmentHealthHeatmap: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch heatmap"})
		return
	}
	defer rows.Close()

	type Row struct {
		Department string  `json:"department"`
		AvgHealth  float64 `json:"avg_health"`
		AlertCount int64   `json:"alert_count"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Department, &r.AvgHealth, &r.AlertCount); err != nil {
			log.Printf("[WARN] scan heatmap: %v", err)
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// GET /dashboard/predictive-forecast
// ============================================================
func GetPredictiveForecast(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`
		WITH dept_health AS (
			SELECT
				d.name AS department,
				ROUND(AVG(COALESCE(a.asset_health_score, 0))::numeric, 2) AS avg_health,
				(SELECT COUNT(*) FROM alerts WHERE acknowledged = false) AS alert_count
			FROM departments d
			LEFT JOIN assets a ON a.department_id = d.id AND a.deleted_at IS NULL
			GROUP BY d.name
		)
		SELECT
			department,
			avg_health,
			alert_count,
			GREATEST(0, LEAST(100, avg_health - (alert_count * 0.1))) AS forecast_next_7,
			GREATEST(0, LEAST(100, avg_health - (alert_count * 0.2))) AS forecast_next_30
		FROM dept_health
		ORDER BY department;
		`,
	)
	if err != nil {
		log.Printf("[ERROR] GetPredictiveForecast: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate forecast"})
		return
	}
	defer rows.Close()

	type Row struct {
		Department     string  `json:"department"`
		AvgHealth      float64 `json:"avg_health"`
		AlertCount     int64   `json:"alert_count"`
		ForecastNext7  float64 `json:"forecast_next_7"`
		ForecastNext30 float64 `json:"forecast_next_30"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.Department, &r.AvgHealth, &r.AlertCount, &r.ForecastNext7, &r.ForecastNext30); err != nil {
			log.Printf("[WARN] scan predictive forecast: %v", err)
			continue
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// GET /dashboard/recommendations
// ============================================================
func GetRecommendations(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`
		WITH dept_health AS (
			SELECT
				d.name AS department,
				ROUND(AVG(COALESCE(a.asset_health_score, 0))::numeric, 2) AS avg_health,
				(SELECT COUNT(*) FROM alerts WHERE acknowledged = false) AS alert_count
			FROM departments d
			LEFT JOIN assets a ON a.department_id = d.id AND a.deleted_at IS NULL
			GROUP BY d.name
		)
		SELECT department, avg_health, alert_count FROM dept_health;
		`,
	)
	if err != nil {
		log.Printf("[ERROR] GetRecommendations: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch recommendations"})
		return
	}
	defer rows.Close()

	type Suggestion struct {
		Department string `json:"department"`
		Action     string `json:"action"`
		Reason     string `json:"reason"`
	}

	var suggestions []Suggestion
	for rows.Next() {
		var dept string
		var avgHealth float64
		var alertCount int64
		if err := rows.Scan(&dept, &avgHealth, &alertCount); err != nil {
			log.Printf("[WARN] scan recommendations: %v", err)
			continue
		}

		var action, reason string
		switch {
		case avgHealth < 50 && alertCount > 10:
			action = "Audit aset dan cek kepatuhan operasional."
			reason = "Rata-rata kesehatan rendah dan banyak alert aktif."
		case avgHealth < 70 && alertCount > 5:
			action = "Lakukan preventive maintenance minggu ini."
			reason = "Tren penurunan kesehatan terdeteksi."
		case avgHealth > 85 && alertCount < 3:
			action = "Kondisi stabil — pertahankan perawatan rutin."
			reason = "Aset dalam kondisi baik dan sedikit alert."
		default:
			action = "Pantau tren minggu ini."
			reason = "Tidak ada anomali signifikan."
		}

		suggestions = append(suggestions, Suggestion{
			Department: dept,
			Action:     action,
			Reason:     reason,
		})
	}

	c.JSON(http.StatusOK, gin.H{"recommendations": suggestions})
}

// ============================================================
// GET /dashboard/correlation
// ============================================================
func GetCorrelationMatrix(c *gin.Context) {
	rows, err := database.Pool.Query(
		c.Request.Context(),
		`
		WITH corr AS (
			SELECT 
				d.name AS department,
				COUNT(DISTINCT a.id) AS assets,
				COUNT(DISTINCT t.id) AS tickets,
				COUNT(DISTINCT al.id) AS alerts,
				ROUND(COALESCE(AVG(a.asset_health_score), 100)::numeric, 2) AS avg_health
			FROM departments d
			LEFT JOIN assets a ON a.department_id = d.id
			LEFT JOIN tickets t ON t.related_asset_id = a.id
			LEFT JOIN alerts al ON al.asset_id = a.id
			GROUP BY d.name
		)
		SELECT 
			department,
			COALESCE(assets, 0) AS assets,
			COALESCE(tickets, 0) AS tickets,
			COALESCE(alerts, 0) AS alerts,
			COALESCE(avg_health, 0) AS avg_health,
			COALESCE(ROUND((alerts::numeric / NULLIF(assets, 0)) * 100, 2), 0) AS alert_ratio,
			COALESCE(ROUND((tickets::numeric / NULLIF(assets, 0)) * 100, 2), 0) AS ticket_ratio
		FROM corr
		ORDER BY department;
		`,
	)
	if err != nil {
		log.Printf("[ERROR] GetCorrelationMatrix: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate correlation matrix"})
		return
	}
	defer rows.Close()

	type CorrRow struct {
		Department  string  `json:"department"`
		Assets      int64   `json:"assets"`
		Tickets     int64   `json:"tickets"`
		Alerts      int64   `json:"alerts"`
		AvgHealth   float64 `json:"avg_health"`
		AlertRatio  float64 `json:"alert_ratio"`
		TicketRatio float64 `json:"ticket_ratio"`
	}

	var list []CorrRow
	for rows.Next() {
		var r CorrRow
		if err := rows.Scan(
			&r.Department,
			&r.Assets,
			&r.Tickets,
			&r.Alerts,
			&r.AvgHealth,
			&r.AlertRatio,
			&r.TicketRatio,
		); err != nil {
			log.Printf("[ERROR][GetCorrelationMatrix] scan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// GET /compliance/audit-logs?limit=5
// ============================================================
func GetRecentAuditLogs(c *gin.Context) {
	limit := 5
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT entity_name, action, 
		        COALESCE(e.name,'System') AS actor_name, 
		        created_at
		   FROM audit_logs al
		   LEFT JOIN employees e ON e.id=al.actor_id
		   ORDER BY al.created_at DESC
		   LIMIT $1`, limit)
	if err != nil {
		log.Printf("[ERROR] GetRecentAuditLogs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}
	defer rows.Close()
	type Row struct {
		EntityName string    `json:"entity_name"`
		Action     string    `json:"action"`
		ActorName  string    `json:"actor_name"`
		CreatedAt  time.Time `json:"created_at"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.EntityName, &r.Action, &r.ActorName, &r.CreatedAt); err != nil {
			log.Printf("[WARN] scan audit logs: %v", err)
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

// ============================================================
// GET /api/v1/dashboard/health-status
// ============================================================
func GetHealthStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"data": gin.H{
			"system_uptime":  "99.97%",
			"assets_healthy": 142,
			"assets_at_risk": 8,
			"alerts_active":  3,
		},
	})
}

// ============================================================
// GET /api/v1/compliance/details?category=Compliant
// ============================================================
func GetComplianceDetails(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing category"})
		return
	}

	query := `
		SELECT department_name, total_compliance_index, last_audit_date
		FROM compliance_summary
		WHERE 1=1
	`
	switch category {
	case "Compliant":
		query += " AND total_compliance_index >= 80"
	case "Partially Compliant":
		query += " AND total_compliance_index BETWEEN 50 AND 79"
	case "Non-Compliant":
		query += " AND total_compliance_index BETWEEN 1 AND 49"
	case "Pending":
		query += " AND (total_compliance_index IS NULL OR total_compliance_index = 0)"
	default:
		query += " AND FALSE"
	}

	query += " ORDER BY department_name;"

	rows, err := database.Pool.Query(c.Request.Context(), query)
	if err != nil {
		log.Printf("[ERROR] GetComplianceDetails: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch compliance details"})
		return
	}
	defer rows.Close()

	type Row struct {
		DepartmentName string     `json:"department_name"`
		TotalIndex     *float64   `json:"total_compliance_index"`
		LastAuditDate  *time.Time `json:"last_audit_date"`
	}

	var list []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.DepartmentName, &r.TotalIndex, &r.LastAuditDate); err != nil {
			log.Printf("[WARN] scan compliance details: %v", err)
			continue
		}
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// 🔮 GET /dashboard/predictive-risk
// ============================================================
func GetPredictiveRisk(c *gin.Context) {
	data, err := services.ComputeAssetRiskForecast(c.Request.Context())
	if err != nil {
		log.Printf("[ERROR] GetPredictiveRisk: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute forecast"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"predictions": data})
}

// ============================================================
// 7️⃣ Alert Stats Summary
// ============================================================
func getAlertStatsSummary(c *gin.Context) ([]StatCard, error) {
	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT severity, COUNT(*) FROM alerts 
		  WHERE acknowledged=FALSE GROUP BY severity;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cards []StatCard
	for rows.Next() {
		var s string
		var v int64
		rows.Scan(&s, &v)
		cards = append(cards, StatCard{
			Title: fmt.Sprintf("Alert %s", s),
			Value: v,
		})
	}
	return cards, nil
}
