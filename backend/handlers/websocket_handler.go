// File: backend/handlers/websocket_handler.go
package handlers

import (
	"log"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/auth"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-gonic/gin"
	gws "github.com/gorilla/websocket"
)

var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func WebSocketHandler(c *gin.Context) {
	// 1. Ambil token dari query parameter URL
	tokenString := c.Query("token")

	// 2. Gunakan fungsi validasi terpusat yang sudah kita buat
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		log.Println("WebSocket connection rejected: invalid token.", err)
		return
	}

	// 3. Upgrade koneksi HTTP menjadi WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// 4. Buat dan daftarkan klien baru ke Hub
	client := &websocket.Client{
		Conn:   conn,
		UserID: claims.UserID,
	}
	hub := websocket.GetHub()
	hub.RegisterClient(client)

	// 5. Jalankan goroutine untuk mendeteksi koneksi terputus
	go func() {
		defer func() {
			hub.UnregisterClient(client)
			conn.Close()
		}()
		for {
			// ReadMessage digunakan hanya untuk mendeteksi penutupan koneksi oleh klien
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}
