// File: backend/handlers/websocket_handler.go
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/auth"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-gonic/gin"
	gws "github.com/gorilla/websocket"
)

// ============================================================
// ⚙️ Konfigurasi upgrader (terima semua origin untuk internal use)
// ============================================================
var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ============================================================
// 🔹 WebSocketHandler — entry utama untuk koneksi realtime
// ============================================================
func WebSocketHandler(c *gin.Context) {
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		log.Println("[WS] Invalid token:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("[WS] Upgrade error:", err)
		return
	}

	client := &websocket.Client{
		Conn:   conn,
		UserID: claims.UserID,
		Role:   claims.Role,
	}

	hub := websocket.GetHub()
	hub.RegisterClient(client)

	// heartbeat & graceful disconnect
	go func() {
		defer func() {
			hub.UnregisterClient(client)
			conn.Close()
			log.Printf("[WS] User %d disconnected", claims.UserID)
		}()
		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
	log.Printf("[WS] ✅ User %d (%s) connected", claims.UserID, claims.Role)
}

// ============================================================
// 📦 Struktur pesan realtime standar
// ============================================================
type WSMessage struct {
	Type      string      `json:"type"`      // alert | audit | ticket | asset | custom
	Action    string      `json:"action"`    // CREATE | UPDATE | DELETE | ACK | NOTICE
	Data      interface{} `json:"data"`      // payload fleksibel
	Timestamp string      `json:"timestamp"` // RFC3339
}

// ============================================================
// 🔹 Fungsi pembantu untuk broadcast realtime event
// ============================================================

// BroadcastJSON — backward compatible dengan versi lama
func BroadcastJSON(entity, action, detail string) {
	msg := WSMessage{
		Type:      entity,
		Action:    action,
		Data:      map[string]string{"message": detail},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(msg)
	websocket.BroadcastToAll(string(data))
}

// BroadcastAlert — kirim alert langsung ke semua klien
func BroadcastAlert(message string, severity string) {
	payload := WSMessage{
		Type:      "alert",
		Action:    strings.ToUpper(severity),
		Data:      map[string]string{"message": message, "severity": severity},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(payload)
	websocket.BroadcastToAll(string(data))
	log.Printf("[WS_ALERT] %s", message)
}

// BroadcastAudit — kirim audit log baru ke dashboard realtime
func BroadcastAudit(entity string, action string, actor string) {
	payload := WSMessage{
		Type:   "audit",
		Action: action,
		Data: map[string]string{
			"entity": entity,
			"actor":  actor,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(payload)
	websocket.BroadcastToAll(string(data))
	log.Printf("[WS_AUDIT] %s %s by %s", entity, action, actor)
}
