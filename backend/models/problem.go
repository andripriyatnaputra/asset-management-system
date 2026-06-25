package models

import "time"

// Problem merepresentasikan entitas Problem Management sesuai ITIL & ISO 20000-1 Cl. 8.7.
type Problem struct {
	ID                 int64      `json:"id" db:"id"`
	Title              string     `json:"title" db:"title"`
	Description        *string    `json:"description,omitempty" db:"description"`
	Status             string     `json:"status" db:"status"`   // Open | Under Investigation | Known Error | Resolved | Closed
	Priority           string     `json:"priority" db:"priority"` // Low | Medium | High | Critical
	AssignedTo         *int64     `json:"assigned_to,omitempty" db:"assigned_to"`
	RootCause          *string    `json:"root_cause,omitempty" db:"root_cause"`
	Workaround         *string    `json:"workaround,omitempty" db:"workaround"`
	KnownError         bool       `json:"known_error" db:"known_error"`
	PermanentSolution  *string    `json:"permanent_solution,omitempty" db:"permanent_solution"`
	RelatedAssetID     *int64     `json:"related_asset_id,omitempty" db:"related_asset_id"`
	CreatedBy          *int64     `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy          *int64     `json:"updated_by,omitempty" db:"updated_by"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ProblemInfo adalah versi Problem dengan data join untuk tampilan list.
type ProblemInfo struct {
	ID               int64      `json:"id"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	Status           string     `json:"status"`
	Priority         string     `json:"priority"`
	KnownError       bool       `json:"known_error"`
	AssignedTo       *int64     `json:"assigned_to,omitempty"`
	AssigneeName     *string    `json:"assignee_name,omitempty"`
	RelatedAssetID   *int64     `json:"related_asset_id,omitempty"`
	RelatedAssetName *string    `json:"related_asset_name,omitempty"`
	CreatedBy        *int64     `json:"created_by,omitempty"`
	CreatedByName    *string    `json:"created_by_name,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
	IncidentCount    int        `json:"incident_count"`
}

// ProblemDetail adalah tampilan lengkap Problem dengan incidents terkait.
type ProblemDetail struct {
	ProblemInfo
	Workaround        *string           `json:"workaround,omitempty"`
	RootCause         *string           `json:"root_cause,omitempty"`
	PermanentSolution *string           `json:"permanent_solution,omitempty"`
	LinkedIncidents   []ProblemIncident `json:"linked_incidents,omitempty"`
	Postmortem        *IncidentPostmortem `json:"postmortem,omitempty"`
}

// ProblemIncident merepresentasikan relasi many-to-many antara Problem dan Ticket (incident).
type ProblemIncident struct {
	ID        int64      `json:"id" db:"id"`
	ProblemID int64      `json:"problem_id" db:"problem_id"`
	TicketID  int64      `json:"ticket_id" db:"ticket_id"`
	LinkedBy  *int64     `json:"linked_by,omitempty" db:"linked_by"`
	LinkedAt  time.Time  `json:"linked_at" db:"linked_at"`
	Notes     *string    `json:"notes,omitempty" db:"notes"`

	// Transient join fields
	TicketSubject  *string `json:"ticket_subject,omitempty"`
	TicketStatus   *string `json:"ticket_status,omitempty"`
	TicketPriority *string `json:"ticket_priority,omitempty"`
	LinkedByName   *string `json:"linked_by_name,omitempty"`
}

// IncidentPostmortem merepresentasikan post-mortem / Root Cause Analysis dari incident.
// Sesuai ITIL Post-Incident Review & ISO 20000-1 Cl. 8.6.
type IncidentPostmortem struct {
	ID                  int64      `json:"id" db:"id"`
	TicketID            int64      `json:"ticket_id" db:"ticket_id"`
	ProblemID           *int64     `json:"problem_id,omitempty" db:"problem_id"`
	Timeline            string     `json:"timeline" db:"timeline"`           // JSON array of timeline events
	RootCause           *string    `json:"root_cause,omitempty" db:"root_cause"`
	ContributingFactors *string    `json:"contributing_factors,omitempty" db:"contributing_factors"`
	LessonsLearned      *string    `json:"lessons_learned,omitempty" db:"lessons_learned"`
	ActionItems         string     `json:"action_items" db:"action_items"`   // JSON array of action items
	ReviewedBy          *int64     `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedBy           *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`

	// Transient join fields
	TicketSubject  *string `json:"ticket_subject,omitempty"`
	ReviewedByName *string `json:"reviewed_by_name,omitempty"`
	CreatedByName  *string `json:"created_by_name,omitempty"`
}

// EscalationRule mendefinisikan aturan eskalasi otomatis untuk ticket.
// Sesuai ITIL Incident Management escalation & ISO 20000-1 Cl. 8.6.3.
type EscalationRule struct {
	ID                   int64      `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	CategoryCode         *string    `json:"category_code,omitempty" db:"category_code"`
	ServiceCode          *string    `json:"service_code,omitempty" db:"service_code"`
	Priority             string     `json:"priority" db:"priority"` // Low | Medium | High | Critical
	TriggerAfterMinutes  int        `json:"trigger_after_minutes" db:"trigger_after_minutes"`
	Action               string     `json:"action" db:"action"` // reassign | notify | raise_priority | raise_escalation_level
	EscalateToRole       *string    `json:"escalate_to_role,omitempty" db:"escalate_to_role"`
	EscalateToEmployee   *int64     `json:"escalate_to_employee,omitempty" db:"escalate_to_employee"`
	NotifyEmails         *string    `json:"notify_emails,omitempty" db:"notify_emails"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	CreatedBy            *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`

	// Transient join fields
	EscalateToEmployeeName *string `json:"escalate_to_employee_name,omitempty"`
}
