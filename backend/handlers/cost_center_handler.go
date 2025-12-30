package handlers

import (
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 📦 MODEL
// ============================================================
type CostCenter struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// ============================================================
// 📋 GET ALL COST CENTERS (only active)
// ============================================================
func GetCostCenters(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(),
		`SELECT id, code, name, created_at, updated_at
		   FROM cost_centers 
		  WHERE deleted_at IS NULL
		  ORDER BY code`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load cost centers"})
		return
	}
	defer rows.Close()

	var list []CostCenter
	for rows.Next() {
		var cc CostCenter
		rows.Scan(&cc.ID, &cc.Code, &cc.Name, &cc.CreatedAt, &cc.UpdatedAt)
		list = append(list, cc)
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ============================================================
// 🆕 CREATE COST CENTER
// ============================================================
func CreateCostCenter(c *gin.Context) {
	var body CostCenter
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 🔹 validate unique (code OR name)
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM cost_centers 
		                WHERE (code=$1 OR name=$2) AND deleted_at IS NULL)`,
		body.Code, body.Name).Scan(&exists)
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "code or name already exists"})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c.Request.Context(),
		`INSERT INTO cost_centers (code, name, created_at, updated_at) 
		 VALUES ($1,$2,NOW(),NOW()) RETURNING id`,
		body.Code, body.Name).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "cost_centers", id, "CREATE", body)
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "cost center created"})
}

// ============================================================
// ✏️ UPDATE COST CENTER
// ============================================================
func UpdateCostCenter(c *gin.Context) {
	id := c.Param("id")
	var body CostCenter

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// 🔹 validate existence
	var exists bool
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM cost_centers WHERE id=$1 AND deleted_at IS NULL)`, id,
	).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "cost center not found"})
		return
	}

	_, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE cost_centers 
		    SET code=$1, name=$2, updated_at=NOW()
		  WHERE id=$3 AND deleted_at IS NULL`,
		body.Code, body.Name, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	middleware.LogAction(c, "cost_centers", mustAtoi64(id), "UPDATE", body)
	c.JSON(http.StatusOK, gin.H{"message": "cost center updated"})
}

// ============================================================
// 🗑 SOFT DELETE WITH FULL LINKAGE CHECKING
// ============================================================
func DeleteCostCenter(c *gin.Context) {
	id := mustAtoi64(c.Param("id"))

	// 🔹 1. cek linkage ke departments
	var depCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM departments 
		  WHERE cost_center_id=$1 AND deleted_at IS NULL`, id).Scan(&depCount)

	// 🔹 2. cek linkage ke budgets
	var budCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM budgets 
		  WHERE cost_center_id=$1 AND deleted_at IS NULL`, id).Scan(&budCount)

	// 🔹 3. cek linkage ke contracts (via contract.cost_center_id)
	var contractCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM contracts 
		  WHERE cost_center_id=$1 AND deleted_at IS NULL`, id).Scan(&contractCount)

	// 🔹 4. cek linkage ke licenses (melalui budgets)
	// licenses → budget_id → budget.cost_center_id
	var licenseCount int
	_ = database.Pool.QueryRow(c.Request.Context(),
		`SELECT COUNT(*) 
		   FROM licenses l
		   JOIN budgets b ON b.id = l.budget_id
		  WHERE b.cost_center_id=$1 AND l.deleted_at IS NULL`, id).Scan(&licenseCount)

	if depCount > 0 || budCount > 0 || contractCount > 0 || licenseCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "cannot delete cost center; still referenced",
			"departments": depCount,
			"budgets":     budCount,
			"contracts":   contractCount,
			"licenses":    licenseCount,
		})
		return
	}

	// 🔹 soft delete
	_, err := database.Pool.Exec(c.Request.Context(),
		`UPDATE cost_centers 
		    SET deleted_at=NOW(), updated_at=NOW()
		  WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}

	middleware.LogAction(c, "cost_centers", id, "DELETE", gin.H{"deleted": true})
	c.JSON(http.StatusOK, gin.H{"message": "cost center deleted (soft)"})
}
