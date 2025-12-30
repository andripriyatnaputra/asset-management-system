package models

type AssetStatus string

const (
	StatusInStock     AssetStatus = "in_stock"
	StatusAssigned    AssetStatus = "assigned"
	StatusMaintenance AssetStatus = "maintenance"
	StatusRetired     AssetStatus = "retired"
	StatusDisposed    AssetStatus = "disposed"
)
