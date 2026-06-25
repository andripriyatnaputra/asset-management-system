package models

import "time"

// WebhookSubscription mendefinisikan endpoint eksternal yang menerima event ITAM.
type WebhookSubscription struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	URL       string    `json:"url" db:"url"`
	Events    []string  `json:"events" db:"events"` // e.g. ["ticket.created","asset.assigned"]
	Secret    *string   `json:"secret,omitempty" db:"secret"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedBy *int64    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Transient
	CreatedByName *string `json:"created_by_name,omitempty"`
}

// WebhookDeliveryLog menyimpan hasil pengiriman payload ke subscriber.
type WebhookDeliveryLog struct {
	ID             int64      `json:"id" db:"id"`
	SubscriptionID int64      `json:"subscription_id" db:"subscription_id"`
	EventType      string     `json:"event_type" db:"event_type"`
	Payload        string     `json:"payload" db:"payload"` // JSONB stored as string
	Status         string     `json:"status" db:"status"`   // pending|delivered|failed
	ResponseCode   *int       `json:"response_code,omitempty" db:"response_code"`
	ResponseBody   *string    `json:"response_body,omitempty" db:"response_body"`
	AttemptCount   int        `json:"attempt_count" db:"attempt_count"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty" db:"last_attempt_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`

	// Transient
	WebhookName *string `json:"webhook_name,omitempty"`
}

// AssetQRCode menyimpan data QR/barcode yang dicetak untuk aset.
type AssetQRCode struct {
	ID        int64      `json:"id" db:"id"`
	AssetID   int64      `json:"asset_id" db:"asset_id"`
	QRData    string     `json:"qr_data" db:"qr_data"`   // URL atau identifier
	Format    string     `json:"format" db:"format"`     // qr|barcode|datamatrix
	LabelData *string    `json:"label_data,omitempty" db:"label_data"` // JSONB
	PrintedAt *time.Time `json:"printed_at,omitempty" db:"printed_at"`
	PrintedBy *int64     `json:"printed_by,omitempty" db:"printed_by"`
	CreatedBy *int64     `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`

	// Transient
	AssetName     *string `json:"asset_name,omitempty"`
	AssetTag      *string `json:"asset_tag,omitempty"`
	PrintedByName *string `json:"printed_by_name,omitempty"`
}

// LDAPSyncConfig menyimpan konfigurasi koneksi AD/LDAP.
type LDAPSyncConfig struct {
	ID           int64     `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Host         string    `json:"host" db:"host"`
	Port         int       `json:"port" db:"port"`
	UseTLS       bool      `json:"use_tls" db:"use_tls"`
	BaseDN       string    `json:"base_dn" db:"base_dn"`
	BindDN       string    `json:"bind_dn" db:"bind_dn"`
	BindPassword *string   `json:"bind_password,omitempty" db:"bind_password"`
	UserFilter   string    `json:"user_filter" db:"user_filter"`
	FieldMap     string    `json:"field_map" db:"field_map"` // JSONB: {"sAMAccountName":"username",...}
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// LDAPSyncLog mencatat hasil sinkronisasi AD/LDAP.
type LDAPSyncLog struct {
	ID           int64      `json:"id" db:"id"`
	ConfigID     int64      `json:"config_id" db:"config_id"`
	Status       string     `json:"status" db:"status"` // running|success|partial|failed
	UsersFound   int        `json:"users_found" db:"users_found"`
	UsersSynced  int        `json:"users_synced" db:"users_synced"`
	UsersSkipped int        `json:"users_skipped" db:"users_skipped"`
	Errors       string     `json:"errors" db:"errors"` // JSONB array
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty" db:"finished_at"`
	TriggeredBy  *int64     `json:"triggered_by,omitempty" db:"triggered_by"`

	// Transient
	ConfigName      *string `json:"config_name,omitempty"`
	TriggeredByName *string `json:"triggered_by_name,omitempty"`
}
