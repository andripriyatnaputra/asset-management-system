// Package unit contains pure unit tests with no database dependency.
package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andripriyatnaputra/asset-management-system/backend/handlers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func notifRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/notifications", handlers.GetMyNotifications)
	r.GET("/notifications/unread-count", handlers.GetUnreadCount)
	r.PUT("/notifications/:id/read", handlers.MarkNotificationRead)
	r.PUT("/notifications/read-all", handlers.MarkAllRead)
	r.DELETE("/notifications/:id", handlers.DeleteNotification)
	return r
}

// All notification endpoints require a valid user_id in gin context.
// Without it, every endpoint must return 401 Unauthorized.

func TestGetMyNotifications_Unauthorized(t *testing.T) {
	r := notifRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "unauthorized", body["error"])
}

func TestGetUnreadCount_Unauthorized(t *testing.T) {
	r := notifRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMarkNotificationRead_Unauthorized(t *testing.T) {
	r := notifRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/42/read", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMarkAllRead_Unauthorized(t *testing.T) {
	r := notifRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/notifications/read-all", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteNotification_Unauthorized(t *testing.T) {
	r := notifRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/notifications/42", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
