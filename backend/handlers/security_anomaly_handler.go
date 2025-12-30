package handlers

import (
	"math"
	"net/http"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/gin-gonic/gin"
)

// GET /api/v1/audit/anomalies
func GetSecurityAnomalies(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT actor_id, action, COUNT(*) AS total,
		       DATE(created_at) AS date
		  FROM v_security_audit
		 WHERE created_at > NOW() - INTERVAL '30 days'
		 GROUP BY actor_id, action, DATE(created_at)
		 ORDER BY DATE(created_at)
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Entry struct {
		ActorID *int64  `json:"actor_id"`
		Action  string  `json:"action"`
		Total   int     `json:"total"`
		Date    string  `json:"date"`
		Score   float64 `json:"score"`
	}

	var list []Entry
	userActivity := map[int64][]int{}

	for rows.Next() {
		var e Entry
		rows.Scan(&e.ActorID, &e.Action, &e.Total, &e.Date)
		if e.ActorID != nil {
			userActivity[*e.ActorID] = append(userActivity[*e.ActorID], e.Total)
		}
		list = append(list, e)
	}

	// compute anomaly score (z-score)
	for i := range list {
		if list[i].ActorID != nil {
			values := userActivity[*list[i].ActorID]
			if len(values) > 1 {
				mean, std := meanStd(values)
				list[i].Score = math.Abs(float64(list[i].Total)-mean) / std
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func meanStd(xs []int) (float64, float64) {
	var sum float64
	for _, x := range xs {
		sum += float64(x)
	}
	mean := sum / float64(len(xs))
	var variance float64
	for _, x := range xs {
		diff := float64(x) - mean
		variance += diff * diff
	}
	std := math.Sqrt(variance / float64(len(xs)))
	if std == 0 {
		std = 1
	}
	return mean, std
}
