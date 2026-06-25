package handlers

import (
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/gin-gonic/gin"
)

// GetMyNotifications godoc
// @Summary List notifikasi milik user yang login
// @Tags Notifications
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(20)
// @Param unread query bool false "Filter hanya yang belum dibaca"
// @Success 200 {object} pagedResponse
// @Security BearerAuth
// @Router /notifications [get]
func GetMyNotifications(c *gin.Context) {
	actor := getActorID(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	unreadOnly := c.Query("unread") == "true"
	pg := getPagination(c)

	where := "WHERE n.user_id = $1"
	args := []interface{}{*actor}
	idx := 2

	if unreadOnly {
		where += " AND n.is_read = false"
	}

	var total int
	_ = database.Pool.QueryRow(c, "SELECT COUNT(*) FROM notifications n "+where, args...).Scan(&total)

	rows, err := database.Pool.Query(c,
		"SELECT id, user_id, type, title, message, entity_type, entity_id, is_read, created_at "+
			"FROM notifications n "+where+
			" ORDER BY created_at DESC LIMIT $"+strconv.Itoa(idx)+" OFFSET $"+strconv.Itoa(idx+1),
		append(args, pg.Limit, pg.Offset)...,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message,
			&n.EntityType, &n.EntityID, &n.IsRead, &n.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.Notification{}
	}
	c.JSON(http.StatusOK, pagedResponse{Data: list, Total: total, Page: pg.Page, Limit: pg.Limit})
}

// GetUnreadCount godoc
// @Summary Jumlah notifikasi belum dibaca
// @Tags Notifications
// @Produce json
// @Success 200 {object} map[string]int
// @Security BearerAuth
// @Router /notifications/unread-count [get]
func GetUnreadCount(c *gin.Context) {
	actor := getActorID(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var count int
	_ = database.Pool.QueryRow(c,
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false", *actor,
	).Scan(&count)
	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// MarkNotificationRead godoc
// @Summary Tandai satu notifikasi sebagai sudah dibaca
// @Tags Notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /notifications/{id}/read [put]
func MarkNotificationRead(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	_, err := database.Pool.Exec(c,
		"UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2", id, *actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notifikasi ditandai sudah dibaca"})
}

// MarkAllRead godoc
// @Summary Tandai semua notifikasi sebagai sudah dibaca
// @Tags Notifications
// @Produce json
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /notifications/read-all [put]
func MarkAllRead(c *gin.Context) {
	actor := getActorID(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	_, err := database.Pool.Exec(c,
		"UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false", *actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "semua notifikasi ditandai sudah dibaca"})
}

// DeleteNotification godoc
// @Summary Hapus notifikasi milik user sendiri
// @Tags Notifications
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /notifications/{id} [delete]
func DeleteNotification(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	actor := getActorID(c)
	if actor == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	_, err := database.Pool.Exec(c,
		"DELETE FROM notifications WHERE id = $1 AND user_id = $2", id, *actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notifikasi dihapus"})
}
