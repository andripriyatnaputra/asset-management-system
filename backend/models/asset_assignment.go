package models
import "time"
type AssetAssignment struct {
	ID         int64      `json:"id"`
	AssetID    int64      `json:"asset_id"`
	EmployeeID int64      `json:"employee_id"`
	AssignedAt time.Time  `json:"assigned_at"`
	ReturnedAt *time.Time `json:"returned_at,omitempty"`
	Notes      string     `json:"notes"`
}
