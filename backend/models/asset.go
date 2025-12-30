package models

import "time"

// Asset merepresentasikan entitas aset sesuai ISO/IEC 19770-10:2025 (ITAM Grade A++).
// Versi ini mempertahankan linkage lama sekaligus menambah field baru dari schema terbaru.
type Asset struct {
	ID                 int64      `json:"id" db:"id"`
	Name               string     `json:"name" db:"name"`
	AssetTag           string     `json:"asset_tag" db:"asset_tag"` // unique
	Status             string     `json:"status" db:"status"`       // in_stock, assigned, maintenance, disposed
	AssetTypeID        *int64     `json:"asset_type_id,omitempty" db:"asset_type_id"`
	DepartmentID       *int64     `json:"department_id,omitempty" db:"department_id"`
	CostCenterID       *int64     `json:"cost_center_id,omitempty" db:"cost_center_id"`
	LocationID         *int64     `json:"location_id,omitempty" db:"location_id"`
	PurchaseDate       *time.Time `json:"purchase_date,omitempty" db:"purchase_date"`
	PurchaseCost       *float64   `json:"purchase_cost,omitempty" db:"purchase_cost"`
	InitialPrice       *float64   `json:"initial_price,omitempty" db:"initial_price"`
	Vendor             *string    `json:"vendor,omitempty" db:"vendor"`
	WarrantyExpiry     *time.Time `json:"warranty_expiry,omitempty" db:"warranty_expiry"`
	UsefulLifeMonths   *int64     `json:"useful_life_months,omitempty" db:"useful_life_months"`
	DepreciationMethod *string    `json:"depreciation_method,omitempty" db:"depreciation_method"`
	SalvageValue       *float64   `json:"salvage_value,omitempty" db:"salvage_value"`
	CurrentValue       *float64   `json:"current_value,omitempty" db:"current_value"`
	Currency           *string    `json:"currency,omitempty" db:"currency"` // default: IDR
	LifecycleStage     *string    `json:"lifecycle_stage,omitempty" db:"lifecycle_stage"`
	AssetHealthScore   *float64   `json:"asset_health_score,omitempty" db:"asset_health_score"`
	Condition          *string    `json:"asset_condition,omitempty" db:"condition"`
	ComplianceFlag     *bool      `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote     *string    `json:"compliance_note,omitempty" db:"compliance_note"`

	// 🔹 Field lama yang masih dipakai di handler / governance (tidak semua tersimpan di tabel assets)
	SerialNumber       *string    `json:"serial_number,omitempty"`        // legacy linkage
	AssetCriticality   *string    `json:"asset_criticality,omitempty"`    // legacy linkage
	AcquisitionType    *string    `json:"acquisition_type,omitempty"`     // legacy linkage
	OwnershipType      *string    `json:"ownership_type,omitempty"`       // legacy linkage
	DisposedApprovedBy *int64     `json:"disposed_approved_by,omitempty"` // legacy linkage
	DisposalDate       *time.Time `json:"disposal_date,omitempty"`        // legacy linkage
	Disposed           bool       `json:"disposed"`                       // legacy linkage
	Notes              *string    `json:"notes,omitempty"`                // legacy linkage
	ContractID         *int64     `json:"contract_id,omitempty"`          // linkage
	LicenseID          *int64     `json:"license_id,omitempty"`           // linkage
	BudgetID           *int64     `json:"budget_id,omitempty"`            // linkage

	// 🔹 Linkage tampilan (transient, tidak tersimpan langsung di tabel)
	AssetTypeName        *string `json:"asset_type_name,omitempty"`
	DepartmentName       *string `json:"department_name,omitempty"`
	CostCenter           *string `json:"cost_center,omitempty"`
	CurrentLocationText  *string `json:"current_location_text,omitempty"`
	AssignedToEmployeeID *int64  `json:"assigned_to_employee_id,omitempty"`
	EmployeeName         *string `json:"employee_name,omitempty"`
	LastAuditStatus      *string `json:"last_audit_status,omitempty"`

	// 🔹 Audit metadata (auto trigger)
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
