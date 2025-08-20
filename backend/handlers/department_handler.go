// File: backend/handlers/department_handler.go
package handlers

import (
	"context"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// CreateDepartment membuat departemen baru
func CreateDepartment(c *gin.Context) {
	var newDept models.Department
	if err := c.ShouldBindJSON(&newDept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `INSERT INTO departments (name) VALUES ($1) RETURNING id`
	err := database.Pool.QueryRow(context.Background(), query, newDept.Name).Scan(&newDept.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create department"})
		return
	}
	c.JSON(http.StatusCreated, newDept)
}

// GetAllDepartments mengambil semua data departemen
func GetAllDepartments(c *gin.Context) {
	var departments []models.Department
	rows, err := database.Pool.Query(context.Background(), "SELECT id, name FROM departments ORDER BY name ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var dept models.Department
		if err := rows.Scan(&dept.ID, &dept.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan department data"})
			return
		}
		departments = append(departments, dept)
	}
	c.JSON(http.StatusOK, departments)
}

func UpdateDepartment(c *gin.Context) {
	departmentID := c.Param("id")
	var deptData models.Department

	if err := c.ShouldBindJSON(&deptData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	query := `UPDATE departments SET name = $1 WHERE id = $2`

	commandTag, err := database.Pool.Exec(context.Background(), query, deptData.Name, departmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update department"})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Department updated successfully"})
}

func DeleteDepartment(c *gin.Context) {
	departmentID := c.Param("id")

	// PENTING: Cek apakah ada karyawan yang masih terhubung ke departemen ini
	var count int64
	err := database.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM employees WHERE department_id = $1", departmentID).Scan(&count)
	if err == nil && count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tidak bisa menghapus, departemen masih digunakan oleh karyawan."})
		return
	}

	// Jika tidak ada, lanjutkan penghapusan
	commandTag, err := database.Pool.Exec(context.Background(), "DELETE FROM departments WHERE id = $1", departmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete department"})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Department deleted successfully"})
}
