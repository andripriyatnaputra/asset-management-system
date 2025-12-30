package services

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
)

type TicketNotification struct {
	ToEmail string
	Subject string
	Message string
}

// Simple email sender (you can replace with SendGrid/Mailgun later)
func SendEmailNotification(ctx context.Context, n TicketNotification) {
	from := os.Getenv("SMTP_FROM")
	pass := os.Getenv("SMTP_PASS")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", from, pass, host)

	body := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", n.ToEmail, n.Subject, n.Message)
	if err := smtp.SendMail(addr, auth, from, []string{n.ToEmail}, []byte(body)); err != nil {
		log.Printf("[MAIL] failed to send to %s: %v\n", n.ToEmail, err)
	}
}

// Notify on new ticket creation
func NotifyTicketCreated(ticketID int64) {
	go func() {
		var email string
		err := database.Pool.QueryRow(context.Background(),
			`SELECT e.email
			   FROM employees e
			  WHERE e.role IN ('it_support','super_admin')
			  ORDER BY e.id ASC
			  LIMIT 1`,
		).Scan(&email)
		if err == nil && email != "" {
			SendEmailNotification(context.Background(), TicketNotification{
				ToEmail: email,
				Subject: fmt.Sprintf("Tiket #%d baru dibuat", ticketID),
				Message: fmt.Sprintf("Tiket #%d telah dibuat dan menunggu penanganan di portal ITSM.", ticketID),
			})
		}
	}()
}
