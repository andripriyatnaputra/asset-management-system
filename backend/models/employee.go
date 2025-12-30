package models

import "time"

// Employee merepresentasikan pengguna atau staf dalam sistem ITAM/ITSM.
// Versi ini sesuai dengan ISO/IEC 19770-10:2025 (Grade A++).
type Employee struct {
	ID           int64  `json:"id" db:"id"`
	EmployeeNIK  string `json:"employee_nik" db:"employee_nik"` // unique
	Name         string `json:"name" db:"name"`
	Email        string `json:"email" db:"email"` // unique
	DepartmentID *int64 `json:"department_id,omitempty" db:"department_id"`
	Role         string `json:"role" db:"role"` // super_admin, admin, manager, auditor, user
	PasswordHash string `json:"-" db:"password_hash"`

	// 🔹 Account control
	Active      *bool      `json:"active,omitempty" db:"active"`               // true = aktif
	LastLoginAt *time.Time `json:"last_login_at,omitempty" db:"last_login_at"` // diisi via middleware
	CreatedBy   *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy   *int64     `json:"updated_by,omitempty" db:"updated_by"`

	// 🔹 Metadata (auto trigger)
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (untuk tampilan / join)
	DepartmentName *string `json:"department_name,omitempty"`
	DelegatedRole  *string `json:"delegated_role,omitempty"` // dari role_delegations (opsional)
}
