package models

import "time"

// Notification adalah pesan sistem kepada user tertentu.
type Notification struct {
	ID         int64      `json:"id" db:"id"`
	UserID     int64      `json:"user_id" db:"user_id"`
	Type       string     `json:"type" db:"type"`       // license_expiry|dr_test_due|evidence_expired|ticket_assigned|...
	Title      string     `json:"title" db:"title"`
	Message    string     `json:"message" db:"message"`
	EntityType *string    `json:"entity_type,omitempty" db:"entity_type"`
	EntityID   *int64     `json:"entity_id,omitempty" db:"entity_id"`
	IsRead     bool       `json:"is_read" db:"is_read"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}
