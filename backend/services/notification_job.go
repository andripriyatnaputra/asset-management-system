package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
)

// RunNotificationJobs menjalankan semua job notifikasi secara periodik.
func RunNotificationJobs(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Jalankan sekali saat startup
	runAllNotifJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAllNotifJobs(ctx)
		}
	}
}

func runAllNotifJobs(ctx context.Context) {
	notifyExpiringLicenses(ctx)
	notifyDRTestsDue(ctx)
	notifyExpiredEvidence(ctx)
}

// CreateNotification menyimpan notifikasi ke DB lalu push realtime ke user via WebSocket.
func CreateNotification(ctx context.Context, userID int64, notifType, title, message, entityType string, entityID *int64) {
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, message, entity_type, entity_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, notifType, title, message, entityType, entityID)
	if err != nil {
		log.Printf("[Notif] gagal insert notifikasi: %v", err)
		return
	}

	// Push realtime ke user via WebSocket (best-effort, non-blocking)
	payload, _ := json.Marshal(map[string]interface{}{
		"type": "notification",
		"data": map[string]interface{}{
			"type":        notifType,
			"title":       title,
			"message":     message,
			"entity_type": entityType,
			"entity_id":   entityID,
		},
	})
	websocket.GetHub().SendToUser(userID, payload)
}

// notifyExpiringLicenses: kirim notif ke asset_manager & admin untuk lisensi yang
// akan expired dalam 30 hari dan belum dinotifkan hari ini.
func notifyExpiringLicenses(ctx context.Context) {
	rows, err := database.Pool.Query(ctx, `
		SELECT l.id, l.name, l.expiration_date
		FROM licenses l
		WHERE l.deleted_at IS NULL
		  AND l.expiration_date BETWEEN now() AND now() + INTERVAL '30 days'
		  AND NOT EXISTS (
		      SELECT 1 FROM notifications n
		      WHERE n.entity_type = 'license' AND n.entity_id = l.id
		        AND n.type = 'license_expiry'
		        AND n.created_at >= now() - INTERVAL '1 day'
		  )
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	// Ambil semua user dengan role asset_manager & super_admin
	type manager struct{ ID int64 }
	mgrRows, err := database.Pool.Query(ctx, `
		SELECT id FROM employees WHERE role IN ('super_admin','asset_manager') AND deleted_at IS NULL
	`)
	if err != nil {
		return
	}
	defer mgrRows.Close()

	var managers []int64
	for mgrRows.Next() {
		var id int64
		if err := mgrRows.Scan(&id); err == nil {
			managers = append(managers, id)
		}
	}
	mgrRows.Close()

	for rows.Next() {
		var licID int64
		var name string
		var expDate time.Time
		if err := rows.Scan(&licID, &name, &expDate); err != nil {
			continue
		}
		daysLeft := int(time.Until(expDate).Hours() / 24)
		title := fmt.Sprintf("Lisensi akan expired: %s", name)
		msg := fmt.Sprintf("Lisensi '%s' akan expired dalam %d hari (%s). Harap perbarui segera.",
			name, daysLeft, expDate.Format("02 Jan 2006"))

		id := licID
		for _, mgrID := range managers {
			CreateNotification(ctx, mgrID, "license_expiry", title, msg, "license", &id)
		}
	}
}

// notifyDRTestsDue: notif ke owner DR plan jika test sudah overdue atau jatuh tempo 7 hari.
func notifyDRTestsDue(ctx context.Context) {
	rows, err := database.Pool.Query(ctx, `
		SELECT p.id, p.name, p.owner_id, p.next_test_due
		FROM dr_plans p
		WHERE p.status = 'active'
		  AND p.next_test_due IS NOT NULL
		  AND p.next_test_due <= now() + INTERVAL '7 days'
		  AND p.owner_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM notifications n
		      WHERE n.entity_type = 'dr_plan' AND n.entity_id = p.id
		        AND n.type = 'dr_test_due'
		        AND n.created_at >= now() - INTERVAL '1 day'
		  )
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var planID, ownerID int64
		var name string
		var dueDate time.Time
		if err := rows.Scan(&planID, &name, &ownerID, &dueDate); err != nil {
			continue
		}
		overdue := time.Now().After(dueDate)
		title := fmt.Sprintf("DR Test jatuh tempo: %s", name)
		var msg string
		if overdue {
			msg = fmt.Sprintf("DR/BCP plan '%s' melewati jadwal test (due: %s). Segera jadwalkan test.",
				name, dueDate.Format("02 Jan 2006"))
		} else {
			days := int(time.Until(dueDate).Hours() / 24)
			msg = fmt.Sprintf("DR/BCP plan '%s' dijadwalkan test dalam %d hari (%s).",
				name, days, dueDate.Format("02 Jan 2006"))
		}
		id := planID
		CreateNotification(ctx, ownerID, "dr_test_due", title, msg, "dr_plan", &id)
	}
}

// notifyExpiredEvidence: notif ke submitted_by jika evidence expired dan belum dinotifkan.
func notifyExpiredEvidence(ctx context.Context) {
	rows, err := database.Pool.Query(ctx, `
		SELECT ce.id, ce.title, ce.submitted_by
		FROM compliance_evidence ce
		WHERE ce.expires_at < now()
		  AND ce.status != 'expired'
		  AND ce.submitted_by IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM notifications n
		      WHERE n.entity_type = 'compliance_evidence' AND n.entity_id = ce.id
		        AND n.type = 'evidence_expired'
		        AND n.created_at >= now() - INTERVAL '1 day'
		  )
	`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var evID, submittedBy int64
		var title string
		if err := rows.Scan(&evID, &title, &submittedBy); err != nil {
			continue
		}
		// Auto-mark evidence as expired
		_, _ = database.Pool.Exec(ctx,
			"UPDATE compliance_evidence SET status='expired', updated_at=now() WHERE id=$1", evID)

		id := evID
		CreateNotification(ctx, submittedBy, "evidence_expired",
			"Evidence compliance kadaluarsa",
			fmt.Sprintf("Evidence '%s' telah kadaluarsa dan perlu diperbarui.", title),
			"compliance_evidence", &id)
	}
}
