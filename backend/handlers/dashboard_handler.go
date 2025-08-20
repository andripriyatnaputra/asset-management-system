// File: backend/handlers/dashboard_handler.go
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

type StatCard struct {
	Title string `json:"title"`
	Value int64  `json:"value"`
}

type ChartData struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type RecentActivity struct {
	AssetName    string    `json:"asset_name"`
	EmployeeName string    `json:"employee_name"`
	AssignedAt   time.Time `json:"assigned_at"`
	Notes        string    `json:"notes"`
}

type DashboardStats struct {
	StatCards       []StatCard       `json:"stat_cards"`
	RecentActivity  []RecentActivity `json:"recent_activity"`
	AssetsByType    []ChartData      `json:"assets_by_type"`
	EmployeesByDept []ChartData      `json:"employees_by_dept"`
}

func GetDashboardStats(c *gin.Context) {
	var stats DashboardStats
	var err error

	// 1. Ambil data untuk Stat Cards
	stats.StatCards, err = getStatCards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stat cards"})
		return
	}

	// 2. Ambil data untuk Aktivitas Terakhir
	stats.RecentActivity, err = getRecentActivities()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent activity"})
		return
	}

	// 3. AMBIL DATA BARU UNTUK CHARTS
	stats.AssetsByType, err = getAssetsByType()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get assets by type"})
		return
	}
	stats.EmployeesByDept, err = getEmployeesByDept()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get employees by department"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func getStatCards() ([]StatCard, error) {
	cards := make([]StatCard, 4)
	cards[0].Title = "Total Aset"
	cards[1].Title = "Aset Dipinjam"
	cards[2].Title = "Aset Tersedia"
	cards[3].Title = "Total Karyawan"

	query := `
		SELECT
			(SELECT COUNT(*) FROM assets WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM assets WHERE status = 'Assigned' AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM assets WHERE status = 'In Stock' AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM employees WHERE deleted_at IS NULL)
	`
	err := database.Pool.QueryRow(context.Background(), query).Scan(
		&cards[0].Value, &cards[1].Value, &cards[2].Value, &cards[3].Value,
	)

	return cards, err
}

func getRecentActivities() ([]RecentActivity, error) {
	activities := []RecentActivity{}
	query := `
		SELECT a.name, e.name, aa.assigned_at, aa.notes
		FROM asset_assignments aa
		JOIN assets a ON aa.asset_id = a.id
		JOIN employees e ON aa.employee_id = e.id
		ORDER BY aa.assigned_at DESC
		LIMIT 5`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var act RecentActivity
		if err := rows.Scan(&act.AssetName, &act.EmployeeName, &act.AssignedAt, &act.Notes); err != nil {
			return nil, err
		}
		activities = append(activities, act)
	}
	return activities, nil
}

func getAssetsByType() ([]ChartData, error) {
	data := []ChartData{}
	query := `
		SELECT at.name, COUNT(a.id) as value
		FROM assets a
		JOIN asset_types at ON a.asset_type_id = at.id
		WHERE a.deleted_at IS NULL
		GROUP BY at.name
		ORDER BY value DESC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cd ChartData
		if err := rows.Scan(&cd.Name, &cd.Value); err != nil {
			return nil, err
		}
		data = append(data, cd)
	}
	return data, nil
}

func getEmployeesByDept() ([]ChartData, error) {
	data := []ChartData{}
	query := `
		SELECT d.name, COUNT(e.id) as value
		FROM employees e
		JOIN departments d ON e.department_id = d.id
		WHERE e.deleted_at IS NULL
		GROUP BY d.name
		ORDER BY value DESC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var cd ChartData
		if err := rows.Scan(&cd.Name, &cd.Value); err != nil {
			return nil, err
		}
		data = append(data, cd)
	}
	return data, nil
}
