// File: backend/handlers/ticket_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"github.com/andripriyatnaputra/asset-management-system/backend/models"
	"github.com/andripriyatnaputra/asset-management-system/backend/services"
	"github.com/andripriyatnaputra/asset-management-system/backend/websocket"
	"github.com/gin-gonic/gin"
)

type CreateTicketRequest struct {
	Subject        string `json:"subject" binding:"required"`
	Description    string `json:"description"`
	Priority       string `json:"priority"`
	RelatedAssetID *int64 `json:"related_asset_id"`
}

// CreateTicket allows a logged-in user to create a new help desk ticket
func CreateTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	createdByEmployeeID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	if req.Priority == "" {
		req.Priority = "Medium"
	}

	query := `
		INSERT INTO tickets (subject, description, priority, related_asset_id, created_by_employee_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6) 
		RETURNING id, status, created_at, updated_at`

	var newTicket models.Ticket
	err := database.Pool.QueryRow(context.Background(), query,
		req.Subject, req.Description, req.Priority, req.RelatedAssetID, createdByEmployeeID, time.Now(),
	).Scan(&newTicket.ID, &newTicket.Status, &newTicket.CreatedAt, &newTicket.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket", "detail": err.Error()})
		return
	}

	newTicket.Subject = req.Subject
	newTicket.Description = req.Description
	newTicket.Priority = req.Priority
	newTicket.RelatedAssetID = req.RelatedAssetID
	newTicket.CreatedByEmployeeID = createdByEmployeeID.(int64)

	c.JSON(http.StatusCreated, newTicket)
}

func GetAllTickets(c *gin.Context) {
	userRole, _ := c.Get("userRole")
	userID, _ := c.Get("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	status := c.Query("status")
	priority := c.Query("priority")

	baseQuery := ` FROM tickets t JOIN employees e ON t.created_by_employee_id = e.id `
	whereClauses := []string{"t.deleted_at IS NULL"}
	params := []interface{}{}
	paramCount := 1

	if userRole != "super_admin" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.created_by_employee_id = $%d", paramCount))
		params = append(params, userID)
		paramCount++
	}
	if status != "" && status != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.status = $%d", paramCount))
		params = append(params, status)
		paramCount++
	}
	if priority != "" && priority != "all" {
		whereClauses = append(whereClauses, fmt.Sprintf("t.priority = $%d", paramCount))
		params = append(params, priority)
		paramCount++
	}
	whereClauseStr := " WHERE " + strings.Join(whereClauses, " AND ")

	countQuery := `SELECT COUNT(t.id)` + baseQuery + whereClauseStr
	var totalRecords int64
	err := database.Pool.QueryRow(context.Background(), countQuery, params...).Scan(&totalRecords)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count tickets"})
		return
	}

	selectClause := `SELECT t.id, t.subject, t.status, t.priority, t.created_by_employee_id, e.name as created_by_employee_name, t.related_asset_id, t.created_at, t.updated_at`
	dataQuery := fmt.Sprintf(`%s %s %s ORDER BY t.updated_at DESC LIMIT $%d OFFSET $%d`,
		selectClause, baseQuery, whereClauseStr, paramCount, paramCount+1)

	finalParams := append(params, limit, offset)
	rows, err := database.Pool.Query(context.Background(), dataQuery, finalParams...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}
	defer rows.Close()

	tickets := []models.TicketInfo{}
	for rows.Next() {
		var ticket models.TicketInfo
		if err := rows.Scan(&ticket.ID, &ticket.Subject, &ticket.Status, &ticket.Priority, &ticket.CreatedByEmployeeID, &ticket.CreatedByEmployeeName, &ticket.RelatedAssetID, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan ticket data"})
			return
		}
		tickets = append(tickets, ticket)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tickets,
		"pagination": gin.H{
			"total_records": totalRecords,
			"current_page":  page,
			"page_size":     limit,
			"total_pages":   (totalRecords + int64(limit) - 1) / int64(limit),
		},
	})
}

func GetTicketByID(c *gin.Context) {
	ticketID := c.Param("id")
	var ticketDetail models.TicketDetail

	// Ambil info user yang sedang login dari token
	userRole, _ := c.Get("userRole")
	userID, _ := c.Get("userID")

	// 1. Ambil informasi utama tiket (blok duplikat sudah dihapus)
	queryTicket := `
		SELECT 
			t.id, t.subject, t.status, t.priority, t.created_by_employee_id, e.name as created_by_employee_name, 
			t.created_at, t.updated_at, t.description
		FROM tickets t
		JOIN employees e ON t.created_by_employee_id = e.id
		WHERE t.id = $1 AND t.deleted_at IS NULL`

	err := database.Pool.QueryRow(context.Background(), queryTicket, ticketID).Scan(
		&ticketDetail.ID, &ticketDetail.Subject, &ticketDetail.Status, &ticketDetail.Priority,
		&ticketDetail.CreatedByEmployeeID, &ticketDetail.CreatedByEmployeeName,
		&ticketDetail.CreatedAt, &ticketDetail.UpdatedAt, &ticketDetail.Description,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// --- LOGIKA OTORISASI YANG HILANG DITAMBAHKAN DI SINI ---
	// Cek apakah user adalah admin ATAU pemilik tiket
	if userRole != "super_admin" && ticketDetail.CreatedByEmployeeID != userID.(int64) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to view this ticket"})
		return
	}
	// -----------------------------------------------------------

	// 2. Ambil semua komentar untuk tiket ini (logika ini sudah benar)
	queryComments := `
		SELECT tc.id, tc.employee_id, e.name as employee_name, tc.comment, tc.created_at
		FROM ticket_comments tc
		JOIN employees e ON tc.employee_id = e.id
		WHERE tc.ticket_id = $1
		ORDER BY tc.created_at ASC`

	rows, err := database.Pool.Query(context.Background(), queryComments, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments"})
		return
	}
	defer rows.Close()

	comments := []models.TicketCommentInfo{}
	for rows.Next() {
		var comment models.TicketCommentInfo
		if err := rows.Scan(&comment.ID, &comment.EmployeeID, &comment.EmployeeName, &comment.Comment, &comment.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan comment data"})
			return
		}
		comments = append(comments, comment)
	}
	ticketDetail.Comments = comments

	queryLogs := `SELECT id, asset_id, ticket_id, log_type, description, cost, log_date, created_at FROM asset_maintenance_logs WHERE ticket_id = $1 ORDER BY log_date DESC`

	logRows, err := database.Pool.Query(context.Background(), queryLogs, ticketID)
	if err != nil { /* ... error handling ... */
	}
	defer logRows.Close()

	logs := []models.AssetMaintenanceLog{}
	for logRows.Next() {
		var log models.AssetMaintenanceLog
		// Scan semua field log
		if err := logRows.Scan(&log.ID, &log.AssetID, &log.TicketID, &log.LogType, &log.Description, &log.Cost, &log.LogDate, &log.CreatedAt); err != nil { /* ... */
		}
		logs = append(logs, log)
	}
	ticketDetail.MaintenanceLogs = logs

	c.JSON(http.StatusOK, ticketDetail)
}

// Mengambil email pembuat tiket dan admin yang di-assign
func getTicketStakeholderEmails(ticketID string) (creatorEmail string, assigneeEmail string) {
	query := `
		SELECT 
			creator.email, 
			assignee.email 
		FROM tickets t
		JOIN employees creator ON t.created_by_employee_id = creator.id
		LEFT JOIN employees assignee ON t.assigned_to_employee_id = assignee.id
		WHERE t.id = $1`

	database.Pool.QueryRow(context.Background(), query, ticketID).Scan(&creatorEmail, &assigneeEmail)
	return
}

type AddCommentRequest struct {
	Comment string `json:"comment" binding:"required"`
}

// AddCommentToTicket menambahkan komentar baru ke sebuah tiket
func AddCommentToTicket(c *gin.Context) {
	ticketID := c.Param("id")
	var req AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment text is required"})
		return
	}

	employeeID, _ := c.Get("userID")
	userName, _ := c.Get("userName")

	tx, err := database.Pool.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	// 1. Masukkan komentar baru
	queryInsert := `INSERT INTO ticket_comments (ticket_id, employee_id, comment) VALUES ($1, $2, $3)`
	_, err = tx.Exec(context.Background(), queryInsert, ticketID, employeeID, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}

	// 2. Perbarui 'updated_at' di tiket utama
	_, err = tx.Exec(context.Background(), "UPDATE tickets SET updated_at = NOW() WHERE id = $1", ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket timestamp"})
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// KIRIM NOTIFIKASI (EMAIL & WEBSOCKET)
	go sendNewCommentNotification(ticketID, userName.(string))

	c.JSON(http.StatusCreated, gin.H{"message": "Comment added successfully"})
}

type UpdateTicketRequest struct {
	Status     string `json:"status"`
	Priority   string `json:"priority"`
	AssignedTo *int64 `json:"assigned_to_employee_id"`
}

func UpdateTicket(c *gin.Context) {
	ticketID := c.Param("id")
	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var currentStatus string
	err := database.Pool.QueryRow(context.Background(), "SELECT status FROM tickets WHERE id = $1", ticketID).Scan(&currentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	query := `UPDATE tickets SET status = $1, priority = $2, assigned_to_employee_id = $3, updated_at = NOW() WHERE id = $4`
	commandTag, err := database.Pool.Exec(context.Background(), query, req.Status, req.Priority, req.AssignedTo, ticketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket", "detail": err.Error()})
		return
	}
	if commandTag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	// KIRIM NOTIFIKASI (EMAIL & WEBSOCKET) JIKA STATUS BERUBAH
	if currentStatus != req.Status {
		go sendStatusUpdateNotification(ticketID, req.Status)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket updated successfully"})
}

// Mengirim notifikasi komentar baru
type NotificationMessage struct {
	Type     string `json:"type"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message"`
}

func getTicketStakeholders(ticketID string) (creatorEmail string, creatorID int64, assigneeEmail string, assigneeID int64) {
	query := `
		SELECT 
			creator.id, creator.email, 
			assignee.id, assignee.email
		FROM tickets t
		JOIN employees creator ON t.created_by_employee_id = creator.id
		LEFT JOIN employees assignee ON t.assigned_to_employee_id = assignee.id
		WHERE t.id = $1`

	database.Pool.QueryRow(context.Background(), query, ticketID).Scan(&creatorID, &creatorEmail, &assigneeID, &assigneeEmail)
	return
}

func sendNewCommentNotification(ticketID string, commenterName string) {
	creatorEmail, creatorID, assigneeEmail, assigneeID := getTicketStakeholders(ticketID)

	// Siapkan pesan notifikasi
	subject := fmt.Sprintf("Komentar Baru pada Tiket #%s", ticketID)
	body := fmt.Sprintf("Halo,<br><br>Ada komentar baru dari <b>%s</b> pada tiket #%s.<br><br>Silakan cek aplikasi untuk detailnya.", commenterName, ticketID)
	wsMessage := NotificationMessage{
		Type:     "NEW_COMMENT",
		TicketID: ticketID,
		Message:  fmt.Sprintf("%s menambahkan komentar baru pada tiket #%s", commenterName, ticketID),
	}
	jsonMsg, _ := json.Marshal(wsMessage)
	hub := websocket.GetHub()

	// Kirim email & websocket ke pembuat tiket
	if creatorEmail != "" {
		services.SendEmail(creatorEmail, subject, body)
		hub.SendToUser(creatorID, jsonMsg)
	}

	// Kirim email & websocket ke admin yang ditugaskan
	if assigneeEmail != "" && assigneeEmail != creatorEmail {
		services.SendEmail(assigneeEmail, subject, body)
		hub.SendToUser(assigneeID, jsonMsg)
	}
}

func sendStatusUpdateNotification(ticketID string, newStatus string) {
	creatorEmail, creatorID, _, assigneeID := getTicketStakeholders(ticketID)

	subject := fmt.Sprintf("Update Status pada Tiket #%s", ticketID)
	body := fmt.Sprintf("Halo,<br><br>Status tiket Anda #%s telah diperbarui menjadi: <b>%s</b>.<br><br>Silakan cek aplikasi untuk detailnya.", ticketID, newStatus)
	wsMessage := NotificationMessage{
		Type:     "STATUS_UPDATE",
		TicketID: ticketID,
		Message:  fmt.Sprintf("Status tiket #%s diubah menjadi %s", ticketID, newStatus),
	}
	jsonMsg, _ := json.Marshal(wsMessage)
	hub := websocket.GetHub()

	if creatorEmail != "" {
		services.SendEmail(creatorEmail, subject, body)
		hub.SendToUser(creatorID, jsonMsg)
	}
	if assigneeID != 0 && assigneeID != creatorID {
		hub.SendToUser(assigneeID, jsonMsg)
	}
}
