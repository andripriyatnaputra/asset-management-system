package models

import "time"

// AuditSession merepresentasikan sesi audit fisik aset (stok opname atau verifikasi lapangan).
type AuditSession struct {
	ID                  int64      `json:"id" db:"id"`
	Name                string     `json:"name" db:"name"`
	Status              string     `json:"status" db:"status"` // open | in_progress | closed
	CreatedByEmployeeID int64      `json:"created_by_employee_id" db:"created_by_employee_id"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// 🔸 Transient linkage (optional join info)
	CreatedByName *string `json:"created_by_name,omitempty"`
}

// AuditedAsset merepresentasikan hasil audit dari satu aset dalam sesi tertentu.
type AuditedAsset struct {
	ID          int64      `json:"id" db:"id"`
	SessionID   int64      `json:"session_id" db:"session_id"`
	AssetID     int64      `json:"asset_id" db:"asset_id"`
	Status      string     `json:"status" db:"status"` // found | missing | damaged
	VerifiedBy  *int64     `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	FoundAt     *time.Time `json:"found_at,omitempty" db:"found_at"`
	Notes       *string    `json:"notes,omitempty" db:"notes"`

	// 🔸 Transient linkage (optional join info)
	AssetTag        *string `json:"asset_tag,omitempty"`
	AssetName       *string `json:"asset_name,omitempty"`
	VerifiedByName  *string `json:"verified_by_name,omitempty"`
	SessionName     *string `json:"session_name,omitempty"`
}

// AuditLog merepresentasikan jejak aktivitas sistem (hash-chained log) — Grade A++ compliant.
type AuditLog struct {
	ID          int64     `json:"id" db:"id"`
	ActorID     *int64    `json:"actor_id,omitempty" db:"actor_id"`
	EntityName  string    `json:"entity_name" db:"entity_name"`
	EntityID    *int64    `json:"entity_id,omitempty" db:"entity_id"`
	Action      string    `json:"action" db:"action"`   // insert, update, delete, etc.
	Changes     string    `json:"changes" db:"changes"` // JSON of modified fields
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	IPAddress   *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent   *string   `json:"user_agent,omitempty" db:"user_agent"`
	RequestPath *string   `json:"request_path,omitempty" db:"request_path"`
	Hash        *string   `json:"hash,omitempty" db:"hash"`           // diisi otomatis trigger
	PrevHash    *string   `json:"prev_hash,omitempty" db:"prev_hash"` // diisi otomatis trigger
}
