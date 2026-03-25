package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
)

// EmailSender interface for sending emails, allowing dependency injection
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
	return &SMTPEmailer{
		Host:     config.Env.SMTPHost,
		Port:     config.Env.SMTPPort,
		Username: config.Env.SMTPUser,
		Password: config.Env.SMTPPass,
		From:     config.Env.SMTPFrom,
	}
}

var Client EmailSender

func InitEmailer() {
	Client = NewSMTPConfig()
}
