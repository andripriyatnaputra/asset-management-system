package models

import "time"

// AssetSpecification menyimpan detail hardware sesuai ISO 19770-2 (SWID tagging).
// Terhubung 1-to-1 dengan tabel assets.
type AssetSpecification struct {
	ID             int64      `json:"id" db:"id"`
	AssetID        int64      `json:"asset_id" db:"asset_id"`
	// CPU
	CPUModel       *string    `json:"cpu_model,omitempty" db:"cpu_model"`
	CPUCores       *int       `json:"cpu_cores,omitempty" db:"cpu_cores"`
	CPUSpeedGHz    *float64   `json:"cpu_speed_ghz,omitempty" db:"cpu_speed_ghz"`
	// Memory
	RAMGB          *float64   `json:"ram_gb,omitempty" db:"ram_gb"`
	RAMType        *string    `json:"ram_type,omitempty" db:"ram_type"`
	// Storage
	StorageGB      *float64   `json:"storage_gb,omitempty" db:"storage_gb"`
	StorageType    *string    `json:"storage_type,omitempty" db:"storage_type"`
	// Display
	ScreenSizeInch *float64   `json:"screen_size_inch,omitempty" db:"screen_size_inch"`
	Resolution     *string    `json:"resolution,omitempty" db:"resolution"`
	// Network
	MACAddress     *string    `json:"mac_address,omitempty" db:"mac_address"`
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	// Firmware & OS
	BIOSVersion    *string    `json:"bios_version,omitempty" db:"bios_version"`
	FirmwareVersion *string   `json:"firmware_version,omitempty" db:"firmware_version"`
	OSName         *string    `json:"os_name,omitempty" db:"os_name"`
	OSVersion      *string    `json:"os_version,omitempty" db:"os_version"`
	OSLicenseKey   *string    `json:"os_license_key,omitempty" db:"os_license_key"`
	// Physical
	FormFactor     *string    `json:"form_factor,omitempty" db:"form_factor"`
	Color          *string    `json:"color,omitempty" db:"color"`
	WeightKG       *float64   `json:"weight_kg,omitempty" db:"weight_kg"`
	// Audit
	LastScannedAt  *time.Time `json:"last_scanned_at,omitempty" db:"last_scanned_at"`
	CreatedBy      *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	AssetName *string `json:"asset_name,omitempty"`
	AssetTag  *string `json:"asset_tag,omitempty"`
}

// SoftwareUsageLog mencatat sesi pemakaian lisensi per user/device.
// Digunakan untuk SAM reconciliation (ISO 19770-1).
type SoftwareUsageLog struct {
	ID           int64      `json:"id" db:"id"`
	LicenseID    int64      `json:"license_id" db:"license_id"`
	AssetID      *int64     `json:"asset_id,omitempty" db:"asset_id"`
	EmployeeID   *int64     `json:"employee_id,omitempty" db:"employee_id"`
	SessionStart time.Time  `json:"session_start" db:"session_start"`
	SessionEnd   *time.Time `json:"session_end,omitempty" db:"session_end"`
	UsageMinutes *int       `json:"usage_minutes,omitempty" db:"usage_minutes"`
	Source       string     `json:"source" db:"source"` // manual|agent|import|sccm|jamf
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`

	// Transient
	LicenseName  *string `json:"license_name,omitempty"`
	AssetName    *string `json:"asset_name,omitempty"`
	EmployeeName *string `json:"employee_name,omitempty"`
}

// LicenseReconciliation adalah hasil view v_license_reconciliation.
// Menampilkan delta antara entitlement vs aktual instalasi.
type LicenseReconciliation struct {
	LicenseID             int64      `json:"license_id"`
	LicenseName           string     `json:"license_name"`
	LicenseType           *string    `json:"license_type,omitempty"`
	LicenseModel          *string    `json:"license_model,omitempty"`
	EntitledSeats         int        `json:"entitled_seats"`
	InstalledSeats        int        `json:"installed_seats"`
	AvailableSeats        int        `json:"available_seats"`
	ReconciliationStatus  string     `json:"reconciliation_status"` // compliant|over_licensed|under_utilized
	ExpirationDate        *time.Time `json:"expiration_date,omitempty"`
	ComplianceStatus      *string    `json:"compliance_status,omitempty"`
	Vendor                *string    `json:"vendor,omitempty"`
	Cost                  *float64   `json:"cost,omitempty"`
	Currency              *string    `json:"currency,omitempty"`
	ActiveUsers90d        int        `json:"active_users_90d"`
	LastUsedAt            *time.Time `json:"last_used_at,omitempty"`
}

// AssetDisposalRecord mencatat proses disposal aset termasuk kepatuhan lingkungan.
// Sesuai regulasi e-waste (RoHS/WEEE) dan ISO 19770-10.
type AssetDisposalRecord struct {
	ID                     int64      `json:"id" db:"id"`
	AssetID                int64      `json:"asset_id" db:"asset_id"`
	DisposalMethod         string     `json:"disposal_method" db:"disposal_method"`
	DataWipeMethod         *string    `json:"data_wipe_method,omitempty" db:"data_wipe_method"`
	DataWipeCompleted      bool       `json:"data_wipe_completed" db:"data_wipe_completed"`
	CertificateNumber      *string    `json:"certificate_number,omitempty" db:"certificate_number"`
	CertificateURL         *string    `json:"certificate_url,omitempty" db:"certificate_url"`
	EnvironmentalCompliant bool       `json:"environmental_compliant" db:"environmental_compliant"`
	RegulatoryNotes        *string    `json:"regulatory_notes,omitempty" db:"regulatory_notes"`
	Vendor                 *string    `json:"vendor,omitempty" db:"vendor"`
	DisposalValue          *float64   `json:"disposal_value,omitempty" db:"disposal_value"`
	AuthorizationBy        int64      `json:"authorization_by" db:"authorization_by"`
	ExecutedBy             *int64     `json:"executed_by,omitempty" db:"executed_by"`
	DateDisposed           time.Time  `json:"date_disposed" db:"date_disposed"`
	CreatedBy              *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`

	// Transient
	AssetName         *string `json:"asset_name,omitempty"`
	AssetTag          *string `json:"asset_tag,omitempty"`
	AuthorizedByName  *string `json:"authorized_by_name,omitempty"`
	ExecutedByName    *string `json:"executed_by_name,omitempty"`
}
