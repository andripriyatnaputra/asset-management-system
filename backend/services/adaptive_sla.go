// File: backend/services/adaptive_sla.go
package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// AdaptiveSLAEngine memperbarui SLA Policy secara dinamis berdasarkan kondisi sistem
func AdaptiveSLAEngine() {
	log.Println("[SLA] 🕒 Adaptive SLA Engine started")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		adjustSLAPolicies()
	}
}

// =============================================================
// 🔧 Inti Penyesuaian SLA
// =============================================================
func adjustSLAPolicies() {
	ctx := context.Background()

	query := `
		WITH stats AS (
			SELECT 
				sp.id AS policy_id,
				sp.service_code,
				sp.response_minutes,
				sp.resolve_minutes,
				COALESCE(AVG(a.asset_health_score), 100) AS avg_health,
				COUNT(t.id) FILTER (WHERE t.status NOT IN ('Resolved','Closed')) AS open_tickets,
				COUNT(al.id) FILTER (WHERE al.acknowledged = false) AS active_alerts
			FROM sla_policies sp
			LEFT JOIN tickets t ON t.service_code = sp.service_code
			LEFT JOIN assets a ON a.id = t.related_asset_id
			LEFT JOIN alerts al ON al.category = 'system'
			WHERE sp.is_active = TRUE
			GROUP BY sp.id, sp.service_code, sp.response_minutes, sp.resolve_minutes
		)
		UPDATE sla_policies sp
		SET 
			response_minutes = ROUND(LEAST(720, GREATEST(15, 
				s.response_minutes * 
				CASE 
					WHEN s.avg_health < 50 THEN 0.85
					WHEN s.avg_health > 90 THEN 1.10
					ELSE 1.00
				END
			))),
			resolve_minutes = ROUND(LEAST(1440, GREATEST(30, 
				s.resolve_minutes * 
				CASE 
					WHEN s.open_tickets > 10 THEN 1.20
					WHEN s.active_alerts > 5 THEN 1.15
					ELSE 1.00
				END
			))),
			compliance_score = ROUND(100 - (s.active_alerts * 0.5 + s.open_tickets * 0.8))
		FROM stats s
		WHERE sp.id = s.policy_id;
	`

	_, err := database.Pool.Exec(ctx, query)
	if err != nil {
		log.Printf("[SLA] ❌ Adaptive SLA update failed: %v", err)
		return
	}

	log.Println("[SLA] ✅ Adaptive SLA recalibration complete")

	// Catat audit log sistem
	change := map[string]interface{}{
		"source":    "adaptive_engine",
		"timestamp": time.Now().Format(time.RFC3339),
		"note":      "Adaptive SLA recalibration executed successfully",
	}
	changeJSON, _ := json.Marshal(change)
	_, _ = database.Pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, entity_name, action, changes, created_at, user_agent)
		VALUES (NULL, 'sla_policies', 'AUTO_ADAPTIVE', $1, now(), 'system-job')
	`, changeJSON)

	// Kirim notifikasi WebSocket
	BroadcastAlert("Adaptive SLA Engine telah melakukan penyesuaian otomatis kebijakan SLA aktif.", "info")
}
