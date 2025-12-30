package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// ============================================================
// 🤖 Auto-Remediation & Smart Alert Orchestration Engine
// ============================================================

func RunAutoRemediationWatcher(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Hour)
	defer ticker.Stop()

	log.Println("[WATCHER] Auto-Remediation engine started (interval 3h)")
	for {
		select {
		case <-ctx.Done():
			log.Println("[WATCHER] Stopping Auto-Remediation watcher...")
			return
		case <-ticker.C:
			log.Println("[WATCHER] Running auto-remediation check...")
			ExecuteAutoRemediation(ctx)
		}
	}
}

// ============================================================
// 🚨 ExecuteAutoRemediation — scan and handle risky assets
// ============================================================
func ExecuteAutoRemediation(ctx context.Context) {
	riskList, err := ComputeAssetRiskForecast(ctx)
	if err != nil {
		log.Printf("[AUTO_REMEDIATION_ERR] %v", err)
		return
	}

	for _, asset := range riskList {
		if asset.RiskIndex >= 80 {
			// Cek apakah sudah ada alert aktif
			var exists bool
			_ = database.Pool.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM alerts
					 WHERE message ILIKE $1 AND acknowledged=false
				)`,
				fmt.Sprintf("%%Asset #%d%%", asset.AssetID),
			).Scan(&exists)
			if exists {
				continue // sudah ada alert aktif
			}

			// 🔔 Buat alert baru
			msg := fmt.Sprintf("⚠️ Asset #%d (%s) critical risk: %.1f%% — %s",
				asset.AssetID, asset.AssetName, asset.RiskIndex, asset.PredictionNote)
			_, _ = database.Pool.Exec(ctx, `
				INSERT INTO alerts (message, severity, category, acknowledged, created_at)
				VALUES ($1,'critical','auto_remediation',false,NOW())
			`, msg)
			BroadcastAlert(msg, "critical")

			// 🎟️ Buat tiket otomatis
			subject := fmt.Sprintf("Auto-Remediation: %s", asset.AssetName)
			desc := fmt.Sprintf("System detected critical governance risk (%.1f). %s",
				asset.RiskIndex, asset.PredictionNote)
			_, _ = database.Pool.Exec(ctx, `
				INSERT INTO tickets (subject, description, status, priority, created_by_employee_id, related_asset_id, category_code)
				VALUES ($1,$2,'Open','High',1,$3,'AUTO-REMEDIATION')
			`, subject, desc, asset.AssetID)
			log.Printf("[AUTO_REMEDIATION] Ticket created for asset #%d", asset.AssetID)

			// 🚧 Tandai aset sedang diperiksa
			_, _ = database.Pool.Exec(ctx, `
				UPDATE assets SET lifecycle_stage='under_remediation', updated_at=NOW()
				 WHERE id=$1`, asset.AssetID)
		}
	}
	log.Println("[AUTO_REMEDIATION] cycle complete")
}
