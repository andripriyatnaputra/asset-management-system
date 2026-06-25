package models

import "time"

// ChangeRequest merepresentasikan Change Management sesuai ISO 20000-1 Cl. 9.2 & ITIL.
// Type: standard (pre-approved), normal (CAB review), emergency (expedited).
// Workflow: draft → submitted → under_review → approved → scheduled →
//           implementing → implemented → verified → closed  (atau rejected).
type ChangeRequest struct {
	ID                 int64      `json:"id" db:"id"`
	CRNumber           string     `json:"cr_number" db:"cr_number"` // e.g. CR-2026-001
	Title              string     `json:"title" db:"title"`
	Description        *string    `json:"description,omitempty" db:"description"`
	Type               string     `json:"type" db:"type"`             // standard | normal | emergency
	Status             string     `json:"status" db:"status"`         // draft … closed | rejected
	RiskLevel          string     `json:"risk_level" db:"risk_level"` // low | medium | high | critical
	ImpactAssessment   *string    `json:"impact_assessment,omitempty" db:"impact_assessment"`
	RollbackPlan       *string    `json:"rollback_plan,omitempty" db:"rollback_plan"`
	ChangeWindowStart  *time.Time `json:"change_window_start,omitempty" db:"change_window_start"`
	ChangeWindowEnd    *time.Time `json:"change_window_end,omitempty" db:"change_window_end"`
	CABRequired        bool       `json:"cab_required" db:"cab_required"`
	RelatedAssetID     *int64     `json:"related_asset_id,omitempty" db:"related_asset_id"`
	RelatedTicketID    *int64     `json:"related_ticket_id,omitempty" db:"related_ticket_id"`
	CreatedBy          *int64     `json:"created_by,omitempty" db:"created_by"`
	ApprovedBy         *int64     `json:"approved_by,omitempty" db:"approved_by"`
	ImplementedBy      *int64     `json:"implemented_by,omitempty" db:"implemented_by"`
	SubmittedAt        *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	ApprovedAt         *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	ImplementedAt      *time.Time `json:"implemented_at,omitempty" db:"implemented_at"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	ClosedAt           *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ChangeRequestInfo adalah versi ringkas untuk tampilan list dengan join data.
type ChangeRequestInfo struct {
	ID                int64      `json:"id"`
	CRNumber          string     `json:"cr_number"`
	Title             string     `json:"title"`
	Type              string     `json:"type"`
	Status            string     `json:"status"`
	RiskLevel         string     `json:"risk_level"`
	CABRequired       bool       `json:"cab_required"`
	ChangeWindowStart *time.Time `json:"change_window_start,omitempty"`
	ChangeWindowEnd   *time.Time `json:"change_window_end,omitempty"`
	RelatedAssetID    *int64     `json:"related_asset_id,omitempty"`
	RelatedAssetName  *string    `json:"related_asset_name,omitempty"`
	RelatedTicketID   *int64     `json:"related_ticket_id,omitempty"`
	CreatedBy         *int64     `json:"created_by,omitempty"`
	CreatedByName     *string    `json:"created_by_name,omitempty"`
	ApprovedBy        *int64     `json:"approved_by,omitempty"`
	ApprovedByName    *string    `json:"approved_by_name,omitempty"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	ImplementedAt     *time.Time `json:"implemented_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	TaskTotal         int        `json:"task_total"`
	TaskDone          int        `json:"task_done"`
	ApprovalTotal     int        `json:"approval_total"`
	ApprovalApproved  int        `json:"approval_approved"`
}

// ChangeRequestDetail adalah tampilan penuh dengan approvals dan tasks.
type ChangeRequestDetail struct {
	ChangeRequestInfo
	Description      *string          `json:"description,omitempty"`
	ImpactAssessment *string          `json:"impact_assessment,omitempty"`
	RollbackPlan     *string          `json:"rollback_plan,omitempty"`
	Approvals        []ChangeApproval `json:"approvals,omitempty"`
	Tasks            []ChangeTask     `json:"tasks,omitempty"`
}

// ChangeApproval merepresentasikan satu suara dalam proses CAB.
type ChangeApproval struct {
	ID         int64      `json:"id" db:"id"`
	ChangeID   int64      `json:"change_id" db:"change_id"`
	ApproverID int64      `json:"approver_id" db:"approver_id"`
	Decision   string     `json:"decision" db:"decision"` // approved | rejected | abstain | pending
	Comment    *string    `json:"comment,omitempty" db:"comment"`
	DecidedAt  *time.Time `json:"decided_at,omitempty" db:"decided_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`

	// Transient
	ApproverName *string `json:"approver_name,omitempty"`
}

// ChangeTask merepresentasikan satu sub-task implementasi dalam Change Request.
type ChangeTask struct {
	ID          int64      `json:"id" db:"id"`
	ChangeID    int64      `json:"change_id" db:"change_id"`
	Title       string     `json:"title" db:"title"`
	Description *string    `json:"description,omitempty" db:"description"`
	Status      string     `json:"status" db:"status"` // pending | in_progress | done | skipped
	AssignedTo  *int64     `json:"assigned_to,omitempty" db:"assigned_to"`
	SeqOrder    int        `json:"seq_order" db:"seq_order"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	AssigneeName *string `json:"assignee_name,omitempty"`
}
