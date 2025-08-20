// File: backend/handlers/budget_handler.go
package handlers

import (
	"context"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// CreateBudget adds a new budget
func CreateBudget(c *gin.Context) {
	var budget models.Budget
	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `INSERT INTO budgets (name, department_id, start_date, end_date, total_amount)
			  VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	err := database.Pool.QueryRow(context.Background(), query,
		budget.Name, budget.DepartmentID, budget.StartDate, budget.EndDate, budget.TotalAmount,
	).Scan(&budget.ID, &budget.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget", "detail": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, budget)
}

// GetAllBudgets retrieves all budgets
type BudgetInfo struct {
	models.Budget
	SpentAmount float64 `json:"spent_amount"`
}

// Replace the old GetAllBudgets function
func GetAllBudgets(c *gin.Context) {
	var budgets []BudgetInfo
	query := `
		SELECT 
			b.id, b.name, b.department_id, b.start_date, b.end_date, b.total_amount, b.created_at,
			COALESCE(SUM(bt.amount), 0) as spent_amount
		FROM budgets b
		LEFT JOIN budget_transactions bt ON b.id = bt.budget_id
		WHERE b.deleted_at IS NULL
		GROUP BY b.id
		ORDER BY b.start_date DESC`

	rows, err := database.Pool.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch budgets"})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var budget BudgetInfo
		if err := rows.Scan(&budget.ID, &budget.Name, &budget.DepartmentID, &budget.StartDate, &budget.EndDate, &budget.TotalAmount, &budget.CreatedAt, &budget.SpentAmount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan budget data"})
			return
		}
		budgets = append(budgets, budget)
	}
	c.JSON(http.StatusOK, budgets)
}

func UpdateBudget(c *gin.Context) {
	budgetID := c.Param("id")
	var budget models.Budget

	if err := c.ShouldBindJSON(&budget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	query := `
		UPDATE budgets 
		SET 
			name = $1, 
			department_id = $2, 
			start_date = $3, 
			end_date = $4, 
			total_amount = $5
		WHERE id = $6 AND deleted_at IS NULL`

	commandTag, err := database.Pool.Exec(context.Background(), query,
		budget.Name, budget.DepartmentID, budget.StartDate, budget.EndDate, budget.TotalAmount, budgetID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update budget", "detail": err.Error()})
		return
	}

	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Budget updated successfully"})
}
