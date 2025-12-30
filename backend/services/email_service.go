// File: backend/services/email_service.go
package services

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/andripriyatnaputra/asset-management-system/backend/database"
	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string) {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	smtpUser := os.Getenv("SMTP_USERNAME")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	smtpSender := os.Getenv("SMTP_SENDER")

	m := gomail.NewMessage()
	m.SetHeader("From", smtpSender)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)

	// Kirim email
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Could not send email to %s: %v", to, err)
		database.Pool.Exec(context.Background(),
			`INSERT INTO email_logs (recipient, subject, body_preview, status, error_message)
   			VALUES ($1,$2,LEFT($3,200),'FAILED',$4)`,
			to, subject, body, err.Error())

	} else {
		log.Printf("Email sent successfully to %s", to)
		// Tambahkan ke dalam email_service.go setelah log.Printf(...)
		database.Pool.Exec(context.Background(),
			`INSERT INTO email_logs (recipient, subject, body_preview, status, error_message)
  			 VALUES ($1,$2,LEFT($3,200),'SENT',NULL)`,
			to, subject, body)
	}
}
