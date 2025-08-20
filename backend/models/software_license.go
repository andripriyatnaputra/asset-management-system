// File: backend/models/software_license.go
package models

import "time"

type SoftwareLicense struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	LicenseKey     *string    `json:"license_key,omitempty"` // Pointer for nullable
	TotalSeats     int        `json:"total_seats"`
	PurchaseDate   *time.Time `json:"purchase_date,omitempty"`   // Pointer for nullable
	ExpirationDate *time.Time `json:"expiration_date,omitempty"` // Pointer for nullable
	Cost           *float64   `json:"cost,omitempty"`            // Pointer for nullable
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"-"` // For soft delete
}

type SoftwareInstallation struct {
	ID               int64     `json:"id"`
	AssetID          int64     `json:"asset_id"`
	LicenseID        int64     `json:"license_id"`
	InstallationDate time.Time `json:"installation_date"`
	Notes            *string   `json:"notes,omitempty"`
}

// Struct for joining data
type InstalledSoftwareInfo struct {
	InstallationID   int64     `json:"installation_id"`
	LicenseID        int64     `json:"license_id"`
	LicenseName      string    `json:"license_name"`
	LicenseKey       *string   `json:"license_key,omitempty"`
	InstallationDate time.Time `json:"installation_date"`
}
