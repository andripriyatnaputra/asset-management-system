package models

import "time"

// Contract merepresentasikan perjanjian atau kontrak (pembelian, lisensi, maintenance)
// sesuai ISO/IEC 19770-10:2025 (ITAM Grade A++).
type Contract struct {
	ID             int64      `json:"id" db:"id"`
	ContractNumber string     `json:"contract_number" db:"contract_number"`
	VendorID       *int64     `json:"vendor_id,omitempty" db:"vendor_id"` // relasi ke master vendor (opsional)
	Vendor         *string    `json:"vendor,omitempty" db:"vendor"`
	ContractType   *string    `json:"contract_type,omitempty" db:"contract_type"` // Purchase / Subscription / Maintenance
	StartDate      *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate        *time.Time `json:"end_date,omitempty" db:"end_date"`
	TotalValue     *float64   `json:"total_value,omitempty" db:"total_value"`
	Currency       *string    `json:"currency,omitempty" db:"currency"` // default: IDR
	PaymentTerms   *string    `json:"payment_terms,omitempty" db:"payment_terms"`
	ContactPerson  *string    `json:"contact_person,omitempty" db:"contact_person"`
	ContactEmail   *string    `json:"contact_email,omitempty" db:"contact_email"`
	AttachmentURL  *string    `json:"attachment_url,omitempty" db:"attachment_url"`
	Notes          *string    `json:"notes,omitempty" db:"notes"`
	Status         *string    `json:"status,omitempty" db:"status"`       // active | expired | terminated
	BudgetID       *int64     `json:"budget_id,omitempty" db:"budget_id"` // referensi ke budgets
	CostCenter     *string    `json:"cost_center,omitempty" db:"cost_center"`

	// 🔹 Compliance fields (Grade A++)
	ComplianceFlag *bool   `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote *string `json:"compliance_note,omitempty" db:"compliance_note"`

	// 🔹 Audit metadata
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (tampilan / join)
	VendorName    *string `json:"vendor_name,omitempty"`
	BudgetName    *string `json:"budget_name,omitempty"`
	Department    *string `json:"department,omitempty"`
	RemainingDays *int64  `json:"remaining_days,omitempty"` // hasil kalkulasi dari EndDate
}
