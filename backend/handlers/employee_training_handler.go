// File: backend/handlers/employee_training_handler.go
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 📘 GetEmployeeTrainings (final fix, support /me)
// ============================================================

func GetEmployeeTrainings(c *gin.Context) {
	idStr := c.Param("id")
	var empID int64

	fmt.Println("[DEBUG di GetEmployeeTrainings] param id =", idStr)

	uid, exists := c.Get("user_id")
	fmt.Println("[DEBUG di GetEmployeeTrainings] c.Get(user_id) =", uid, "exists =", exists)

	// 🔹 Jika path "/employees/me/trainings"
	if idStr == "me" {
		uid, ok := c.Get("user_id")
		if !ok || uid == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		switch v := uid.(type) {
		case int:
			empID = int64(v)
		case int64:
			empID = v
		case float64:
			empID = int64(v)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
			return
		}
	} else {
		// 🔹 Jika ID numerik
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
			return
		}
		empID = id
	}

	rows, err := database.Pool.Query(c, `
		SELECT id, employee_id, training_name, certificate_url, completed_at, created_at
		  FROM employee_trainings
		 WHERE employee_id = $1
		 ORDER BY completed_at DESC NULLS LAST, created_at DESC
	`, empID)
	if err != nil {
		log.Printf("[ERROR][GetEmployeeTrainings] query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trainings"})
		return
	}
	defer rows.Close()

	type Training struct {
		ID             int64      `json:"id"`
		EmployeeID     int64      `json:"employee_id"`
		TrainingName   string     `json:"training_name"`
		CertificateURL *string    `json:"certificate_url"`
		CompletedAt    *time.Time `json:"completed_at"`
		CreatedAt      *time.Time `json:"created_at"`
	}

	var list []Training
	for rows.Next() {
		var t Training
		if err := rows.Scan(&t.ID, &t.EmployeeID, &t.TrainingName, &t.CertificateURL, &t.CompletedAt, &t.CreatedAt); err != nil {
			log.Printf("[ERROR][GetEmployeeTrainings] scan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan training"})
			return
		}
		list = append(list, t)
	}

	c.JSON(http.StatusOK, list)
}

func GetMyTrainings(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok || uid == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var empID int64
	switch v := uid.(type) {
	case int:
		empID = int64(v)
	case int64:
		empID = v
	case float64:
		empID = int64(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
		return
	}

	// Query sama persis
	rows, err := database.Pool.Query(c, `
        SELECT id, employee_id, training_name, certificate_url, completed_at, created_at
          FROM employee_trainings
         WHERE employee_id = $1
         ORDER BY completed_at DESC NULLS LAST, created_at DESC
    `, empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trainings"})
		return
	}
	defer rows.Close()

	type Training struct {
		ID             int64      `json:"id"`
		EmployeeID     int64      `json:"employee_id"`
		TrainingName   string     `json:"training_name"`
		CertificateURL *string    `json:"certificate_url"`
		CompletedAt    *time.Time `json:"completed_at"`
		CreatedAt      *time.Time `json:"created_at"`
	}

	var list []Training
	for rows.Next() {
		var t Training
		if err := rows.Scan(&t.ID, &t.EmployeeID, &t.TrainingName, &t.CertificateURL, &t.CompletedAt, &t.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan training"})
			return
		}
		list = append(list, t)
	}

	c.JSON(http.StatusOK, list)
}

// POST /employees/:id/trainings
func AddEmployeeTraining(c *gin.Context) {
	idStr := c.Param("id")
	empID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
		return
	}

	var input struct {
		TrainingName   string  `json:"training_name" binding:"required"`
		CertificateURL *string `json:"certificate_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "detail": err.Error()})
		return
	}

	// ✅ pastikan employee ada
	var exists bool
	_ = database.Pool.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND deleted_at IS NULL)`, empID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
		return
	}

	// default: training baru tanpa tanggal completed
	var id int64
	err = database.Pool.QueryRow(c, `
		INSERT INTO employee_trainings (employee_id, training_name, certificate_url, completed_at)
		VALUES ($1, $2, $3, NULL)
		RETURNING id`,
		empID, input.TrainingName, input.CertificateURL,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to insert training"})
		return
	}

	middleware.LogAction(c, "employee_trainings", id, "CREATE", input)
	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"message": "training added successfully",
	})
}
