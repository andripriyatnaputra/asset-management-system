// File: backend/models/budget.go
package models

import "time"

type Budget struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	DepartmentID *int64     `json:"department_id,omitempty"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      time.Time  `json:"end_date"`
	TotalAmount  float64    `json:"total_amount"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"-"`
}

type BudgetTransaction struct {
	ID              int64     `json:"id"`
	BudgetID        int64     `json:"budget_id"`
	AssetID         int64     `json:"asset_id"`
	Amount          float64   `json:"amount"`
	TransactionDate time.Time `json:"transaction_date"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
