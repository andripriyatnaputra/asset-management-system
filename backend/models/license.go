package models

import "time"

// License merepresentasikan lisensi perangkat lunak (software license)
// sesuai ISO/IEC 19770-10:2025 (ITAM Grade A++).
// License merepresentasikan lisensi perangkat lunak (software license)
// sesuai ISO/IEC 19770-10:2025 (ITAM Grade A++).
type License struct {
	ID             int64   `json:"id" db:"id"`
	Name           string  `json:"name" db:"name"`
	LicenseKey     *string `json:"license_key,omitempty" db:"license_key"`
	Vendor         *string `json:"vendor,omitempty" db:"vendor"`
	Publisher      *string `json:"publisher,omitempty" db:"publisher"`
	Version        *string `json:"version,omitempty" db:"version"`
	LicenseType    *string `json:"license_type,omitempty" db:"license_type"`   // perpetual / subscription
	LicenseModel   *string `json:"license_model,omitempty" db:"license_model"` // user / device / site / concurrent
	LicenseScope   *string `json:"license_scope,omitempty" db:"license_scope"` // global / department / single-user
	Metric         *string `json:"metric,omitempty" db:"metric"`
	Category       *string `json:"category,omitempty" db:"category"` // software / os / security / etc.
	ContractID     *int64  `json:"contract_id,omitempty" db:"contract_id"`
	ContractNumber *string `json:"contract_number,omitempty" db:"contract_number"`
	BudgetID       *int64  `json:"budget_id,omitempty" db:"budget_id"`

	// Seats & biaya
	TotalSeats        int        `json:"total_seats" db:"total_seats"`
	UsedSeats         *int       `json:"used_seats,omitempty" db:"used_seats"` // alias dari subquery di handler
	Cost              *float64   `json:"cost,omitempty" db:"cost"`
	Currency          *string    `json:"currency,omitempty" db:"currency"` // default: IDR
	PurchaseDate      *time.Time `json:"purchase_date,omitempty" db:"purchase_date"`
	ExpirationDate    *time.Time `json:"expiration_date,omitempty" db:"expiration_date"`
	MaintenanceExpiry *time.Time `json:"maintenance_expiry,omitempty" db:"maintenance_expiry"`
	VerificationDate  *time.Time `json:"verification_date,omitempty" db:"verification_date"`

	EntitlementDoc       *string `json:"entitlement_doc,omitempty" db:"entitlement_doc"`
	ProcurementReference *string `json:"procurement_reference,omitempty" db:"procurement_reference"`

	// 🔹 Kepatuhan
	ComplianceFlag   *bool    `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote   *string  `json:"compliance_note,omitempty" db:"compliance_note"`
	ComplianceScore  *float64 `json:"compliance_score,omitempty" db:"compliance_score"`
	ComplianceStatus *string  `json:"status,omitempty" db:"compliance_status"`

	// 🔹 Audit metadata
	Active    *bool      `json:"active,omitempty" db:"active"`
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (untuk view)
	ContractName *string  `json:"contract_name,omitempty"`
	BudgetName   *string  `json:"budget_name,omitempty"`
	Department   *string  `json:"department,omitempty"`
	UsagePercent *float64 `json:"usage_percent,omitempty"`
}

// SoftwareInstallation mencatat hubungan lisensi–aset (installasi aktual).
type SoftwareInstallation struct {
	ID               int64      `json:"id" db:"id"`
	AssetID          int64      `json:"asset_id" db:"asset_id"`
	LicenseID        int64      `json:"license_id" db:"license_id"`
	InstallationDate time.Time  `json:"installation_date" db:"installation_date"`
	InstalledBy      *int64     `json:"installed_by,omitempty" db:"installed_by"`
	RemovedAt        *time.Time `json:"removed_at,omitempty" db:"removed_at"`
	Notes            *string    `json:"notes,omitempty" db:"notes"`

	// 🔹 Metadata
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage
	AssetName       *string `json:"asset_name,omitempty"`
	LicenseName     *string `json:"license_name,omitempty"`
	InstalledByName *string `json:"installed_by_name,omitempty"`
}

// InstalledSoftwareInfo digunakan untuk tampilan ringkas join software terinstal.
type InstalledSoftwareInfo struct {
	InstallationID   int64     `json:"installation_id"`
	LicenseID        int64     `json:"license_id"`
	LicenseName      string    `json:"license_name"`
	LicenseKey       *string   `json:"license_key,omitempty"`
	InstallationDate time.Time `json:"installation_date"`
}
