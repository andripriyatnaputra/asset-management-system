package handlers

import (
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/security"
	"github.com/gin-gonic/gin"
)

// POST /internal/revoke
func RevokeTokenHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// revoke for 24 hours
	security.RevokeToken(req.Token, 24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"revoked":   true,
		"expiresIn": "24h",
	})
}
