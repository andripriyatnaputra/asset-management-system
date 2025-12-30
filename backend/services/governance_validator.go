package services

import (
	"strings"

	"github.com/andripriyatnaputra/asset-management-system/backend/models"
)

// ============================================================
// ✅ ValidateAssetGovernance
// ============================================================
// Memeriksa kelengkapan governance linkage untuk tiap aset
// Sesuai ISO/IEC 19770-10:2025 — Governance & Lifecycle Compliance
// ============================================================
func ValidateAssetGovernance(asset models.Asset) (bool, string) {
	missing := []string{}

	if asset.BudgetID == nil {
		missing = append(missing, "missing budget linkage")
	}
	if asset.ContractID == nil {
		missing = append(missing, "missing contract linkage")
	}
	if asset.LicenseID == nil {
		missing = append(missing, "missing license linkage")
	}
	if asset.LifecycleStage == nil || *asset.LifecycleStage == "" {
		missing = append(missing, "missing lifecycle stage")
	}
	if asset.AssetCriticality == nil {
		missing = append(missing, "missing asset criticality")
	}

	if len(missing) == 0 {
		return true, "compliant"
	}
	return false, strings.Join(missing, ", ")
}
