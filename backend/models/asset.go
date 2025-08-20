package models

import "time"

type Asset struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	AssetTag      string     `json:"asset_tag"`
	Status        string     `json:"status"`
	AssetTypeID   *int64     `json:"asset_type_id"`
	AssetTypeName *string    `json:"asset_type_name,omitempty"`
	PurchaseDate  time.Time  `json:"purchase_date,omitempty"`
	InitialPrice  float64    `json:"initial_price,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"-"`
}
