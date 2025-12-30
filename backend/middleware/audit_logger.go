package middleware

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-gonic/gin"
)

// LogAction mencatat aktivitas CRUD ke tabel audit_logs.
// Kolom hash/prev_hash otomatis diisi oleh trigger di PostgreSQL.

// =============================================================
// 🔹 LEVEL 1: REQUEST-LEVEL AUDIT (stdout logging)
// =============================================================

// AuditLogger mencatat semua request API ke stdout (bukan ke DB).
// Dipasang pada level group /api/v1 agar context user sudah tersedia.
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		path := c.Request.URL.Path

		// 🚫 Skip swagger, health, auth, OPTIONS, dan file statis
		if c.Request.Method == "OPTIONS" ||
			strings.HasPrefix(path, "/swagger") ||
			strings.HasPrefix(path, "/api/v1/health") ||
			strings.HasPrefix(path, "/static") {
			return
		}

		// 🔹 Default action (GET / POST / PUT / DELETE)
		actionName := strings.ToUpper(c.Request.Method)

		// 🔹 Override untuk endpoint auth
		if p := c.FullPath(); p != "" {
			switch {
			case strings.Contains(p, "/auth/login"):
				actionName = "LOGIN"
			case strings.Contains(p, "/auth/logout"):
				actionName = "LOGOUT"
			case strings.Contains(p, "/auth/refresh"):
				actionName = "REFRESH_TOKEN"
			}
		}

		// 🔹 Ambil role dan user_id dari context
		roleVal, _ := c.Get("role")
		userVal, _ := c.Get("user_id")

		var actorID *int64
		role := "<unauthenticated>"

		if r, ok := roleVal.(string); ok && r != "" {
			role = r
		}

		switch v := userVal.(type) {
		case int:
			tmp := int64(v)
			actorID = &tmp
		case int64:
			actorID = &v
		case float64:
			tmp := int64(v)
			actorID = &tmp
		case string:
			// jika bukan angka, biarkan NULL (anonim)
		}

		// 🔹 Tulis log ke console
		log.Printf("[AUDIT] action=%s method=%s path=%s role=%s user_id=%v status=%d duration=%v",
			actionName,
			c.Request.Method,
			path,
			role,
			func() interface{} {
				if actorID != nil {
					return *actorID
				}
				return "<none>"
			}(),
			c.Writer.Status(),
			duration,
		)

		// 🔹 Non-blocking insert ke database
		go func(actor *int64, action, path string) {
			//if !strings.HasPrefix(path, "/api/v1/") || strings.Contains(path, "/auth") {
			//	return
			//}

			_, err := database.Pool.Exec(context.Background(), `
				INSERT INTO audit_logs (actor_id, entity_name, action, created_at, request_path)
				VALUES ($1, $2, $3, NOW(), $4)
			`, actor, "system_audit", action, path)

			if err != nil {
				log.Printf("[AUDIT_LOG_ERROR] %v", err)
			}
		}(actorID, actionName, path)
	}
}

// =============================================================
// 🔹 LEVEL 2: USER-LEVEL AUDIT (DB persistence)
// =============================================================

// LogAction mencatat aktivitas CRUD nyata ke tabel audit_logs (DB).
// Dipanggil langsung dari handlers yang memodifikasi data.
func LogAction(c *gin.Context, entity string, entityID int64, action string, changes any) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[AUDIT_LOG_ERROR] panic recovered: %v", r)
		}
	}()

	var actorID *int64

	// ✅ Ambil user_id dari context (kalau sudah login)
	if uid, ok := c.Get("user_id"); ok {
		switch v := uid.(type) {
		case int64:
			actorID = &v
		case int:
			tmp := int64(v)
			actorID = &tmp
		case float64:
			tmp := int64(v)
			actorID = &tmp
		}
	}

	// ✅ Jika belum ada (misalnya saat login pertama kali)
	if actorID == nil && entity == "employees" && action == "LOGIN" {
		actorID = &entityID
	}

	changeJSON := []byte("{}")
	if changes != nil {
		if b, err := json.Marshal(changes); err == nil {
			changeJSON = b
		}
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	reqPath := c.Request.URL.Path

	_, err := database.Pool.Exec(
		c.Request.Context(),
		`INSERT INTO audit_logs (actor_id, entity_name, entity_id, action, changes, ip_address, user_agent, request_path, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now())`,
		actorID, entity, entityID, action, changeJSON, ip, userAgent, reqPath,
	)
	if err != nil {
		log.Printf("[AUDIT_LOG_ERROR] %v", err)
		return
	}

	// ✅ realtime broadcast
	payload := map[string]any{
		"type":      "audit",
		"entity":    entity,
		"action":    action,
		"entity_id": entityID,
		"time":      time.Now().Format(time.RFC3339),
	}
	jsonData, _ := json.Marshal(payload)
	if entity != "audit_logs" { // hindari loop broadcast diri sendiri
		websocket.BroadcastToAll(string(jsonData))
	}
}
