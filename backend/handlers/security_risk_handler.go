package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// GET /api/v1/security/risk-insight
func GetSecurityRiskInsight(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT actor_id, action, COUNT(*) AS total,
		       MAX(created_at) AS last_activity
		  FROM v_security_audit
		 WHERE created_at > NOW() - INTERVAL '30 days'
		 GROUP BY actor_id, action
		 ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Entry struct {
		ActorID      *int64    `json:"actor_id"`
		Action       string    `json:"action"`
		Total        int       `json:"total"`
		LastActivity time.Time `json:"last_activity"`
	}

	var list []Entry
	for rows.Next() {
		var e Entry
		_ = rows.Scan(&e.ActorID, &e.Action, &e.Total, &e.LastActivity)
		list = append(list, e)
	}

	// Analisis risiko sederhana (bobot berbasis aksi)
	type Risk struct {
		ActorID  *int64  `json:"actor_id"`
		Action   string  `json:"action"`
		Score    float64 `json:"score"`
		Severity string  `json:"severity"`
		Message  string  `json:"message"`
	}

	var risks []Risk
	for _, r := range list {
		score := float64(r.Total)
		switch {
		case r.Action == "LOGIN" && score > 20:
			risks = append(risks, Risk{
				ActorID:  r.ActorID,
				Action:   r.Action,
				Score:    score,
				Severity: "high",
				Message:  "Login frekuensi sangat tinggi — potensi serangan brute force.",
			})
		case r.Action == "REFRESH_TOKEN" && score > 50:
			risks = append(risks, Risk{
				ActorID:  r.ActorID,
				Action:   r.Action,
				Score:    score,
				Severity: "medium",
				Message:  "Refresh token berlebihan — periksa durasi token & rotasi.",
			})
		case r.Action == "DELEGATION_CREATE":
			risks = append(risks, Risk{
				ActorID:  r.ActorID,
				Action:   r.Action,
				Score:    score,
				Severity: "low",
				Message:  "Delegasi role aktif — pastikan periode dan otorisasi sesuai kebijakan.",
			})
		default:
			if score > 100 {
				risks = append(risks, Risk{
					ActorID:  r.ActorID,
					Action:   r.Action,
					Score:    score,
					Severity: "low",
					Message:  "Aktivitas tinggi terdeteksi.",
				})
			}
		}
	}

	// Normalisasi skor & rank
	var total float64
	for _, r := range risks {
		total += r.Score
	}
	for i := range risks {
		if total > 0 {
			risks[i].Score = math.Round((risks[i].Score / total) * 100)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"generated_at":    time.Now(),
		"recommendations": risks,
	})
}
