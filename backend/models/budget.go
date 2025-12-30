package models

import "time"

// ===========================================================
// Budget (ISO/IEC 19770-10:2025 - ITAM Grade A++)
// ===========================================================
type Budget struct {
	ID           int64      `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	DepartmentID *int64     `json:"department_id,omitempty" db:"department_id"`
	Category     *string    `json:"category,omitempty" db:"category"` // CAPEX / OPEX
	Currency     *string    `json:"currency,omitempty" db:"currency"` // default: IDR
	CostCenter   *string    `json:"cost_center,omitempty" db:"cost_center"`
	StartDate    time.Time  `json:"start_date" db:"start_date"`
	EndDate      time.Time  `json:"end_date" db:"end_date"`
	TotalAmount  float64    `json:"total_amount" db:"total_amount"`
	UsedAmount   *float64   `json:"used_amount,omitempty" db:"used_amount"` // otomatis via trigger
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (tampilan)
	DepartmentName *string  `json:"department_name,omitempty"`
	Status         *string  `json:"status,omitempty"`   // computed: active/expired
	Progress       *float64 `json:"progress,omitempty"` // computed dari Used/Total
}

// ===========================================================
// BudgetTransaction - Realisasi Biaya (multi-entity linkage)
// ===========================================================
type BudgetTransaction struct {
	ID              int64     `json:"id" db:"id"`
	BudgetID        int64     `json:"budget_id" db:"budget_id"`
	EntityType      string    `json:"entity_type" db:"entity_type"` // asset / license / contract
	EntityID        *int64    `json:"entity_id,omitempty" db:"entity_id"`
	Amount          float64   `json:"amount" db:"amount"`
	TransactionType *string   `json:"transaction_type,omitempty" db:"transaction_type"` // expense / income / adjustment
	Currency        *string   `json:"currency,omitempty" db:"currency"`
	ExchangeRate    *float64  `json:"exchange_rate,omitempty" db:"exchange_rate"`
	TaxAmount       *float64  `json:"tax_amount,omitempty" db:"tax_amount"`
	CostCenter      *string   `json:"cost_center,omitempty" db:"cost_center"`
	Category        *string   `json:"category,omitempty" db:"category"` // CAPEX / OPEX
	Notes           *string   `json:"notes,omitempty" db:"notes"`
	CreatedBy       *int64    `json:"created_by,omitempty" db:"created_by"`
	TransactionDate time.Time `json:"transaction_date" db:"transaction_date"`

	// 🔹 Compliance
	ComplianceFlag *bool   `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote *string `json:"compliance_note,omitempty" db:"compliance_note"`

	// 🔹 Metadata (auto trigger)
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (join display)
	BudgetName    *string `json:"budget_name,omitempty"`
	Department    *string `json:"department,omitempty"`
	EntityName    *string `json:"entity_name,omitempty"`
	TransactionBy *string `json:"transaction_by,omitempty"`
}
