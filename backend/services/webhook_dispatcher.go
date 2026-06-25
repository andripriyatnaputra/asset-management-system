package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

// DispatchWebhook mengirim event ke semua webhook subscription yang aktif dan cocok.
// Dipanggil fire-and-forget dari handler.
func DispatchWebhook(eventType string, payload map[string]interface{}) {
	go dispatchAsync(eventType, payload)
}

func dispatchAsync(eventType string, payload map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Tambah metadata ke payload
	payload["event"] = eventType
	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Webhook] gagal marshal payload: %v", err)
		return
	}

	// Query subscriptions aktif yang subscribe ke event ini
	rows, err := database.Pool.Query(ctx, `
		SELECT id, url, secret
		FROM webhook_subscriptions
		WHERE is_active = true
		  AND $1 = ANY(events)
	`, eventType)
	if err != nil {
		log.Printf("[Webhook] gagal query subscriptions: %v", err)
		return
	}
	defer rows.Close()

	type sub struct {
		ID     int64
		URL    string
		Secret *string
	}
	var subs []sub
	for rows.Next() {
		var s sub
		if err := rows.Scan(&s.ID, &s.URL, &s.Secret); err == nil {
			subs = append(subs, s)
		}
	}
	rows.Close()

	for _, s := range subs {
		go deliverWebhook(ctx, s.ID, s.URL, s.Secret, eventType, body)
	}
}

func deliverWebhook(ctx context.Context, subID int64, url string, secret *string, eventType string, body []byte) {
	// Buat delivery log entry
	var logID int64
	now := time.Now()
	_ = database.Pool.QueryRow(ctx, `
		INSERT INTO webhook_delivery_logs (subscription_id, event_type, payload, status, attempt_count, last_attempt_at)
		VALUES ($1, $2, $3, 'pending', 1, $4)
		RETURNING id
	`, subID, eventType, string(body), now).Scan(&logID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		updateDeliveryLog(logID, "failed", nil, fmt.Sprintf("request error: %v", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ITAM-Event", eventType)

	// HMAC-SHA256 signature jika secret ada
	if secret != nil && *secret != "" {
		mac := hmac.New(sha256.New, []byte(*secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-ITAM-Signature", "sha256="+sig)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		updateDeliveryLog(logID, "failed", nil, fmt.Sprintf("delivery error: %v", err))
		return
	}
	defer resp.Body.Close()

	status := "delivered"
	if resp.StatusCode >= 400 {
		status = "failed"
	}
	code := resp.StatusCode
	updateDeliveryLog(logID, status, &code, fmt.Sprintf("HTTP %d", resp.StatusCode))
}

func updateDeliveryLog(logID int64, status string, code *int, body string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = database.Pool.Exec(ctx, `
		UPDATE webhook_delivery_logs
		SET status = $1, response_code = $2, response_body = $3
		WHERE id = $4
	`, status, code, body, logID)
}
