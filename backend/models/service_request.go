package models

import "time"

// ServiceCatalog mendefinisikan layanan yang tersedia untuk di-request.
// Sesuai ISO 20000-1 Cl. 8.6 - Service Request Management.
type ServiceCatalog struct {
	ID                    int64      `json:"id" db:"id"`
	Code                  string     `json:"code" db:"code"`
	Name                  string     `json:"name" db:"name"`
	Category              *string    `json:"category,omitempty" db:"category"`
	Description           *string    `json:"description,omitempty" db:"description"`
	SLAPolicyID           *int64     `json:"sla_policy_id,omitempty" db:"sla_policy_id"`
	ApprovalRequired      bool       `json:"approval_required" db:"approval_required"`
	FulfillmentSLAMinutes *int       `json:"fulfillment_sla_minutes,omitempty" db:"fulfillment_sla_minutes"`
	IsActive              bool       `json:"is_active" db:"is_active"`
	CreatedBy             *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt             *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Transient
	SLAPolicyName *string `json:"sla_policy_name,omitempty"`
	CreatedByName *string `json:"created_by_name,omitempty"`
}

// ServiceRequest merepresentasikan permintaan layanan dari pengguna.
// Workflow: submitted → pending_approval → approved → in_fulfillment → completed
//           atau submitted/approved → rejected/cancelled.
type ServiceRequest struct {
	ID                int64      `json:"id" db:"id"`
	SRNumber          string     `json:"sr_number" db:"sr_number"` // e.g. SR-2026-0001
	ServiceCatalogID  int64      `json:"service_catalog_id" db:"service_catalog_id"`
	Subject           string     `json:"subject" db:"subject"`
	Description       *string    `json:"description,omitempty" db:"description"`
	Status            string     `json:"status" db:"status"`   // submitted…completed/cancelled/rejected
	Priority          string     `json:"priority" db:"priority"` // Low|Medium|High|Critical
	RequestedBy       int64      `json:"requested_by" db:"requested_by"`
	AssignedTo        *int64     `json:"assigned_to,omitempty" db:"assigned_to"`
	DepartmentID      *int64     `json:"department_id,omitempty" db:"department_id"`
	RelatedAssetID    *int64     `json:"related_asset_id,omitempty" db:"related_asset_id"`
	Notes             *string    `json:"notes,omitempty" db:"notes"`
	FulfilledAt       *time.Time `json:"fulfilled_at,omitempty" db:"fulfilled_at"`
	ClosedAt          *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ServiceRequestInfo adalah versi ringkas untuk list dengan join data.
type ServiceRequestInfo struct {
	ID                  int64      `json:"id"`
	SRNumber            string     `json:"sr_number"`
	ServiceCatalogID    int64      `json:"service_catalog_id"`
	ServiceCatalogName  string     `json:"service_catalog_name"`
	ServiceCatalogCode  string     `json:"service_catalog_code"`
	Subject             string     `json:"subject"`
	Status              string     `json:"status"`
	Priority            string     `json:"priority"`
	RequestedBy         int64      `json:"requested_by"`
	RequestedByName     string     `json:"requested_by_name"`
	AssignedTo          *int64     `json:"assigned_to,omitempty"`
	AssignedToName      *string    `json:"assigned_to_name,omitempty"`
	DepartmentID        *int64     `json:"department_id,omitempty"`
	DepartmentName      *string    `json:"department_name,omitempty"`
	RelatedAssetID      *int64     `json:"related_asset_id,omitempty"`
	RelatedAssetName    *string    `json:"related_asset_name,omitempty"`
	ApprovalRequired    bool       `json:"approval_required"`
	FulfilledAt         *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	PendingApprovals    int        `json:"pending_approvals"`
}

// ServiceRequestDetail adalah tampilan lengkap dengan approval history.
type ServiceRequestDetail struct {
	ServiceRequestInfo
	Description *string            `json:"description,omitempty"`
	Notes       *string            `json:"notes,omitempty"`
	Approvals   []ApprovalWorkflow `json:"approvals,omitempty"`
}

// ApprovalWorkflow merepresentasikan satu langkah persetujuan (multi-level, multi-entity).
// Mendukung: service_request dan change_request.
type ApprovalWorkflow struct {
	ID         int64      `json:"id" db:"id"`
	EntityType string     `json:"entity_type" db:"entity_type"` // service_request | change_request
	EntityID   int64      `json:"entity_id" db:"entity_id"`
	Level      int        `json:"level" db:"level"`
	ApproverID int64      `json:"approver_id" db:"approver_id"`
	Status     string     `json:"status" db:"status"` // pending|approved|rejected|skipped
	Comment    *string    `json:"comment,omitempty" db:"comment"`
	DecidedAt  *time.Time `json:"decided_at,omitempty" db:"decided_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`

	// Transient
	ApproverName *string `json:"approver_name,omitempty"`
}
