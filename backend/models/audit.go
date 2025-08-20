// File: backend/models/audit.go
package models

import "time"

type AuditSession struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Status              string     `json:"status"`
	CreatedByEmployeeID int64      `json:"created_by_employee_id"`
	CreatedAt           time.Time  `json:"created_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
}

type AuditedAsset struct {
	ID        int64      `json:"id"`
	SessionID int64      `json:"session_id"`
	AssetID   int64      `json:"asset_id"`
	Status    string     `json:"status"`
	FoundAt   *time.Time `json:"found_at,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
}
