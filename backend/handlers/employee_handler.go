package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type CreateEmployeeRequest struct {
	EmployeeNIK  string `json:"employee_nik" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	DepartmentID *int64 `json:"department_id"` // Diubah menjadi ID
	Password     string `json:"password" binding:"required,min=8"`
	Role         string `json:"role" binding:"required"`
}

func CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	query := `INSERT INTO employees (employee_nik, name, email, department_id, password_hash, role) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	var newEmployeeID int64

	err = database.Pool.QueryRow(context.Background(), query, req.EmployeeNIK, req.Name, req.Email, req.DepartmentID, string(hashedPassword), req.Role).Scan(&newEmployeeID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee", "detail": err.Error()})
		return
	}

	// Kembalikan respons tanpa password
	c.JSON(http.StatusCreated, gin.H{
		"id":           newEmployeeID,
		"employee_nik": req.EmployeeNIK,
		"name":         req.Name,
		"email":        req.Email,
		"department":   req.DepartmentID,
	})
}

func GetAllEmployees(c *gin.Context) {
	// Paginasi
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Filter dan Pencarian
	searchQuery := c.Query("q")
	departmentID := c.Query("department_id")

	// Sorting
	sortBy := c.DefaultQuery("sort_by", "name")
	sortOrder := c.DefaultQuery("sort_order", "asc")
	allowedSortBy := map[string]string{
		"name":            "e.name",
		"employee_nik":    "e.employee_nik",
		"email":           "e.email",
		"department_name": "d.name",
	}
	dbSortBy, ok := allowedSortBy[sortBy]
	if !ok {
		dbSortBy = "e.name" // Default jika input tidak valid
	}

	// Membangun Query SQL secara Dinamis
	baseQuery := `FROM employees e LEFT JOIN departments d ON e.department_id = d.id`
	whereClause := " WHERE e.deleted_at IS NULL"
	params := []interface{}{}
	paramCount := 1

	if searchQuery != "" {
		whereClause += fmt.Sprintf(" AND e.name ILIKE $%d", paramCount)
		params = append(params, "%"+searchQuery+"%")
		paramCount++
	}
	if departmentID != "" {
		whereClause += fmt.Sprintf(" AND e.department_id = $%d", paramCount)
		params = append(params, departmentID)
		paramCount++
	}

	// Query untuk menghitung total record
	countQuery := "SELECT COUNT(e.id) " + baseQuery + whereClause
	var totalRecords int64
	err := database.Pool.QueryRow(context.Background(), countQuery, params...).Scan(&totalRecords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count employees"})
		return
	}

	// Query untuk mengambil data
	dataQuery := fmt.Sprintf(`
		SELECT e.id, e.employee_nik, e.name, e.email, e.department_id, d.name as department_name, e.role 
		%s %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		baseQuery, whereClause, dbSortBy, sortOrder, paramCount, paramCount+1)

	params = append(params, limit, offset)

	rows, err := database.Pool.Query(context.Background(), dataQuery, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees", "detail": err.Error()})
		return
	}
	defer rows.Close()

	employees := []models.Employee{}
	for rows.Next() {
		var emp models.Employee
		if err := rows.Scan(&emp.ID, &emp.EmployeeNIK, &emp.Name, &emp.Email, &emp.DepartmentID, &emp.DepartmentName, &emp.Role); err != nil {
			log.Printf("Error scanning employee row: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan employee data"})
			return
		}
		employees = append(employees, emp)
	}

	// Struktur Respons Baru
	c.JSON(http.StatusOK, gin.H{
		"data": employees,
		"pagination": gin.H{
			"total_records": totalRecords,
			"current_page":  page,
			"page_size":     limit,
			"total_pages":   (totalRecords + int64(limit) - 1) / int64(limit),
		},
	})
}

type UpdateEmployeeRequest struct {
	Name         string `json:"name"`
	EmployeeNIK  string `json:"employee_nik"`
	Email        string `json:"email"`
	DepartmentID *int64 `json:"department_id"`
	Role         string `json:"role"`
}

// UpdateEmployee menangani logika untuk mengubah data seorang karyawan
func UpdateEmployee(c *gin.Context) {
	employeeID := c.Param("id")
	var req UpdateEmployeeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	log.Printf("Updating employee ID %s with data: %+v", employeeID, req)

	if req.Role != "" && req.Role != "super_admin" && req.Role != "employee" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role specified"})
		return
	}

	query := `
		UPDATE employees 
		SET 
			name = $1, 
			employee_nik = $2, 
			email = $3, 
			department_id = $4, 
			role = $5 
		WHERE id = $6`

	var departmentID interface{}
	if req.DepartmentID != nil {
		departmentID = *req.DepartmentID
	} else {
		departmentID = nil
	}

	_, err := database.Pool.Exec(context.Background(), query,
		req.Name, req.EmployeeNIK, req.Email, departmentID, req.Role, employeeID)

	if err != nil {
		// Error handling canggih: Cek apakah error disebabkan oleh duplikat NIK/email
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" { // 23505 = unique_violation
			c.JSON(http.StatusConflict, gin.H{"error": "NIK atau Email sudah digunakan oleh karyawan lain."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully"})
}

func DeleteEmployee(c *gin.Context) {
	// 1. Ambil ID karyawan dari parameter URL, contoh: /api/v1/employees/5
	employeeID := c.Param("id")

	// 2. Siapkan query SQL untuk soft delete.
	// Kita hanya mengisi kolom 'deleted_at' dengan waktu saat ini.
	// Kondisi 'deleted_at IS NULL' memastikan kita tidak menghapus user yang sudah dihapus.
	query := `UPDATE employees SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	// 3. Eksekusi query ke database
	commandTag, err := database.Pool.Exec(context.Background(), query, employeeID)
	if err != nil {
		// Jika ada error database, kirim respons 500 Internal Server Error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	// 4. Periksa apakah ada baris yang terpengaruh.
	// Jika 0, berarti karyawan dengan ID tersebut tidak ditemukan atau sudah dihapus sebelumnya.
	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found or already deleted"})
		return
	}

	// 5. Jika berhasil, kirim pesan sukses
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
}

func GetMyAssignedAssets(c *gin.Context) {
	userID, _ := c.Get("userID")

	// Perbarui struct untuk menyertakan tipe aset
	type AssignedAsset struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Tag           string `json:"asset_tag"`
		AssetTypeName string `json:"asset_type_name"` // <-- Field baru
	}
	var assets []AssignedAsset

	// Perbarui query untuk JOIN dengan tabel asset_types
	query := `
		SELECT a.id, a.name, a.asset_tag, at.name as asset_type_name
		FROM assets a
		JOIN asset_assignments aa ON a.id = aa.asset_id
		JOIN asset_types at ON a.asset_type_id = at.id
		WHERE aa.employee_id = $1 AND aa.returned_at IS NULL AND a.deleted_at IS NULL
		ORDER BY a.name`

	rows, err := database.Pool.Query(context.Background(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assigned assets"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var asset AssignedAsset
		// Perbarui Scan untuk menyertakan kolom baru
		if err := rows.Scan(&asset.ID, &asset.Name, &asset.Tag, &asset.AssetTypeName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan asset data"})
			return
		}
		assets = append(assets, asset)
	}

	c.JSON(http.StatusOK, assets)
}
