// File: backend/models/ticket.go
package models

import "time"

type Ticket struct {
	ID                   int64      `json:"id"`
	Subject              string     `json:"subject"`
	Description          string     `json:"description"`
	Status               string     `json:"status"`
	Priority             string     `json:"priority"`
	CreatedByEmployeeID  int64      `json:"created_by_employee_id"`
	AssignedToEmployeeID *int64     `json:"assigned_to_employee_id,omitempty"` // Pointer for nullable
	RelatedAssetID       *int64     `json:"related_asset_id,omitempty"`        // Pointer for nullable
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"-"`
}

type TicketComment struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	EmployeeID int64     `json:"employee_id"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type TicketInfo struct {
	ID                    int64     `json:"id"`
	Subject               string    `json:"subject"`
	Status                string    `json:"status"`
	Priority              string    `json:"priority"`
	CreatedByEmployeeID   int64     `json:"created_by_employee_id"`
	CreatedByEmployeeName string    `json:"created_by_employee_name"`
	RelatedAssetID        *int64    `json:"related_asset_id,omitempty"` // <-- TAMBAHKAN BARIS INI
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
type TicketCommentInfo struct {
	ID           int64     `json:"id"`
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	Comment      string    `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// Tambahkan struct ini untuk respons detail tiket yang lengkap
type TicketDetail struct {
	TicketInfo
	Description     string                `json:"description"`
	Comments        []TicketCommentInfo   `json:"comments"`
	MaintenanceLogs []AssetMaintenanceLog `json:"maintenance_logs"`
}
