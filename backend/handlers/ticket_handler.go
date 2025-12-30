package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/middleware"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-gonic/gin"
)

//
// ============================================================
// 🔹 Utility & SLA helpers
// ============================================================
//

// nullIfEmpty mengubah string kosong menjadi nil
func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// pick memilih nilai pertama yang tidak kosong
func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseInt64(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

// =====================================================================
// 🔹 SLA Policy Lookup & Calculator
// =====================================================================
func findSLAPolicy(c *gin.Context, categoryCode, serviceCode, impact, urgency string) (models.SLAPolicy, error) {
	var s models.SLAPolicy

	query := `
		SELECT id, name, category_code, service_code, impact, urgency,
		       resulting_priority, response_minutes, resolve_minutes, is_active
		  FROM sla_policies
		 WHERE is_active = true
		   AND (category_code = $1 OR $1 IS NULL OR category_code IS NULL)
		   AND (service_code = $2 OR $2 IS NULL OR service_code IS NULL)
		   AND impact = $3
		   AND urgency = $4
		 ORDER BY category_code NULLS LAST, service_code NULLS LAST
		 LIMIT 1;
	`

	err := database.Pool.QueryRow(c, query,
		categoryCode, serviceCode, impact, urgency,
	).Scan(
		&s.ID, &s.Name, &s.CategoryCode, &s.ServiceCode, &s.Impact, &s.Urgency,
		&s.ResultingPriority, &s.ResponseMinutes, &s.ResolveMinutes, &s.IsActive,
	)

	if err != nil {
		// fallback default SLA (tanpa kategori dan service)
		_ = database.Pool.QueryRow(c, `
			SELECT id, name, category_code, service_code, impact, urgency,
			       resulting_priority, response_minutes, resolve_minutes, is_active
			  FROM sla_policies
			 WHERE is_active = true
			   AND category_code IS NULL
			   AND service_code IS NULL
			 LIMIT 1;
		`).Scan(
			&s.ID, &s.Name, &s.CategoryCode, &s.ServiceCode, &s.Impact, &s.Urgency,
			&s.ResultingPriority, &s.ResponseMinutes, &s.ResolveMinutes, &s.IsActive,
		)
	}

	return s, err
}

func computeSLA(policy models.SLAPolicy) (respDue, resDue time.Time, flag bool, score float64) {
	now := time.Now()
	respDue = now.Add(time.Duration(policy.ResponseMinutes) * time.Minute)
	resDue = now.Add(time.Duration(policy.ResolveMinutes) * time.Minute)
	return respDue, resDue, true, 100
}

// =====================================================================
// 🔹 SLA Recalculation Helper + Auto Breach Alert
// =====================================================================
func RecalculateSLA(c *gin.Context, ticketID int64) {
	var wasBreached bool
	var subject string
	var assignedTo *int64

	// Ambil status sebelumnya
	_ = database.Pool.QueryRow(c, `
		SELECT COALESCE(breach_flag,false), subject, assigned_to_employee_id
		  FROM tickets WHERE id=$1 AND deleted_at IS NULL
	`, ticketID).Scan(&wasBreached, &subject, &assignedTo)

	// Jalankan recalculation
	_, err := database.Pool.Exec(c, `
		UPDATE tickets
		   SET compliance_flag = CASE WHEN NOW() > sla_due_at THEN FALSE ELSE TRUE END,
		       compliance_score = CASE WHEN NOW() > sla_due_at THEN 0 ELSE 100 END,
		       breach_flag = CASE WHEN NOW() > sla_due_at THEN TRUE ELSE FALSE END,
		       updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
	`, ticketID)
	if err != nil {
		log.Printf("[SLA_RECALC_ERROR] Ticket #%d: %v", ticketID, err)
		return
	}

	// Cek apakah baru breach (tidak breach sebelumnya, tapi sekarang breach)
	var nowBreached bool
	_ = database.Pool.QueryRow(c, `
		SELECT COALESCE(breach_flag,false) FROM tickets WHERE id=$1
	`, ticketID).Scan(&nowBreached)

	if !wasBreached && nowBreached {
		// 🔔 Kirim alert via WebSocket
		alertMsg := fmt.Sprintf("⚠️ Ticket #%d ('%s') telah melewati batas SLA!", ticketID, subject)
		services.BroadcastAlert(alertMsg, "warning")

		// 📧 Opsional: Kirim email ke assignee bila ada
		if assignedTo != nil {
			var email string
			_ = database.Pool.QueryRow(c, `SELECT email FROM employees WHERE id=$1`, assignedTo).Scan(&email)
			if email != "" {
				go services.SendEmailNotification(context.Background(), services.TicketNotification{
					ToEmail: email,
					Subject: "Peringatan Pelanggaran SLA",
					Message: fmt.Sprintf("Tiket #%d ('%s') telah melewati batas waktu SLA.", ticketID, subject),
				})
			}
		}

		log.Printf("[SLA_BREACH_ALERT] Ticket #%d breached SLA", ticketID)
	}
}

// ============================================================
// 🔹 CREATE TICKET — POST /tickets
// ============================================================
func CreateTicket(c *gin.Context) {
	var in struct {
		Subject              string  `json:"subject" binding:"required"`
		Description          string  `json:"description"`
		Detail               string  `json:"detail"` // alias lama
		Impact               string  `json:"impact" binding:"required"`
		Urgency              string  `json:"urgency" binding:"required"`
		CategoryCode         *string `json:"category_code"`
		ServiceCode          *string `json:"service_code"`
		RelatedAssetID       *int64  `json:"related_asset_id"`
		AssetID              *int64  `json:"asset_id"` // alias lama
		AssignedToEmployeeID *int64  `json:"assigned_to_employee_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Backward-compat: map detail -> description
	if in.Description == "" && in.Detail != "" {
		in.Description = in.Detail
	}

	// Backward-compat: map asset_id -> related_asset_id
	if in.RelatedAssetID == nil && in.AssetID != nil {
		in.RelatedAssetID = in.AssetID
	}

	// =======================================================
	// 🔹 Validasi enumerasi impact dan urgency
	// =======================================================
	validImpacts := map[string]bool{"Low": true, "Medium": true, "High": true}
	validUrgencies := map[string]bool{"Low": true, "Medium": true, "High": true}

	if !validImpacts[in.Impact] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid impact value (allowed: Low, Medium, High)"})
		return
	}
	if !validUrgencies[in.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid urgency value (allowed: Low, Medium, High)"})
		return
	}

	// =======================================================
	// 🔹 Ambil context user (ID dan role)
	// =======================================================
	roleVal, _ := c.Get("role")
	userRole, _ := roleVal.(string)

	userIDVal, _ := c.Get("user_id")
	var userID int64
	switch v := userIDVal.(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	}

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	createdBy := &userID

	// =======================================================
	// 🔹 Validasi kepemilikan asset (khusus employee)
	// =======================================================
	if in.RelatedAssetID != nil {
		var exists bool
		_ = database.Pool.QueryRow(c,
			`SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND deleted_at IS NULL)`,
			in.RelatedAssetID).Scan(&exists)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid related asset"})
			return
		}

		// Jika role employee → cek kepemilikan aset
		if userRole == "employee" {
			var owned bool
			_ = database.Pool.QueryRow(c, `
				SELECT EXISTS(
					SELECT 1 FROM asset_assignments 
					 WHERE asset_id=$1 AND employee_id=$2 AND returned_at IS NULL
				)`, in.RelatedAssetID, userID).Scan(&owned)

			if !owned {
				c.JSON(http.StatusForbidden, gin.H{"error": "asset tidak terdaftar atas nama Anda"})
				return
			}
		}
	}

	// =======================================================
	// 🔹 Batasi kemampuan assign tiket
	// =======================================================
	if userRole != "super_admin" && userRole != "it_support" {
		in.AssignedToEmployeeID = nil
	}

	// =======================================================
	// 🔹 SLA policy
	// =======================================================
	policy, _ := findSLAPolicy(c,
		strOrEmpty(in.CategoryCode),
		strOrEmpty(in.ServiceCode),
		in.Impact, in.Urgency)

	if policy.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no matching SLA policy found"})
		return
	}

	respDue, resDue, flag, score := computeSLA(policy)

	// =======================================================
	// 🔹 Insert tiket baru
	// =======================================================
	var id int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO tickets
		(subject, description, impact, urgency, priority, status,
		 created_by_employee_id, assigned_to_employee_id, related_asset_id,
		 category_code, service_code, sla_policy_id,
		 response_due_at, sla_due_at,
		 compliance_flag, compliance_score, created_at)
		VALUES ($1,$2,$3,$4,$5,'Open',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
		RETURNING id`,
		in.Subject, in.Description, in.Impact, in.Urgency,
		policy.ResultingPriority,
		createdBy,
		in.AssignedToEmployeeID,
		in.RelatedAssetID,
		in.CategoryCode,
		in.ServiceCode,
		policy.ID,
		respDue, resDue,
		flag, score).Scan(&id)

	if err != nil {
		log.Printf("[TICKET_CREATE_ERROR] %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	// =======================================================
	// 🔹 Logging & notifikasi
	// =======================================================
	middleware.LogAction(c, "tickets", id, "CREATE", in)
	services.BroadcastAlert(fmt.Sprintf("🎫 Ticket #%d created (%s)", id, in.Subject), "info")

	c.JSON(http.StatusCreated, gin.H{
		"id":               id,
		"priority":         policy.ResultingPriority,
		"response_due_at":  respDue,
		"resolve_due_at":   resDue,
		"compliance_flag":  flag,
		"compliance_score": score,
	})
}

// ============================================================
// 🧩 UPDATE TICKET — perbaikan status, SLA, dan eskalasi
// ============================================================
func UpdateTicket(c *gin.Context) {
	id := c.Param("id")

	// ✅ Pastikan user ID valid tanpa type mismatch
	uid, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var userID int64
	switch v := uid.(type) {
	case int:
		userID = int64(v)
	case int64:
		userID = v
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user type"})
		return
	}

	// ✅ Input tambahan (impact, urgency, dsb)
	var in struct {
		Status     *string `json:"status"`
		AssigneeID *int64  `json:"assigned_to_employee_id"`
		Resolution *string `json:"resolution"`
		Impact     *string `json:"impact"`
		Urgency    *string `json:"urgency"`
		Category   *string `json:"category_code"`
		Service    *string `json:"service_code"`
		Escalate   *bool   `json:"escalate"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// =======================================================
	// 🔹 Validasi enumerasi status, impact, urgency
	// =======================================================
	validStatuses := map[string]bool{
		"Open": true, "In Progress": true, "Resolved": true, "Closed": true,
	}
	validImpacts := map[string]bool{"Low": true, "Medium": true, "High": true}
	validUrgencies := map[string]bool{"Low": true, "Medium": true, "High": true}

	if in.Status != nil && !validStatuses[*in.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status value (allowed: Open, In Progress, Resolved, Closed)"})
		return
	}
	if in.Impact != nil && !validImpacts[*in.Impact] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid impact value (allowed: Low, Medium, High)"})
		return
	}
	if in.Urgency != nil && !validUrgencies[*in.Urgency] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid urgency value (allowed: Low, Medium, High)"})
		return
	}

	// ✅ Ambil status lama & SLA
	var oldStatus string
	var slaDue, createdAt *time.Time
	err := database.Pool.QueryRow(c,
		`SELECT status, sla_due_at, created_at FROM tickets WHERE id=$1 AND deleted_at IS NULL`, id).
		Scan(&oldStatus, &slaDue, &createdAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	if createdAt == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "createdAt is nil"})
		return
	}

	// ✅ Blokir update jika tiket sudah Closed
	if oldStatus == "Closed" && (in.Status == nil || *in.Status != "Open") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is closed"})
		return
	}

	// ✅ Validasi transisi status
	if in.Status != nil {
		switch oldStatus {
		case "Open":
			if *in.Status != "In Progress" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition from Open"})
				return
			}
		case "In Progress":
			if *in.Status != "Resolved" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition from In Progress"})
				return
			}
		case "Resolved":
			if *in.Status != "Closed" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transition from Resolved"})
				return
			}
		}
	}

	now := time.Now()
	var resolvedAt, closedAt, responseCompletedAt interface{}
	var responseTimeMinutes, resolutionTimeMinutes interface{}

	// ✅ SLA update sesuai status
	if in.Status != nil {
		switch *in.Status {
		case "In Progress":
			responseCompletedAt = now
			responseTimeMinutes = int(now.Sub(*createdAt).Minutes())
		case "Resolved":
			resolvedAt = now
			resolutionTimeMinutes = int(now.Sub(*createdAt).Minutes())
		case "Closed":
			closedAt = now
		}
	}

	// ✅ Compliance & breach
	var flag bool
	var score float64
	if slaDue != nil {
		flag = !now.After(*slaDue)
		if flag {
			score = 100
		} else {
			score = 0
		}
	}

	// ✅ Escalation
	if in.Escalate != nil && *in.Escalate {
		_, _ = database.Pool.Exec(c, `
			UPDATE tickets
			   SET escalation_level = escalation_level + 1,
			       updated_at = NOW(),
			       updated_by = $1
			 WHERE id = $2 AND deleted_at IS NULL`, userID, id)
		services.BroadcastAlert(fmt.Sprintf("🚨 Ticket #%s di-eskalasi ke level berikutnya", id), "warning")
		c.JSON(http.StatusOK, gin.H{"message": "ticket escalated"})
		return
	}

	// ✅ Update database (impact, urgency, category, dsb)
	query := `
	UPDATE tickets
	   SET status = COALESCE($1, status),
	       assigned_to_employee_id = COALESCE($2, assigned_to_employee_id),
	       impact = COALESCE($3, impact),
	       urgency = COALESCE($4, urgency),
	       category_code = COALESCE($5, category_code),
	       service_code = COALESCE($6, service_code),
	       updated_at = NOW(),
	       updated_by = $7,
	       resolved_at = COALESCE($8, resolved_at),
	       closed_at = COALESCE($9, closed_at),
	       response_completed_at = COALESCE($10, response_completed_at),
	       response_time_minutes = COALESCE($11, response_time_minutes),
	       resolution_time_minutes = COALESCE($12, resolution_time_minutes),
	       compliance_flag = $13,
	       compliance_score = $14,
	       last_status_changed_at = CASE 
	           WHEN $1 IS NOT NULL AND $1 <> status THEN NOW() 
	           ELSE last_status_changed_at 
	       END
	 WHERE id = $15 AND deleted_at IS NULL`

	_, err = database.Pool.Exec(c, query,
		in.Status, in.AssigneeID, in.Impact, in.Urgency, in.Category, in.Service,
		userID, resolvedAt, closedAt, responseCompletedAt,
		responseTimeMinutes, resolutionTimeMinutes,
		flag, score, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ticket"})
		return
	}

	// 🔹 Recalculate SLA setelah update
	RecalculateSLA(c, mustParseInt64(id))

	middleware.LogAction(c, "tickets", mustParseInt64(id), "UPDATE", in)

	// ✅ Broadcast notifikasi jika status berubah
	if in.Status != nil && *in.Status != oldStatus {
		services.BroadcastAlert(fmt.Sprintf("🎫 Ticket #%s status → %s", id, *in.Status), "info")
	}

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

// =====================================================================
// 🔹 ADD COMMENT (POST /tickets/:id/comments)
// =====================================================================
// ============================================================
// 🔹 ADD COMMENT TO TICKET – POST /tickets/:id/comments
// ============================================================
func AddCommentToTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var UploadBasePath = os.Getenv("UPLOAD_PATH_TICKETS")
	if UploadBasePath == "" {
		UploadBasePath = "./uploads/tickets"
	}

	// 🔹 Ambil user dari token (multi-type safe)
	uid, _ := c.Get("user_id")
	var userID int64
	switch v := uid.(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	// 🔹 Handle dua jenis input: JSON biasa dan multipart
	var commentText string
	isResolution := false

	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		commentText = c.PostForm("comment")
		isResolution = c.PostForm("is_resolution") == "true"
	} else {
		var in struct {
			CommentText  string `json:"comment" binding:"required"`
			IsResolution bool   `json:"is_resolution"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		commentText = in.CommentText
		isResolution = in.IsResolution
	}

	// 🔹 Simpan komentar utama
	var commentID int64
	err := database.Pool.QueryRow(c, `
		INSERT INTO ticket_comments (ticket_id, employee_id, comment, created_at)
		 VALUES ($1,$2,$3,NOW())
		 RETURNING id
	`, ticketID, userID, commentText).Scan(&commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}

	// 🔹 Jika multipart: simpan lampiran
	form, _ := c.MultipartForm()
	if form != nil && len(form.File["attachments"]) > 0 {
		for _, file := range form.File["attachments"] {
			mimeType := file.Header.Get("Content-Type")

			// =======================================================
			// 🔹 MIME type whitelist (security hardening)
			// =======================================================
			allowedMIMEs := map[string]bool{
				"application/pdf": true,
				"image/jpeg":      true,
				"image/png":       true,
				"text/plain":      true,
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // DOCX
			}

			if !allowedMIMEs[mimeType] {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("unsupported file type: %s (allowed: pdf, jpeg, png, txt, docx)", mimeType),
				})
				return
			}

			// =======================================================
			// 🔹 Simpan file ke folder upload
			// =======================================================
			dst := fmt.Sprintf("%s/%s_%s", UploadBasePath, time.Now().Format("20060102150405"), file.Filename)

			if err := c.SaveUploadedFile(file, dst); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save attachment"})
				return
			}

			// =======================================================
			// 🔹 Simpan metadata lampiran ke database
			// =======================================================
			_, _ = database.Pool.Exec(c, `
		INSERT INTO ticket_attachments (ticket_id, comment_id, filename, path, mime_type, size, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW())
	`,
				ticketID, commentID, file.Filename, dst, mimeType, file.Size,
			)
		}

	}

	// 🔹 Jika comment bersifat resolusi, update tiket
	if isResolution {
		_, _ = database.Pool.Exec(c, `
			UPDATE tickets
			   SET status='Resolved',
			       resolved_at=NOW(),
			       compliance_flag = CASE WHEN NOW()>sla_due_at THEN FALSE ELSE TRUE END,
			       compliance_score = CASE WHEN NOW()>sla_due_at THEN 0 ELSE 100 END,
			       updated_at=NOW(),
			       updated_by=$2
			 WHERE id=$1 AND deleted_at IS NULL
		`, ticketID, userID)
	}

	// 🔹 Logging & broadcast
	middleware.LogAction(c, "ticket_comments", commentID, "COMMENT", gin.H{"comment": commentText})
	services.BroadcastAlert(fmt.Sprintf("💬 Comment added to ticket #%s", ticketID), "info")

	go func() {
		var ci models.TicketCommentInfo
		_ = database.Pool.QueryRow(c, `
		SELECT c.id, c.employee_id, e.name, c.comment, c.created_at
		  FROM ticket_comments c
		  JOIN employees e ON e.id = c.employee_id
		 WHERE c.id=$1
	`, commentID).Scan(&ci.ID, &ci.EmployeeID, &ci.EmployeeName, &ci.Comment, &ci.CreatedAt)

		// ambil attachments-nya
		aRows, _ := database.Pool.Query(c, `
		SELECT id, filename, path, mime_type, size, created_at
		  FROM ticket_attachments
		 WHERE comment_id=$1
		 ORDER BY id
	`, commentID)
		var atts []models.TicketAttachment
		for aRows.Next() {
			var a models.TicketAttachment
			if err := aRows.Scan(&a.ID, &a.Filename, &a.Path, &a.MimeType, &a.Size, &a.CreatedAt); err == nil {
				a.URL = strings.TrimPrefix(a.Path, ".")
				atts = append(atts, a)
			}
		}
		aRows.Close()
		ci.Attachments = atts

		websocket.BroadcastTicketComment(parseInt64(ticketID), ci)
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "comment added",
		"id":      commentID,
	})
}

// ============================================================
// 🔹 GET RECENT COMMENTS FOR TICKET – GET /tickets/:id/comments/recent
// ============================================================
func GetRecentComments(c *gin.Context) {
	ticketID := c.Param("id")

	rows, err := database.Pool.Query(c, `
		SELECT c.id, c.employee_id, e.name, c.comment, c.created_at,
		       COALESCE(
		           json_agg(
		               json_build_object(
		                   'id', a.id,
		                   'filename', a.filename,
		                   'path', a.path,
		                   'url', REPLACE(a.path, '.', ''),
		                   'mime_type', a.mime_type,
		                   'size', a.size,
		                   'created_at', a.created_at
		               )
		           ) FILTER (WHERE a.id IS NOT NULL),
		           '[]'
		       ) AS attachments
		  FROM ticket_comments c
		  JOIN employees e ON e.id = c.employee_id
		  LEFT JOIN ticket_attachments a ON a.comment_id = c.id
		 WHERE c.ticket_id = $1
		 GROUP BY c.id, e.name, c.comment, c.created_at, c.employee_id
		 ORDER BY c.created_at DESC
		 LIMIT 20;
	`, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load comments"})
		return
	}
	defer rows.Close()

	var comments []models.TicketCommentInfo
	for rows.Next() {
		var ci models.TicketCommentInfo
		var rawJSON []byte
		if err := rows.Scan(&ci.ID, &ci.EmployeeID, &ci.EmployeeName, &ci.Comment, &ci.CreatedAt, &rawJSON); err == nil {
			_ = json.Unmarshal(rawJSON, &ci.Attachments)
			comments = append(comments, ci)
		}
	}

	if comments == nil {
		comments = []models.TicketCommentInfo{}
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket_id": ticketID,
		"comments":  comments,
		"count":     len(comments),
	})
}

// =====================================================================
// 🔹 Resolve / Close / Escalate
// =====================================================================

func ResolveTicket(c *gin.Context) {
	tid := c.Param("id")
	uid, _ := c.Get("user_id")
	userID := int64(uid.(int))
	var in struct {
		ResolutionNote string `json:"resolution_note"`
	}
	_ = c.ShouldBindJSON(&in)

	_, err := database.Pool.Exec(c, `
		UPDATE tickets SET status='Resolved', resolved_at=NOW(),
		       compliance_flag=CASE WHEN NOW()>sla_due_at THEN FALSE ELSE TRUE END,
		       compliance_score=CASE WHEN NOW()>sla_due_at THEN 0 ELSE 100 END,
		       updated_by=$2, updated_at=NOW()
		 WHERE id=$1 AND deleted_at IS NULL`, tid, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "resolve failed"})
		return
	}
	RecalculateSLA(c, mustParseInt64(tid))
	middleware.LogAction(c, "tickets", mustParseInt64(tid), "RESOLVE", in)
	services.BroadcastAlert(fmt.Sprintf("✅ Ticket #%s resolved", tid), "success")
	c.JSON(http.StatusOK, gin.H{"message": "resolved"})
}

func CloseTicket(c *gin.Context) {
	tid := c.Param("id")
	uid, _ := c.Get("user_id")
	userID := int64(uid.(int))
	_, err := database.Pool.Exec(c, `
		UPDATE tickets SET status='Closed', closed_at=NOW(),
		       compliance_flag=CASE WHEN NOW()>sla_due_at THEN FALSE ELSE TRUE END,
		       compliance_score=CASE WHEN NOW()>sla_due_at THEN 0 ELSE 100 END,
		       updated_by=$2, updated_at=NOW()
		 WHERE id=$1 AND deleted_at IS NULL`, tid, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "close failed"})
		return
	}
	RecalculateSLA(c, mustParseInt64(tid))
	middleware.LogAction(c, "tickets", mustParseInt64(tid), "CLOSE", nil)
	services.BroadcastAlert(fmt.Sprintf("🔒 Ticket #%s closed", tid), "info")
	c.JSON(http.StatusOK, gin.H{"message": "closed"})
}

func EscalateTicket(c *gin.Context) {
	tid := c.Param("id")
	var in struct {
		Level int `json:"level" binding:"required"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// 🔒 Role-based authorization
	roleVal, _ := c.Get("role")
	userRole, _ := roleVal.(string)
	if userRole != "super_admin" && userRole != "it_support" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized escalate"})
		return
	}

	// 🔹 Ambil level lama untuk log perbandingan
	var oldLevel int
	err := database.Pool.QueryRow(c, `
		SELECT COALESCE(escalation_level,0) FROM tickets WHERE id=$1 AND deleted_at IS NULL
	`, tid).Scan(&oldLevel)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	// 🔹 Update escalation level
	_, err = database.Pool.Exec(c, `
		UPDATE tickets 
		   SET escalation_level=$1, updated_at=NOW(), updated_by=$2
		 WHERE id=$3 AND deleted_at IS NULL
	`, in.Level, c.GetInt64("user_id"), tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to escalate"})
		return
	}

	// 🔹 Log audit lengkap
	middleware.LogAction(c, "tickets", mustParseInt64(tid), "ESCALATE", gin.H{
		"old_level": oldLevel,
		"new_level": in.Level,
	})

	// 🔹 Broadcast notifikasi
	services.BroadcastAlert(fmt.Sprintf("🚨 Ticket #%s escalated from level %d → %d", tid, oldLevel, in.Level), "warning")

	// 🔹 Update SLA compliance otomatis bila level meningkat signifikan
	if in.Level > oldLevel {
		RecalculateSLA(c, mustParseInt64(tid))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("ticket escalated to level %d", in.Level),
	})
}

// ============================================================
// 🔹 GET TICKET BY ID – GET /tickets/:id
// ============================================================
// ============================================================
// ⚡ OPTIMIZED GET TICKET BY ID – GET /tickets/:id
// ============================================================
func GetTicketByID(c *gin.Context) {
	id := c.Param("id")

	var t models.TicketDetail
	err := database.Pool.QueryRow(c, `
    SELECT 
        t.id, t.subject, t.status, t.priority,
        t.category_code, t.service_code, t.impact, t.urgency,
        t.created_by_employee_id, e1.name AS created_by_name,
        t.assigned_to_employee_id, e2.name AS assigned_to_employee_name,
        t.last_assigned_by, e3.name AS last_assigned_by_name,
        t.related_asset_id,
        t.response_due_at,         -- 🔹 tambahan
        t.sla_due_at,
        t.sla_breached_at,
        t.created_at, t.updated_at, t.description
    FROM tickets t
    JOIN employees e1 ON e1.id = t.created_by_employee_id
    LEFT JOIN employees e2 ON e2.id = t.assigned_to_employee_id
    LEFT JOIN employees e3 ON e3.id = t.last_assigned_by
    WHERE t.id = $1 AND t.deleted_at IS NULL
`, id).Scan(
		&t.ID, &t.Subject, &t.Status, &t.Priority,
		&t.CategoryCode, &t.ServiceCode, &t.Impact, &t.Urgency,
		&t.CreatedByEmployeeID, &t.CreatedByEmployeeName,
		&t.AssignedToEmployeeID, &t.AssignedToEmployeeName,
		&t.LastAssignedBy, &t.LastAssignedByName,
		&t.RelatedAssetID,
		&t.ResponseDueAt, // 🔹 tambahan
		&t.SLADueAt,
		&t.SLABreachedAt,
		&t.CreatedAt, &t.UpdatedAt, &t.Description,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	// =======================================================
	// 🔹 Comments + Attachments (optimized single query)
	// =======================================================
	rows, err := database.Pool.Query(c, `
	SELECT 
		c.id AS comment_id,
		c.employee_id,
		e.name AS employee_name,
		c.comment,
		c.created_at,
		COALESCE(
			json_agg(
				json_build_object(
					'id', a.id,
					'filename', a.filename,
					'path', a.path,
					'url', REPLACE(a.path, '.', ''),
					'mime_type', a.mime_type,
					'size', a.size,
					'created_at', a.created_at
				)
				ORDER BY a.id
			) FILTER (WHERE a.id IS NOT NULL),
			'[]'
		) AS attachments
	FROM ticket_comments c
	JOIN employees e ON e.id = c.employee_id
	LEFT JOIN ticket_attachments a ON a.comment_id = c.id
	WHERE c.ticket_id = $1
	GROUP BY c.id, e.name, c.comment, c.created_at, c.employee_id
	ORDER BY c.created_at ASC
`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load comments"})
		return
	}
	defer rows.Close()

	var comments []models.TicketCommentInfo
	for rows.Next() {
		var ci models.TicketCommentInfo
		var rawJSON []byte
		if err := rows.Scan(&ci.ID, &ci.EmployeeID, &ci.EmployeeName, &ci.Comment, &ci.CreatedAt, &rawJSON); err == nil {
			_ = json.Unmarshal(rawJSON, &ci.Attachments)
			comments = append(comments, ci)
		}
	}
	t.Comments = comments

	// --- Maintenance Logs (opsional) ---
	mrows, _ := database.Pool.Query(c, `
		SELECT id, asset_id, log_type, description, cost,
		       log_date, performed_by_employee_id, vendor, ticket_id, created_at
		  FROM asset_maintenance_logs
		 WHERE ticket_id = $1
		 ORDER BY log_date DESC
	`, id)
	var logs []models.AssetMaintenanceLog
	for mrows.Next() {
		var l models.AssetMaintenanceLog
		_ = mrows.Scan(
			&l.ID, &l.AssetID, &l.LogType, &l.Description,
			&l.Cost, &l.LogDate, &l.PerformedByEmployeeID,
			&l.Vendor, &l.TicketID, &l.CreatedAt,
		)
		logs = append(logs, l)
	}
	mrows.Close()
	t.MaintenanceLogs = logs

	c.JSON(http.StatusOK, t)
}

// ============================================================
// 🔹 LIST + FILTER + CSV – GET /tickets
// ============================================================
// ============================================================
// 🔹 GET ALL TICKETS — GET /tickets
// ============================================================
func GetAllTickets(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	status := strings.TrimSpace(c.Query("status"))
	priority := strings.TrimSpace(c.Query("priority"))
	assignee := strings.TrimSpace(c.Query("assignee"))
	category := strings.TrimSpace(c.Query("category_code"))
	service := strings.TrimSpace(c.Query("service_code"))

	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	where := []string{"t.deleted_at IS NULL"}
	params := []any{}
	i := 1

	if q != "" {
		where = append(where, fmt.Sprintf("(LOWER(t.subject) LIKE $%d OR CAST(t.id AS TEXT) LIKE $%d)", i, i))
		params = append(params, "%"+strings.ToLower(q)+"%")
		i++
	}
	if status != "" && strings.ToLower(status) != "all" {
		where = append(where, fmt.Sprintf("t.status = $%d", i))
		params = append(params, status)
		i++
	}
	if priority != "" && strings.ToLower(priority) != "all" {
		where = append(where, fmt.Sprintf("t.priority = $%d", i))
		params = append(params, priority)
		i++
	}
	if assignee != "" && strings.ToLower(assignee) != "all" {
		where = append(where, fmt.Sprintf("t.assigned_to_employee_id = $%d", i))
		aid, _ := strconv.ParseInt(assignee, 10, 64)
		params = append(params, aid)
		i++
	}
	if category != "" && strings.ToLower(category) != "all" {
		where = append(where, fmt.Sprintf("t.category_code = $%d", i))
		params = append(params, category)
		i++
	}
	if service != "" && strings.ToLower(service) != "all" {
		where = append(where, fmt.Sprintf("t.service_code = $%d", i))
		params = append(params, service)
		i++
	}

	// =======================================================
	// 🔹 Role-based filter
	// =======================================================
	userRole, _ := c.Get("role")
	userID, _ := c.Get("user_id")

	// Jika role adalah employee, hanya tampilkan tiket miliknya sendiri
	if roleStr, ok := userRole.(string); ok && roleStr == "employee" {
		if uid, ok := userID.(int64); ok {
			where = append(where, fmt.Sprintf("t.created_by_employee_id = $%d", i))
			params = append(params, uid)
			i++
		}
	}

	whereSQL := strings.Join(where, " AND ")
	query := fmt.Sprintf(`
		SELECT t.id, t.subject, t.status, t.priority,
		       t.category_code, t.service_code, t.impact, t.urgency,
		       t.created_by_employee_id, e1.name,
		       t.assigned_to_employee_id, e2.name,
		       t.related_asset_id, t.sla_due_at, t.sla_breached_at,
		       t.created_at, t.updated_at
		  FROM tickets t
		  JOIN employees e1 ON e1.id = t.created_by_employee_id
		  LEFT JOIN employees e2 ON e2.id = t.assigned_to_employee_id
		 WHERE %s
		 ORDER BY t.updated_at DESC
		 LIMIT $%d OFFSET $%d`, whereSQL, i, i+1)
	params = append(params, limit, offset)

	rows, err := database.Pool.Query(c, query, params...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query tickets"})
		return
	}
	defer rows.Close()

	var list []models.TicketInfo
	for rows.Next() {
		var ti models.TicketInfo
		if err := rows.Scan(
			&ti.ID, &ti.Subject, &ti.Status, &ti.Priority,
			&ti.CategoryCode, &ti.ServiceCode, &ti.Impact, &ti.Urgency,
			&ti.CreatedByEmployeeID, &ti.CreatedByEmployeeName,
			&ti.AssignedToEmployeeID, &ti.AssignedToEmployeeName,
			&ti.RelatedAssetID, &ti.SLADueAt, &ti.SLABreachedAt,
			&ti.CreatedAt, &ti.UpdatedAt,
		); err == nil {
			list = append(list, ti)
		}
	}

	// =======================================================
	// 🔹 CSV export support
	// =======================================================
	if strings.ToLower(c.Query("format")) == "csv" {
		c.Header("Content-Disposition", "attachment; filename=tickets.csv")
		c.Header("Content-Type", "text/csv")
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"ID", "Subject", "Status", "Priority", "Reporter", "Assignee", "UpdatedAt"})
		for _, t := range list {
			assigneeName := ""
			if t.AssignedToEmployeeName != nil {
				assigneeName = *t.AssignedToEmployeeName
			}
			_ = w.Write([]string{
				fmt.Sprint(t.ID), t.Subject, t.Status, t.Priority,
				t.CreatedByEmployeeName, assigneeName, t.UpdatedAt.Format(time.RFC3339),
			})
		}
		w.Flush()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       list,
		"pagination": gin.H{"current_page": page, "limit": limit},
	})
}

// ============================================================
// 🔹 ASSIGN – POST /tickets/:id/assign
// ============================================================
func AssignTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var input struct {
		AssigneeID int64 `json:"assignee_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	var exists bool
	err := database.Pool.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM tickets WHERE id=$1 AND deleted_at IS NULL)`,
		ticketID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	_, err = database.Pool.Exec(c, `
		UPDATE tickets
		   SET assigned_to_employee_id=$1,
		       status='In Progress',
		       response_completed_at = NOW(),
		       response_time_minutes = EXTRACT(EPOCH FROM (NOW() - created_at))/60,
		       updated_at=NOW()
		 WHERE id=$2`,
		input.AssigneeID, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign ticket"})
		return
	}

	// 🔹 Recalculate SLA setelah assign
	RecalculateSLA(c, mustParseInt64(ticketID))
	middleware.LogAction(c, "tickets", mustParseInt64(ticketID), "ASSIGN", input)

	go func() {
		var email string
		_ = database.Pool.QueryRow(context.Background(),
			`SELECT email FROM employees WHERE id=$1`, input.AssigneeID).Scan(&email)
		if email != "" {
			services.SendEmailNotification(context.Background(), services.TicketNotification{
				ToEmail: email,
				Subject: "Tiket Baru Telah Ditetapkan",
				Message: fmt.Sprintf("Anda ditugaskan menangani tiket #%s.", ticketID),
			})
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "ticket assigned successfully"})
}

// ============================================================
// 🔹 DELETE – soft delete
// ============================================================
// ============================================================
// 🔹 DELETE TICKET (Soft Delete + Linked Records)
// ============================================================
func DeleteTicket(c *gin.Context) {
	ticketID := c.Param("id")
	var exists bool

	// ✅ Pastikan tiket masih aktif
	err := database.Pool.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM tickets WHERE id=$1 AND deleted_at IS NULL)`,
		ticketID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	// ✅ Role-based authorization
	roleVal, _ := c.Get("role")
	userRole, _ := roleVal.(string)
	if userRole != "super_admin" && userRole != "it_support" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized delete"})
		return
	}

	tx, err := database.Pool.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(c)

	// =======================================================
	// 🔹 Tandai tiket sebagai deleted
	// =======================================================
	_, err = tx.Exec(c, `
		UPDATE tickets 
		   SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL
	`, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ticket"})
		return
	}

	// =======================================================
	// 🔹 Soft delete comments & attachments
	// =======================================================
	_, _ = tx.Exec(c, `
		UPDATE ticket_comments 
		   SET deleted_at = NOW() 
		 WHERE ticket_id = $1 AND deleted_at IS NULL
	`, ticketID)

	_, _ = tx.Exec(c, `
		UPDATE ticket_attachments 
		   SET deleted_at = NOW() 
		 WHERE ticket_id = $1 AND deleted_at IS NULL
	`, ticketID)

	// =======================================================
	// 🔹 Audit linkage: log penghapusan tiket & children
	// =======================================================
	middleware.LogAction(c, "tickets", mustParseInt64(ticketID), "DELETE", gin.H{
		"ticket_id": ticketID,
		"cascade":   []string{"ticket_comments", "ticket_attachments"},
	})

	if err = tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit delete"})
		return
	}

	services.BroadcastAlert(fmt.Sprintf("🗑️ Ticket #%s deleted by %s", ticketID, userRole), "warning")

	c.JSON(http.StatusOK, gin.H{"message": "ticket and related data deleted successfully"})
}

// ============================================================
// 🔹 HISTORY – GET /tickets/:id/history
// ============================================================
func GetTicketHistory(c *gin.Context) {
	ticketID := c.Param("id")

	rows, err := database.Pool.Query(c, `
		SELECT id, actor_id, action, changes, created_at
		  FROM audit_logs
		 WHERE entity_name='tickets' AND entity_id=$1
		 ORDER BY created_at ASC`, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ticket history"})
		return
	}
	defer rows.Close()

	type History struct {
		ID        int64       `json:"id"`
		ActorID   *int64      `json:"actor_id,omitempty"`
		Action    string      `json:"action"`
		Changes   interface{} `json:"changes"`
		Timestamp time.Time   `json:"timestamp"`
	}
	var history []History
	for rows.Next() {
		var h History
		_ = rows.Scan(&h.ID, &h.ActorID, &h.Action, &h.Changes, &h.Timestamp)
		history = append(history, h)
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// ============================================================
// 🔹 METRICS – GET /tickets/metrics
// ============================================================
func GetTicketMetrics(c *gin.Context) {
	query := `
		SELECT
			-- ⏱️ MTTR: rata-rata waktu penyelesaian (Resolved/Closed)
			COALESCE(
				ROUND(
					AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 60)
					FILTER (WHERE resolved_at IS NOT NULL),
					2
				),
				0
			) AS mttr_minutes,

			-- ⏱️ MTTA: rata-rata waktu tanggapan pertama
			COALESCE(
				ROUND(
					AVG(EXTRACT(EPOCH FROM (response_completed_at - created_at)) / 60)
					FILTER (WHERE response_completed_at IS NOT NULL),
					2
				),
				0
			) AS mtta_minutes,

			-- ⚠️ breach_rate: persentase tiket yang melewati SLA
			COALESCE(
				ROUND(
					100.0 * SUM(
						CASE 
							WHEN breach_flag THEN 1
							WHEN compliance_flag = FALSE THEN 1
							ELSE 0 
						END
					) / NULLIF(COUNT(*), 0),
					2
				),
				0
			) AS breach_rate
		FROM tickets
		WHERE deleted_at IS NULL;
	`

	var mttr, mtta, breachRate *float64
	err := database.Pool.QueryRow(c, query).Scan(&mttr, &mtta, &breachRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mttr_minutes":    mttr,
		"mtta_minutes":    mtta,
		"sla_breach_rate": breachRate,
		"generated_at":    time.Now(),
	})
}

// ============================================================
// 🔹 PREVIEW SLA – GET /sla-policies/preview
// ============================================================

func PreviewSLA(c *gin.Context) {
	category := strings.TrimSpace(c.Query("category_code"))
	service := strings.TrimSpace(c.Query("service_code"))
	impact := strings.TrimSpace(c.Query("impact"))
	urgency := strings.TrimSpace(c.Query("urgency"))

	if impact == "" || urgency == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "impact dan urgency wajib diisi"})
		return
	}

	// cari SLA paling cocok
	policy, err := findSLAPolicy(c, category, service, impact, urgency)
	if err != nil {
		// fallback ke SLA default (tanpa kategori & service)
		_ = database.Pool.QueryRow(c, `
			SELECT id, name, resulting_priority, response_minutes, resolve_minutes
			  FROM sla_policies
			 WHERE is_active = true
			   AND category_code IS NULL
			   AND service_code IS NULL
			 LIMIT 1`,
		).Scan(&policy.ID, &policy.Name, &policy.ResultingPriority, &policy.ResponseMinutes, &policy.ResolveMinutes)
	}

	if policy.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"policy_id":   policy.ID,
		"policy_name": policy.Name,
		"priority":    policy.ResultingPriority,
		"response":    policy.ResponseMinutes,
		"resolve":     policy.ResolveMinutes,
		"checked_at":  time.Now(),
	})
}

// ============================================================
// 🔹 RESTORE TICKET – PUT /tickets/:id/restore
// ============================================================
func RestoreTicket(c *gin.Context) {
	ticketID := c.Param("id")

	// ✅ Role-based authorization
	roleVal, _ := c.Get("role")
	userRole, _ := roleVal.(string)
	if userRole != "super_admin" && userRole != "it_support" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized restore"})
		return
	}

	tx, err := database.Pool.Begin(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(c)

	// =======================================================
	// 🔹 Periksa apakah tiket sudah dihapus
	// =======================================================
	var deletedAt *time.Time
	err = tx.QueryRow(c, `
		SELECT deleted_at FROM tickets WHERE id=$1
	`, ticketID).Scan(&deletedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}
	if deletedAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is already active"})
		return
	}

	// =======================================================
	// 🔹 Pulihkan tiket dan entitas terkait
	// =======================================================
	_, err = tx.Exec(c, `
		UPDATE tickets 
		   SET deleted_at = NULL, updated_at = NOW()
		 WHERE id = $1
	`, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore ticket"})
		return
	}

	_, _ = tx.Exec(c, `
		UPDATE ticket_comments 
		   SET deleted_at = NULL
		 WHERE ticket_id = $1
	`, ticketID)

	_, _ = tx.Exec(c, `
		UPDATE ticket_attachments 
		   SET deleted_at = NULL
		 WHERE ticket_id = $1
	`, ticketID)

	if err = tx.Commit(c); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit restore"})
		return
	}

	// =======================================================
	// 🔹 Audit logging & notification
	// =======================================================
	middleware.LogAction(c, "tickets", mustParseInt64(ticketID), "RESTORE", gin.H{
		"ticket_id": ticketID,
		"cascade":   []string{"ticket_comments", "ticket_attachments"},
	})

	services.BroadcastAlert(fmt.Sprintf("♻️ Ticket #%s restored by %s", ticketID, userRole), "success")

	c.JSON(http.StatusOK, gin.H{"message": "ticket and related data restored successfully"})
}

// =====================================================================
// 🔹 Helpers
// =====================================================================

func mustParseInt64(s string) int64 {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Panicf("mustParseInt64: failed to parse '%s' as int64: %v", s, err)
	}
	return val
}
