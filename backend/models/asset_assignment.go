package models

import "time"

// AssetAssignment merepresentasikan riwayat penugasan atau peminjaman aset
// sesuai ISO/IEC 19770-10 Grade A++.
type AssetAssignment struct {
	ID         int64      `json:"id" db:"id"`
	AssetID    int64      `json:"asset_id" db:"asset_id"`
	EmployeeID int64      `json:"employee_id" db:"employee_id"`           // penerima aset
	AssignedBy *int64     `json:"assigned_by,omitempty" db:"assigned_by"` // petugas yang memberikan
	ReturnedBy *int64     `json:"returned_by,omitempty" db:"returned_by"` // petugas yang menerima kembali
	AssignedAt time.Time  `json:"assigned_at" db:"assigned_at"`
	ReturnedAt *time.Time `json:"returned_at,omitempty" db:"returned_at"`
	Status     string     `json:"status" db:"status"` // active, returned, lost, damaged
	Notes      *string    `json:"notes,omitempty" db:"notes"`

	// 🔹 Compliance & audit flag
	ComplianceFlag *bool   `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote *string `json:"compliance_note,omitempty" db:"compliance_note"`

	// 🔹 Metadata (auto trigger)
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (untuk view / handler)
	AssetName    *string `json:"asset_name,omitempty"`
	EmployeeName *string `json:"employee_name,omitempty"`
	Department   *string `json:"department,omitempty"`
}
