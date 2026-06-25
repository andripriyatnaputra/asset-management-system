package models

import "time"

// ComplianceFramework adalah registry framework regulasi/standar (ISO 19770, ISO 20000, ITIL).
type ComplianceFramework struct {
	ID          int64     `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Name        string    `json:"name" db:"name"`
	Version     *string   `json:"version,omitempty" db:"version"`
	Description *string   `json:"description,omitempty" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Transient
	ControlCount *int `json:"control_count,omitempty"`
}

// ComplianceControl adalah kontrol individu di dalam framework.
type ComplianceControl struct {
	ID          int64     `json:"id" db:"id"`
	FrameworkID int64     `json:"framework_id" db:"framework_id"`
	ControlCode string    `json:"control_code" db:"control_code"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Category    *string   `json:"category,omitempty" db:"category"`
	Severity    *string   `json:"severity,omitempty" db:"severity"` // low|medium|high|critical
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`

	// Transient
	FrameworkName *string `json:"framework_name,omitempty"`
	EvidenceCount *int    `json:"evidence_count,omitempty"`
}

// ComplianceEvidence menghubungkan entitas sistem (asset, ticket, dll) ke kontrol kepatuhan.
type ComplianceEvidence struct {
	ID           int64      `json:"id" db:"id"`
	ControlID    int64      `json:"control_id" db:"control_id"`
	EntityType   string     `json:"entity_type" db:"entity_type"`   // asset|ticket|change_request|...
	EntityID     int64      `json:"entity_id" db:"entity_id"`
	EvidenceType string     `json:"evidence_type" db:"evidence_type"` // document|screenshot|log|...
	Title        string     `json:"title" db:"title"`
	Description  *string    `json:"description,omitempty" db:"description"`
	FileURL      *string    `json:"file_url,omitempty" db:"file_url"`
	Status       string     `json:"status" db:"status"` // pending|accepted|rejected|expired
	ReviewedBy   *int64     `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	SubmittedBy  *int64     `json:"submitted_by,omitempty" db:"submitted_by"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	ControlCode     *string `json:"control_code,omitempty"`
	ControlName     *string `json:"control_name,omitempty"`
	ReviewedByName  *string `json:"reviewed_by_name,omitempty"`
	SubmittedByName *string `json:"submitted_by_name,omitempty"`
}

// AssetDisposalCompliance adalah hasil view v_asset_disposal_compliance.
type AssetDisposalCompliance struct {
	AssetID                int64      `json:"asset_id"`
	AssetName              string     `json:"asset_name"`
	AssetTag               *string    `json:"asset_tag,omitempty"`
	LifecycleStage         *string    `json:"lifecycle_stage,omitempty"`
	DisposalRecordID       *int64     `json:"disposal_record_id,omitempty"`
	DisposalMethod         *string    `json:"disposal_method,omitempty"`
	DataWipeCompleted      *bool      `json:"data_wipe_completed,omitempty"`
	EnvironmentalCompliant *bool      `json:"environmental_compliant,omitempty"`
	CertificateNumber      *string    `json:"certificate_number,omitempty"`
	DateDisposed           *time.Time `json:"date_disposed,omitempty"`
	AuthorizedBy           *string    `json:"authorized_by,omitempty"`
	ExecutedBy             *string    `json:"executed_by,omitempty"`
	ComplianceStatus       string     `json:"compliance_status"` // compliant|data_wipe_pending|env_non_compliant|missing_record
}

// VendorPerformance mencatat KPI vendor per periode (SLA, response time, NPS).
type VendorPerformance struct {
	ID                int64     `json:"id" db:"id"`
	VendorName        string    `json:"vendor_name" db:"vendor_name"`
	ContractID        *int64    `json:"contract_id,omitempty" db:"contract_id"`
	PeriodStart       time.Time `json:"period_start" db:"period_start"`
	PeriodEnd         time.Time `json:"period_end" db:"period_end"`
	SLACompliancePct  *float64  `json:"sla_compliance_pct,omitempty" db:"sla_compliance_pct"`
	AvgResponseHours  *float64  `json:"avg_response_hours,omitempty" db:"avg_response_hours"`
	TotalTickets      int       `json:"total_tickets" db:"total_tickets"`
	OpenTickets       int       `json:"open_tickets" db:"open_tickets"`
	CriticalIncidents int       `json:"critical_incidents" db:"critical_incidents"`
	NPSScore          *int      `json:"nps_score,omitempty" db:"nps_score"`
	Notes             *string   `json:"notes,omitempty" db:"notes"`
	RecordedBy        *int64    `json:"recorded_by,omitempty" db:"recorded_by"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`

	// Transient
	ContractNumber *string `json:"contract_number,omitempty"`
	RecordedByName *string `json:"recorded_by_name,omitempty"`
}

// ServiceAvailability mencatat uptime/downtime per service per periode.
// Sesuai ITIL Availability Management dan ISO 20000-1.
type ServiceAvailability struct {
	ID                     int64     `json:"id" db:"id"`
	ServiceCode            string    `json:"service_code" db:"service_code"`
	PeriodStart            time.Time `json:"period_start" db:"period_start"`
	PeriodEnd              time.Time `json:"period_end" db:"period_end"`
	DowntimeMinutes        int       `json:"downtime_minutes" db:"downtime_minutes"`
	PlannedDowntimeMinutes int       `json:"planned_downtime_minutes" db:"planned_downtime_minutes"`
	IncidentCount          int       `json:"incident_count" db:"incident_count"`
	AvailabilityPct        float64   `json:"availability_pct" db:"availability_pct"`
	Notes                  *string   `json:"notes,omitempty" db:"notes"`
	RecordedBy             *int64    `json:"recorded_by,omitempty" db:"recorded_by"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`

	// Transient
	ServiceName    *string `json:"service_name,omitempty"`
	RecordedByName *string `json:"recorded_by_name,omitempty"`
}

// ServiceAvailabilitySummary adalah agregat per service.
type ServiceAvailabilitySummary struct {
	ServiceCode    string  `json:"service_code"`
	ServiceName    string  `json:"service_name"`
	AvgAvailPct    float64 `json:"avg_availability_pct"`
	TotalDowntime  int     `json:"total_downtime_minutes"`
	TotalIncidents int     `json:"total_incidents"`
	PeriodCount    int     `json:"period_count"`
}
