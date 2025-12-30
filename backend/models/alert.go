package models

import "time"

// Alert merepresentasikan notifikasi atau peringatan sistem,
// termasuk link ke entitas terkait (asset, ticket, SLA, dsb.).
type Alert struct {
	ID             int64      `json:"id" db:"id"`
	Message        string     `json:"message" db:"message"`
	Severity       string     `json:"severity" db:"severity"`       // info, warning, critical
	Category       string     `json:"category" db:"category"`       // system, asset, ticket, budget, etc.
	Source         *string    `json:"source,omitempty" db:"source"` // modul pemicu (optional)
	EntityName     *string    `json:"entity_name,omitempty" db:"entity_name"`
	EntityID       *int64     `json:"entity_id,omitempty" db:"entity_id"`
	Acknowledged   bool       `json:"acknowledged" db:"acknowledged"`
	AcknowledgedBy *int64     `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy     *int64     `json:"resolved_by,omitempty" db:"resolved_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// 🔸 Transient linkage (tidak tersimpan di DB, hanya untuk tampilan)
	AcknowledgedByName *string `json:"acknowledged_by_name,omitempty"`
	ResolvedByName     *string `json:"resolved_by_name,omitempty"`
}
