package handlers

import (
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/services"

	"github.com/gin-gonic/gin"
)

// POST /governance/review
func PostGovernanceReview(c *gin.Context) {
	var in services.ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Ambil reviewer ID dari context auth
	if uid, ok := c.Get("user_id"); ok {
		in.ReviewerID = uid.(int64)
	}

	if err := services.SaveFeedback(c.Request.Context(), in); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "review feedback saved"})
}
