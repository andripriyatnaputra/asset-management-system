package models

import "time"

// RoleDelegation merepresentasikan pendelegasian peran sementara (role override)
// sesuai ISO/IEC 19770-10:2025 (Grade A++).
type RoleDelegation struct {
	ID           int64      `json:"id" db:"id"`
	DelegatorID  int64      `json:"delegator_id" db:"delegator_id"`   // pemberi wewenang
	DelegateeID  int64      `json:"delegatee_id" db:"delegatee_id"`   // penerima delegasi
	RoleOverride string     `json:"role_override" db:"role_override"` // peran yang diberikan sementara
	StartDate    time.Time  `json:"start_date" db:"start_date"`
	EndDate      time.Time  `json:"end_date" db:"end_date"`
	IsActive     bool       `json:"is_active" db:"is_active"`     // true = aktif saat ini
	Reason       *string    `json:"reason,omitempty" db:"reason"` // alasan delegasi
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`

	// 🔹 Audit metadata
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (tampilan)
	DelegatorName *string `json:"delegator_name,omitempty"`
	DelegateeName *string `json:"delegatee_name,omitempty"`
}
