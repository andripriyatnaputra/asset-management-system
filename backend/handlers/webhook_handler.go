package handlers

import (
	"net/http"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetWebhooks(c *gin.Context) {
	rows, err := database.Pool.Query(c, `
		SELECT w.id, w.name, w.url, w.events, w.secret, w.is_active,
		       w.created_by, w.created_at, w.updated_at,
		       e.name AS created_by_name
		FROM webhook_subscriptions w
		LEFT JOIN employees e ON e.id = w.created_by
		ORDER BY w.name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.WebhookSubscription
	for rows.Next() {
		var w models.WebhookSubscription
		if err := rows.Scan(&w.ID, &w.Name, &w.URL, &w.Events, &w.Secret, &w.IsActive,
			&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt, &w.CreatedByName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Hide secret in list view
		w.Secret = nil
		list = append(list, w)
	}
	if list == nil {
		list = []models.WebhookSubscription{}
	}
	c.JSON(http.StatusOK, list)
}

func CreateWebhook(c *gin.Context) {
	actor := getActorID(c)
	var req struct {
		Name   string   `json:"name" binding:"required"`
		URL    string   `json:"url" binding:"required"`
		Events []string `json:"events" binding:"required"`
		Secret *string  `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minimal 1 event diperlukan"})
		return
	}

	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO webhook_subscriptions (name, url, events, secret, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, req.Name, req.URL, req.Events, req.Secret, actor).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "webhook berhasil dibuat"})
}

func UpdateWebhook(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Name     *string  `json:"name"`
		URL      *string  `json:"url"`
		Events   []string `json:"events"`
		Secret   *string  `json:"secret"`
		IsActive *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := database.Pool.Exec(c, `
		UPDATE webhook_subscriptions SET
			name      = COALESCE($1, name),
			url       = COALESCE($2, url),
			events    = CASE WHEN $3::TEXT[] IS NOT NULL THEN $3 ELSE events END,
			secret    = COALESCE($4, secret),
			is_active = COALESCE($5, is_active),
			updated_at = now()
		WHERE id = $6
	`, req.Name, req.URL, req.Events, req.Secret, req.IsActive, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "webhook berhasil diupdate"})
}

func DeleteWebhook(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, err := database.Pool.Exec(c, `DELETE FROM webhook_subscriptions WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "webhook berhasil dihapus"})
}

func GetWebhookDeliveryLogs(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	rows, err := database.Pool.Query(c, `
		SELECT l.id, l.subscription_id, l.event_type, l.payload,
		       l.status, l.response_code, l.response_body,
		       l.attempt_count, l.last_attempt_at, l.created_at,
		       w.name AS webhook_name
		FROM webhook_delivery_logs l
		JOIN webhook_subscriptions w ON w.id = l.subscription_id
		WHERE l.subscription_id = $1
		ORDER BY l.created_at DESC
		LIMIT 200
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var list []models.WebhookDeliveryLog
	for rows.Next() {
		var l models.WebhookDeliveryLog
		if err := rows.Scan(&l.ID, &l.SubscriptionID, &l.EventType, &l.Payload,
			&l.Status, &l.ResponseCode, &l.ResponseBody,
			&l.AttemptCount, &l.LastAttemptAt, &l.CreatedAt,
			&l.WebhookName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		list = append(list, l)
	}
	if list == nil {
		list = []models.WebhookDeliveryLog{}
	}
	c.JSON(http.StatusOK, list)
}

func TestWebhook(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var w struct {
		URL    string `db:"url"`
		Events []string `db:"events"`
	}
	err := database.Pool.QueryRow(c, `SELECT url, events FROM webhook_subscriptions WHERE id = $1`, id).
		Scan(&w.URL, &w.Events)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "webhook tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.DispatchWebhook("webhook.test", map[string]interface{}{
		"webhook_id": id,
		"url":        w.URL,
		"message":    "test delivery dari ITAM system",
	})
	c.JSON(http.StatusOK, gin.H{"message": "test webhook dikirim (async)"})
}
