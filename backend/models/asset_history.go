// File: backend/models/asset_history.go
package models

import "time"

// AssetHistoryResponse adalah struct yang digunakan untuk menampilkan
// satu baris riwayat aset yang lebih mudah dibaca, dengan menyertakan nama karyawan.
type AssetHistoryResponse struct {
	AssignmentID int64      `json:"assignment_id"`
	EmployeeNIK  string     `json:"employee_nik"`
	EmployeeName string     `json:"employee_name"`
	AssignedAt   time.Time  `json:"assigned_at"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	Notes        string     `json:"notes"`
}
