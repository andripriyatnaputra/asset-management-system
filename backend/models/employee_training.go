package models

import "time"

type EmployeeTraining struct {
	ID             int64      `json:"id"`
	EmployeeID     int64      `json:"employee_id"`
	TrainingName   string     `json:"training_name"`
	CertificateURL *string    `json:"certificate_url,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
