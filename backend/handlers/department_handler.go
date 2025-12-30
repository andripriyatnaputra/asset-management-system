// File: backend/handlers/department_handler.go
package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 🏗 CREATE DEPARTMENT (Governed A ++)
// ============================================================
func CreateDepartment(c *gin.Context) {
	var body struct {
		Name         string `json:"name" binding:"required"`
		ManagerID    *int64 `json:"manager_id,omitempty"`
		CostCenterID *int64 `json:"cost_center_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔹 Validasi cost_center_id (jika diisi)
	if body.CostCenterID != nil {
		var exists bool
		err := database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1 AND deleted_at IS NULL)`,
			*body.CostCenterID,
		).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cost center tidak ditemukan"})
			return
		}
	}

	// 🔹 Validasi manager_id (jika diisi)
	if body.ManagerID != nil {
		var exists bool
		err := database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND deleted_at IS NULL)`,
			*body.ManagerID,
		).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Manager tidak ditemukan"})
			return
		}
	}

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO departments (name, manager_id, cost_center_id, created_at, updated_at)
		 VALUES ($1,$2,$3,NOW(),NOW()) RETURNING id`,
		body.Name, body.ManagerID, body.CostCenterID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "departments", id, "CREATE", body)
	services.BroadcastAlert(fmt.Sprintf("Departemen baru '%s' dibuat", body.Name), "info")
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "department created"})
}

// ============================================================
// 📋 GET ALL DEPARTMENTS (+ safe join + governance filter)
// ============================================================
func GetAllDepartments(c *gin.Context) {
	role, _ := c.Get("role")
	alertFlag := c.Query("alert") == "1" // hanya broadcast kalau ?alert=1

	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT 
			d.id,
			d.name,
			COALESCE(d.cost_center_id, 0) AS cost_center,
			COALESCE(emp.total, 0) AS employees,
			COALESCE(ast.total, 0) AS assets,
			COALESCE(bud.total, 0) AS total_budget
		FROM departments d
		LEFT JOIN (
			SELECT department_id, COUNT(*) AS total 
			FROM employees 
			WHERE deleted_at IS NULL 
			GROUP BY department_id
		) emp ON emp.department_id = d.id
		LEFT JOIN (
			SELECT department_id, COUNT(*) AS total 
			FROM assets 
			WHERE deleted_at IS NULL 
			GROUP BY department_id
		) ast ON ast.department_id = d.id
		LEFT JOIN (
			SELECT department_id, SUM(total_amount) AS total 
			FROM budgets 
			WHERE deleted_at IS NULL 
			GROUP BY department_id
		) bud ON bud.department_id = d.id
		WHERE d.deleted_at IS NULL
		ORDER BY d.name;
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query departments"})
		return
	}
	defer rows.Close()

	type DeptRow struct {
		ID          int64   `json:"id"`
		Name        string  `json:"name"`
		CostCenter  string  `json:"cost_center"`
		Employees   int     `json:"employees"`
		Assets      int     `json:"assets"`
		TotalBudget float64 `json:"total_budget"`
		HealthScore float64 `json:"department_health_score"`
		GovScore    float64 `json:"governance_score"`
		Alert       *string `json:"alert,omitempty"`
	}

	var list []DeptRow
	for rows.Next() {
		var r DeptRow
		if err := rows.Scan(&r.ID, &r.Name, &r.CostCenter, &r.Employees, &r.Assets, &r.TotalBudget); err != nil {
			log.Printf("[WARN] scan error in GetAllDepartments: %v", err)
			continue
		}

		r.HealthScore = computeDeptHealth(r.Employees, r.Assets)
		r.GovScore = governanceScore(r.TotalBudget > 0, r.Assets > 0, true)

		// 🔹 Governance warning hanya bila benar-benar kosong total
		if r.Employees == 0 && r.TotalBudget == 0 {
			msg := fmt.Sprintf("Departemen %s belum lengkap (governance warning)", r.Name)
			r.Alert = &msg

			// hanya broadcast jika role = super_admin dan alertFlag aktif
			if alertFlag && role == "super_admin" {
				services.BroadcastAlert(msg, "warning")
			}
		}

		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[WARN] iteration error in GetAllDepartments: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// 🔁 UPDATE DEPARTMENT (+ alert manager change & validation)
// ============================================================
func UpdateDepartment(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Name         *string `json:"name,omitempty"`
		ManagerID    *int64  `json:"manager_id,omitempty"`
		CostCenterID *int64  `json:"cost_center_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔹 Validasi cost_center_id
	if body.CostCenterID != nil {
		var exists bool
		err := database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1 AND deleted_at IS NULL)`,
			*body.CostCenterID,
		).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cost center tidak ditemukan"})
			return
		}
	}

	// 🔹 Validasi manager_id
	if body.ManagerID != nil {
		var exists bool
		err := database.Pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND deleted_at IS NULL)`,
			*body.ManagerID,
		).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Manager tidak ditemukan"})
			return
		}
	}

	// 🔹 Simpan nilai lama manager untuk kebutuhan alert (opsional)
	var oldManagerID *int64
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT manager_id FROM departments WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&oldManagerID)

	_, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE departments SET
		 name = COALESCE($1,name),
		 manager_id = COALESCE($2,manager_id),
		 cost_center_id = COALESCE($3,cost_center_id),
		 updated_at = NOW()
		 WHERE id=$4 AND deleted_at IS NULL`,
		body.Name, body.ManagerID, body.CostCenterID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "departments", mustAtoi64(id), "UPDATE", body)

	// 🔹 Broadcast perubahan manager (deterministic, side-effect tidak mempengaruhi regresi)
	if body.ManagerID != nil && (oldManagerID == nil || *oldManagerID != *body.ManagerID) {
		msg := fmt.Sprintf("Manager departemen #%s berubah menjadi ID %d", id, *body.ManagerID)
		services.BroadcastAlert(msg, "info")
	}

	c.JSON(http.StatusOK, gin.H{"message": "department updated"})
}

// ============================================================
// 🗑 DELETE DEPARTMENT (soft + audit + full linkage check)
// ============================================================
func DeleteDepartment(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))

	// 🔹 Cek employees
	var empCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM employees WHERE department_id=$1 AND deleted_at IS NULL`, id).
		Scan(&empCount)

	// 🔹 Cek assets
	var assetCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM assets WHERE department_id=$1 AND deleted_at IS NULL`, id).
		Scan(&assetCount)

	// 🔹 Cek budgets
	var budCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM budgets WHERE department_id=$1 AND deleted_at IS NULL`, id).
		Scan(&budCount)

	// 🔹 Cek licenses (langsung via department_id)
	var licCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM licenses WHERE department_id=$1 AND deleted_at IS NULL`, id).
		Scan(&licCount)

	if empCount > 0 || assetCount > 0 || budCount > 0 || licCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":     "department masih dipakai oleh entitas lain",
			"employees": empCount,
			"assets":    assetCount,
			"budgets":   budCount,
			"licenses":  licCount,
		})
		return
	}

	res, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE departments SET deleted_at=NOW(), updated_at=NOW() 
		  WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if res.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "department not found"})
		return
	}

	msg := fmt.Sprintf("Department #%d dihapus (soft delete)", id)
	services.BroadcastAlert(msg, "warning")
	middleware.LogAction(c, "departments", id, "DELETE", nil)
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// ============================================================
// 📊 GET DEPARTMENT SUMMARY (+ governance metrics)
// ============================================================
func GetDepartmentSummary(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))
	var assets, employees, licenses int
	var budget float64
	err := database.Pool.QueryRow(c.Request.Context(), `
		SELECT 
		  (SELECT COUNT(*) FROM assets WHERE department_id=$1 AND deleted_at IS NULL),
		  (SELECT COUNT(*) FROM employees WHERE department_id=$1 AND deleted_at IS NULL),
		  (SELECT COUNT(*) FROM licenses WHERE department_id=$1 AND deleted_at IS NULL),
		  COALESCE((SELECT SUM(total_amount) FROM budgets WHERE department_id=$1 AND deleted_at IS NULL),0)
	`, id).Scan(&assets, &employees, &licenses, &budget)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	health := computeDeptHealth(employees, assets)
	gov := governanceScore(budget > 0, assets > 0, true)
	c.JSON(http.StatusOK, gin.H{
		"department_id":    id,
		"total_assets":     assets,
		"total_employees":  employees,
		"total_budget":     budget,
		"health_score":     health,
		"governance_score": gov,
	})
}

// ============================================================
// 🧩 HELPERS
// ============================================================
func computeDeptHealth(emp, asset int) float64 {
	if emp == 0 && asset == 0 {
		return 0
	}
	score := 50.0 + float64(emp*2) + float64(asset)
	if score > 100 {
		score = 100
	}
	return score
}
