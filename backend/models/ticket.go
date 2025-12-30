package models

import "time"

// Ticket merepresentasikan entitas tiket ITSM (incident / request / problem / change)
// sesuai ISO/IEC 19770-10:2025 (Grade A++).
type Ticket struct {
	ID                    int64      `json:"id" db:"id"`
	Subject               string     `json:"subject" db:"subject"`
	Description           string     `json:"description" db:"description"`
	Status                string     `json:"status" db:"status"`     // Open, In Progress, Resolved, Closed
	Priority              string     `json:"priority" db:"priority"` // Low, Medium, High, Critical
	CategoryCode          *string    `json:"category_code,omitempty" db:"category_code"`
	ServiceCode           *string    `json:"service_code,omitempty" db:"service_code"`
	Impact                *string    `json:"impact,omitempty" db:"impact"`
	Urgency               *string    `json:"urgency,omitempty" db:"urgency"`
	CreatedByEmployeeID   int64      `json:"created_by_employee_id" db:"created_by_employee_id"`
	AssignedToEmployeeID  *int64     `json:"assigned_to_employee_id,omitempty" db:"assigned_to_employee_id"`
	ClosedByEmployeeID    *int64     `json:"closed_by_employee_id,omitempty" db:"closed_by_employee_id"`
	RelatedAssetID        *int64     `json:"related_asset_id,omitempty" db:"related_asset_id"`
	SLAPolicyID           *int64     `json:"sla_policy_id,omitempty" db:"sla_policy_id"`
	ResponseDueAt         *time.Time `json:"response_due_at,omitempty" db:"response_due_at"`
	ResponseCompletedAt   *time.Time `json:"response_completed_at,omitempty" db:"response_completed_at"`
	SLADueAt              *time.Time `json:"sla_due_at,omitempty" db:"sla_due_at"`
	SLABreachedAt         *time.Time `json:"sla_breached_at,omitempty" db:"sla_breached_at"`
	ResponseTimeMinutes   *int       `json:"response_time_minutes,omitempty" db:"response_time_minutes"`
	ResolutionTimeMinutes *int       `json:"resolution_time_minutes,omitempty" db:"resolution_time_minutes"`
	ResolutionDate        *time.Time `json:"resolution_date,omitempty" db:"resolution_date"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ClosedAt              *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	CategoryTier          *string    `json:"category_tier,omitempty" db:"category_tier"`
	LinkedProblemID       *int64     `json:"linked_problem_id,omitempty" db:"linked_problem_id"`
	EscalationLevel       int        `json:"escalation_level" db:"escalation_level"`
	BreachFlag            bool       `json:"breach_flag" db:"breach_flag"`

	// 🔹 Compliance
	ComplianceFlag  *bool    `json:"compliance_flag,omitempty" db:"compliance_flag"`
	ComplianceNote  *string  `json:"compliance_note,omitempty" db:"compliance_note"`
	ComplianceScore *float64 `json:"compliance_score,omitempty" db:"compliance_score"`

	// 🔹 Audit metadata
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ===========================================================
// TicketInfo dan turunan — struktur tampilan / join
// ===========================================================
type TicketInfo struct {
	ID                     int64      `json:"id"`
	Subject                string     `json:"subject"`
	Status                 string     `json:"status"`
	Priority               string     `json:"priority"`
	CategoryCode           *string    `json:"category_code,omitempty"`
	ServiceCode            *string    `json:"service_code,omitempty"`
	Impact                 *string    `json:"impact,omitempty"`
	Urgency                *string    `json:"urgency,omitempty"`
	CreatedByEmployeeID    int64      `json:"created_by_employee_id"`
	CreatedByEmployeeName  string     `json:"created_by_employee_name"`
	AssignedToEmployeeID   *int64     `json:"assigned_to_employee_id,omitempty"`
	AssignedToEmployeeName *string    `json:"assigned_to_employee_name,omitempty"`
	RelatedAssetID         *int64     `json:"related_asset_id,omitempty"`
	ResponseDueAt          *time.Time `json:"response_due_at,omitempty"`
	SLADueAt               *time.Time `json:"sla_due_at,omitempty"`
	SLABreachedAt          *time.Time `json:"sla_breached_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	ResolvedAt             *time.Time `json:"resolved_at,omitempty"`
	ClosedAt               *time.Time `json:"closed_at,omitempty"`
	BreachFlag             bool       `json:"breach_flag"`
	LastAssignedBy         *int64     `json:"last_assigned_by,omitempty"`
	LastAssignedByName     *string    `json:"last_assigned_by_name,omitempty"`
}

// ===========================================================
// Komentar, lampiran, dan detail tiket
// ===========================================================
type TicketComment struct {
	ID         int64     `json:"id" db:"id"`
	TicketID   int64     `json:"ticket_id" db:"ticket_id"`
	EmployeeID int64     `json:"employee_id" db:"employee_id"`
	Comment    string    `json:"comment" db:"comment"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type TicketAttachment struct {
	ID        int64     `json:"id" db:"id"`
	TicketID  int64     `json:"ticket_id" db:"ticket_id"`
	CommentID *int64    `json:"comment_id,omitempty" db:"comment_id"`
	Filename  string    `json:"filename" db:"filename"`
	Path      string    `json:"path" db:"path"`
	URL       string    `json:"url" db:"url"`
	MimeType  *string   `json:"mime_type,omitempty" db:"mime_type"`
	Size      *int64    `json:"size,omitempty" db:"size"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Info komentar dengan lampiran
type TicketCommentInfo struct {
	ID           int64              `json:"id"`
	EmployeeID   int64              `json:"employee_id"`
	EmployeeName string             `json:"employee_name"`
	Comment      string             `json:"comment"`
	CreatedAt    time.Time          `json:"created_at"`
	Attachments  []TicketAttachment `json:"attachments,omitempty"`
}

// Detail tiket untuk tampilan penuh
type TicketDetail struct {
	TicketInfo
	Description     string                `json:"description"`
	Comments        []TicketCommentInfo   `json:"comments,omitempty"`
	MaintenanceLogs []AssetMaintenanceLog `json:"maintenance_logs,omitempty"`
}
