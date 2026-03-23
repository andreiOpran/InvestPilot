package main

import (
	"fmt"
	"net/smtp"
	"os"
)

// EmailSender inteerface for sending emails, allowing dependency inejection
// we use SMTP in dev, and SendGrid in prod
type EmailSender interface {
	SendEmail(to string, subject string, body string) error
}

// SMTPEmailer implements EmailSender interface using GO's net/stmp package
type SMTPEmailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (s *SMTPEmailer) SendEmail(to string, subject string, body string) error {
	// build message mased on SMTP protocol format
	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+"Subject: %s\r\n"+"\r\n"+"%s\r\n",
		to, subject, body,
	))

	// setup auth
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	address := fmt.Sprintf("%s:%s", s.Host, s.Port)

	err := smtp.SendMail(address, auth, s.From, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("Failed to send email: %w", err)
	}

	return nil
}

// read env vars and return SMTP emailer
func NewSMTPConfig() *SMTPEmailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "smtp.gmail.com" // Default to gmail
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = os.Getenv("SMTP_USER")
	}

	return &SMTPEmailer{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		From:     from,
	}
}

var emailClient EmailSender

func initEmailer() {
	emailClient = NewSMTPConfig()
}
