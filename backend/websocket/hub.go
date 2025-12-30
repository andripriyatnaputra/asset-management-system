package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// =============================================================
// 🧩 Struktur Client & Hub
// =============================================================

// Client mewakili 1 koneksi WebSocket aktif (satu pengguna).
type Client struct {
	Conn   *websocket.Conn
	UserID int64
	Role   string // 🔹 Tambahan: menyimpan role pengguna untuk broadcast selektif
}

// Hub mengelola semua koneksi WebSocket aktif dan broadcast pesan global.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte // 🔥 buffer untuk broadcast global
	mu         sync.Mutex
}

var hubInstance *Hub
var once sync.Once

// =============================================================
// 🧩 Singleton Getter
// =============================================================

// GetHub mengembalikan instance tunggal Hub (singleton pattern)
func GetHub() *Hub {
	once.Do(func() {
		hubInstance = &Hub{
			clients:    make(map[*Client]bool),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			broadcast:  make(chan []byte, 50), // buffer untuk antisipasi burst alert
		}
		go hubInstance.Run()
	})
	return hubInstance
}

// =============================================================
// 🧩 Core Loop — event utama untuk mengatur koneksi dan broadcast
// =============================================================

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			log.Printf("🟢 WebSocket connected: user=%d role=%s", client.UserID, client.Role)
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Conn.Close()
				log.Printf("🔴 WebSocket disconnected: user=%d role=%s", client.UserID, client.Role)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.Conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("⚠️ Broadcast failed to user=%d (%s): %v", client.UserID, client.Role, err)
					client.Conn.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// =============================================================
// 🧩 Kirim Pesan ke Pengguna Tertentu (mis. SLA breach owner)
// =============================================================

func (h *Hub) SendToUser(userID int64, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		if client.UserID == userID {
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Error sending to user %d: %v", userID, err)
				client.Conn.Close()
				delete(h.clients, client)
			}
		}
	}
}

// =============================================================
// 🧩 Kirim Pesan ke Role Tertentu (opsional: mis. hanya manager)
// =============================================================

func (h *Hub) SendToRole(role string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		if client.Role == role {
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("❌ Error sending to role %s: %v", role, err)
				client.Conn.Close()
				delete(h.clients, client)
			}
		}
	}
}

// =============================================================
// 🧩 Broadcast ke Semua Client Aktif
// =============================================================

func BroadcastToAll(message string) {
	hub := GetHub()
	select {
	case hub.broadcast <- []byte(message):
		log.Printf("📢 Broadcast sent: %s", message)
	default:
		log.Printf("⚠️ Broadcast dropped (channel full): %s", message)
	}
}

// =============================================================
// 🧩 Register / Unregister Client (digunakan dari handler)
// =============================================================

func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// =============================================================
// 🧩 BroadcastTicketComment — kirim event realtime komentar tiket
// =============================================================
func BroadcastTicketComment(ticketID int64, comment interface{}) {
	payload := gin.H{
		"event": fmt.Sprintf("ticket_comment:%d", ticketID),
		"data":  comment,
	}
	data, _ := json.Marshal(payload)

	hub := GetHub()
	select {
	case hub.broadcast <- data:
		log.Printf("💬 Realtime comment broadcasted for ticket #%d", ticketID)
	default:
		log.Printf("⚠️ Broadcast channel full, comment event dropped for ticket #%d", ticketID)
	}
}
