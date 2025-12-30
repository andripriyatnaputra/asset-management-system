package models

import "time"

// Department merepresentasikan unit organisasi atau divisi dalam sistem ITAM/ITSM.
// Versi ini sudah sesuai dengan ISO/IEC 19770-10:2025 (Grade A++).
type Department struct {
	ID                 int64      `json:"id" db:"id"`
	Name               string     `json:"name" db:"name"`
	ManagerID          *int64     `json:"manager_id,omitempty" db:"manager_id"`
	CostCenter         *string    `json:"cost_center,omitempty" db:"cost_center"`
	ParentDepartmentID *int64     `json:"parent_department_id,omitempty" db:"parent_department_id"` // mendukung hierarki
	Active             *bool      `json:"active,omitempty" db:"active"`                             // true = aktif
	CreatedBy          *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy          *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (untuk tampilan / join)
	ManagerName          *string `json:"manager_name,omitempty"`
	ParentDepartmentName *string `json:"parent_department_name,omitempty"`
}
