// File: backend/models/asset_maintenance.go
package models

import "time"

type AssetMaintenanceLog struct {
	ID          int64     `json:"id"`
	AssetID     int64     `json:"asset_id"`
	LogType     string    `json:"log_type"`
	Description string    `json:"description"`
	Cost        float64   `json:"cost"`
	LogDate     time.Time `json:"log_date"`
	CreatedAt   time.Time `json:"created_at"`
	TicketID    *int64    `json:"ticket_id,omitempty"`
}
