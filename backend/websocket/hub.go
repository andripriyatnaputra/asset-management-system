// File: backend/websocket/hub.go
package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn   *websocket.Conn
	UserID int64
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var hubInstance *Hub
var once sync.Once

func GetHub() *Hub {
	once.Do(func() {
		hubInstance = &Hub{
			clients:    make(map[*Client]bool),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
	})
	return hubInstance
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			log.Printf("Client connected for user ID: %d", client.UserID)
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// --- PERBAIKAN DI SINI ---
				// Panggil metode .Close() pada koneksi, bukan close()
				client.Conn.Close()
				log.Printf("Client disconnected for user ID: %d", client.UserID)
			}
			h.mu.Unlock()
		}
	}
}

// SendToUser mengirim pesan ke pengguna spesifik jika mereka online
func (h *Hub) SendToUser(userID int64, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if client.UserID == userID {
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error sending message to user %d: %v", userID, err)
			}
		}
	}
}

// RegisterClient mendaftarkan klien baru ke hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient menghapus klien dari hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
