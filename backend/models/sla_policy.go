package models

import "time"

// SLAPolicy mendefinisikan kebijakan SLA dinamis untuk tiket atau layanan,
// sesuai ISO/IEC 19770-10:2025 (ITSM Grade A++).
type SLAPolicy struct {
	ID                int64    `json:"id" db:"id"`
	Name              string   `json:"name" db:"name"`
	Description       *string  `json:"description,omitempty" db:"description"`
	CategoryCode      *string  `json:"category_code,omitempty" db:"category_code"`
	ServiceCode       *string  `json:"service_code,omitempty" db:"service_code"`
	Impact            string   `json:"impact" db:"impact"`                         // Low | Medium | High
	Urgency           string   `json:"urgency" db:"urgency"`                       // Low | Medium | High
	ResultingPriority string   `json:"resulting_priority" db:"resulting_priority"` // Low | Medium | High | Critical
	ResponseMinutes   int      `json:"response_minutes" db:"response_minutes"`     // waktu target respon (menit)
	ResolveMinutes    int      `json:"resolve_minutes" db:"resolve_minutes"`       // waktu target resolusi (menit)
	IsActive          bool     `json:"is_active" db:"is_active"`
	ComplianceScore   *float64 `json:"compliance_score,omitempty" db:"compliance_score"`

	// 🔹 Audit metadata
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
