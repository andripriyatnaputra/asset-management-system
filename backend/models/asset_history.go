package models

import "time"

// AssetHistory merekam setiap perubahan status atau siklus hidup aset
// sesuai schema ITAM Grade A++ (trigger-based audit trail).
type AssetHistory struct {
	ID             int64     `json:"id" db:"id"`
	AssetID        int64     `json:"asset_id" db:"asset_id"`
	FromStatus     *string   `json:"from_status,omitempty" db:"from_status"`
	ToStatus       *string   `json:"to_status,omitempty" db:"to_status"`
	ChangedBy      *int64    `json:"changed_by,omitempty" db:"changed_by"`
	ChangedAt      time.Time `json:"changed_at" db:"changed_at"`
	ComplianceFlag *bool     `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote *string   `json:"compliance_note,omitempty" db:"compliance_note"`
	Remarks        *string   `json:"remarks,omitempty" db:"remarks"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`

	// 🔸 Transient linkage (view join)
	AssetTag      *string `json:"asset_tag,omitempty"`
	AssetName     *string `json:"asset_name,omitempty"`
	ChangedByName *string `json:"changed_by_name,omitempty"`
}

// AssetHistoryResponse digunakan untuk tampilan ringkas frontend lama,
// menyertakan nama karyawan dan periode penugasan.
type AssetHistoryResponse struct {
	AssignmentID int64      `json:"assignment_id"`
	EmployeeNIK  string     `json:"employee_nik"`
	EmployeeName string     `json:"employee_name"`
	AssignedAt   time.Time  `json:"assigned_at"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	Notes        string     `json:"notes"`
}
