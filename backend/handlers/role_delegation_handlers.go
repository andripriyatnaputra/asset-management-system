package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 🧾 CREATE ROLE DELEGATION (Governance + Validation)
// ============================================================
func CreateRoleDelegation(c *gin.Context) {
	role, _ := c.Get("role")
	userID, _ := c.Get("user_id")

	// 🛡️ Hanya super_admin dan manager yang boleh membuat delegasi
	if role != "super_admin" && role != "manager" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Tidak memiliki izin untuk membuat delegasi role.",
		})
		return
	}

	var body struct {
		DelegateeID  int64     `json:"delegatee_id" binding:"required"`
		RoleOverride string    `json:"role_override" binding:"required"`
		StartDate    time.Time `json:"start_date" binding:"required"`
		EndDate      time.Time `json:"end_date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !body.EndDate.After(body.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return
	}

	// 🧠 Pastikan tidak overlap
	var overlap bool
	_ = database.Pool.QueryRow(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM role_delegations
			 WHERE delegatee_id=$1
			   AND ($2, $3) OVERLAPS (start_date, end_date)
			   AND deleted_at IS NULL
		)
	`, body.DelegateeID, body.StartDate, body.EndDate).Scan(&overlap)
	if overlap {
		c.JSON(http.StatusConflict, gin.H{"error": "overlapping delegation period"})
		return
	}

	// 🧩 Masukkan ke DB (delegator_id diambil dari JWT)
	var id int64
	err := database.Pool.QueryRow(c.Request.Context(), `
		INSERT INTO role_delegations
		  (delegator_id, delegatee_id, role_override, start_date, end_date, created_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
		RETURNING id
	`, userID, body.DelegateeID, body.RoleOverride, body.StartDate, body.EndDate).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 🎯 Broadcast notifikasi
	msg := fmt.Sprintf("Delegasi role '%s' aktif (%s → %s)",
		body.RoleOverride,
		body.StartDate.Format("2006-01-02"),
		body.EndDate.Format("2006-01-02"))
	services.BroadcastAlert(msg, "info")

	middleware.LogAction(c, "role_delegations", id, "CREATE", body)
	c.JSON(http.StatusCreated, gin.H{
		"id":      id,
		"message": "Role delegation created successfully (Grade A++)",
	})
}

// ============================================================
// 📋 GET ACTIVE DELEGATIONS (with names + integrity score)
// ============================================================
func GetActiveDelegations(c *gin.Context) {
	rows, err := database.Pool.Query(c.Request.Context(), `
		SELECT rd.id,
		       dgr.name  AS delegator_name,
		       dge.name  AS delegatee_name,
		       dgr.email AS delegator_email,
		       dge.email AS delegatee_email,
		       rd.role_override,
		       rd.start_date, rd.end_date, rd.created_at
		  FROM role_delegations rd
		  JOIN employees dgr ON dgr.id = rd.delegator_id
		  JOIN employees dge ON dge.id = rd.delegatee_id
		 WHERE NOW() BETWEEN rd.start_date AND rd.end_date
		   AND rd.deleted_at IS NULL
		 ORDER BY rd.end_date`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID             int64     `json:"id"`
		DelegatorName  string    `json:"delegator_name"`
		DelegateeName  string    `json:"delegatee_name"`
		DelegatorEmail string    `json:"delegator_email"`
		DelegateeEmail string    `json:"delegatee_email"`
		RoleOverride   string    `json:"role_override"`
		StartDate      time.Time `json:"start_date"`
		EndDate        time.Time `json:"end_date"`
		CreatedAt      time.Time `json:"created_at"`
		IntegrityScore float64   `json:"delegation_integrity_score"`
		DaysRemaining  int       `json:"days_remaining"`
	}
	var list []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.DelegatorName, &r.DelegateeName, &r.DelegatorEmail,
			&r.DelegateeEmail, &r.RoleOverride, &r.StartDate, &r.EndDate, &r.CreatedAt)
		// 🔹 Hitung sisa hari & score
		days := int(time.Until(r.EndDate).Hours() / 24)
		if days < 0 {
			days = 0
		}
		r.DaysRemaining = days
		r.IntegrityScore = computeDelegationScore(r.RoleOverride, r.DaysRemaining)
		list = append(list, r)
	}
	c.JSON(http.StatusOK, gin.H{"active_delegations": list})
}

// ============================================================
// ⚙️ Helper Functions
// ============================================================
func computeDelegationScore(role string, days int) float64 {
	base := 70.0
	switch role {
	case "super_admin":
		base = 100
	case "manager":
		base = 90
	case "finance":
		base = 85
	case "it_support":
		base = 80
	}
	if days < 7 {
		base -= 10
	}
	if days < 1 {
		base -= 20
	}
	if base < 0 {
		base = 0
	}
	return base
}
