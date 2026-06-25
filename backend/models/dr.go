package models

import "time"

// DRPlan adalah rencana pemulihan bencana / kelangsungan bisnis.
// Sesuai ISO 22301 (BCMS) dan ITIL Continual Improvement.
type DRPlan struct {
	ID           int64      `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Description  *string    `json:"description,omitempty" db:"description"`
	PlanType     string     `json:"plan_type" db:"plan_type"`      // dr|bcp|contingency
	RTOHours     *float64   `json:"rto_hours,omitempty" db:"rto_hours"`
	RPOHours     *float64   `json:"rpo_hours,omitempty" db:"rpo_hours"`
	Status       string     `json:"status" db:"status"` // draft|active|archived|under_review
	OwnerID      *int64     `json:"owner_id,omitempty" db:"owner_id"`
	LastTestedAt *time.Time `json:"last_tested_at,omitempty" db:"last_tested_at"`
	NextTestDue  *time.Time `json:"next_test_due,omitempty" db:"next_test_due"`
	CreatedBy    *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	OwnerName     *string    `json:"owner_name,omitempty"`
	CreatedByName *string    `json:"created_by_name,omitempty"`
	StepCount     *int       `json:"step_count,omitempty"`
	Steps         []DRPlanStep `json:"steps,omitempty"`
}

// DRPlanStep adalah langkah individual dalam rencana DR/BCP.
type DRPlanStep struct {
	ID              int64   `json:"id" db:"id"`
	PlanID          int64   `json:"plan_id" db:"plan_id"`
	StepOrder       int     `json:"step_order" db:"step_order"`
	Title           string  `json:"title" db:"title"`
	Description     *string `json:"description,omitempty" db:"description"`
	Responsible     *int64  `json:"responsible,omitempty" db:"responsible"`
	DurationMinutes *int    `json:"duration_minutes,omitempty" db:"duration_minutes"`
	IsCritical      bool    `json:"is_critical" db:"is_critical"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`

	// Transient
	ResponsibleName *string `json:"responsible_name,omitempty"`
}

// DRTest adalah sesi pengujian DR/BCP (tabletop, simulasi, full test).
type DRTest struct {
	ID                int64      `json:"id" db:"id"`
	PlanID            int64      `json:"plan_id" db:"plan_id"`
	TestType          string     `json:"test_type" db:"test_type"` // tabletop|walkthrough|simulation|full_test
	ScheduledAt       time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt         *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Status            string     `json:"status" db:"status"` // scheduled|in_progress|completed|cancelled
	RTOAchievedHours  *float64   `json:"rto_achieved_hours,omitempty" db:"rto_achieved_hours"`
	RPOAchievedHours  *float64   `json:"rpo_achieved_hours,omitempty" db:"rpo_achieved_hours"`
	Outcome           *string    `json:"outcome,omitempty" db:"outcome"` // passed|partial|failed
	Notes             *string    `json:"notes,omitempty" db:"notes"`
	ConductedBy       *int64     `json:"conducted_by,omitempty" db:"conducted_by"`
	CreatedBy         *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	PlanName        *string `json:"plan_name,omitempty"`
	ConductedByName *string `json:"conducted_by_name,omitempty"`
}

// DRTestResult menyimpan hasil per langkah dari sebuah DR test.
type DRTestResult struct {
	ID                     int64   `json:"id" db:"id"`
	TestID                 int64   `json:"test_id" db:"test_id"`
	StepID                 *int64  `json:"step_id,omitempty" db:"step_id"`
	Status                 string  `json:"status" db:"status"` // passed|failed|skipped|not_tested
	ActualDurationMinutes  *int    `json:"actual_duration_minutes,omitempty" db:"actual_duration_minutes"`
	Notes                  *string `json:"notes,omitempty" db:"notes"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`

	// Transient
	StepTitle *string `json:"step_title,omitempty"`
	StepOrder *int    `json:"step_order,omitempty"`
}
