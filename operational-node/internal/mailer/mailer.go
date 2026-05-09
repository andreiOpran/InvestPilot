package mailer

import "github.com/andreiOpran/licenta/operational-node/internal/config"

// EmailSender interface for sending emails, allowing dependency injection.
// Provider is selected at startup via EMAIL_PROVIDER env var ("smtp" or "resend").
type EmailSender interface {
	SendEmail(to string, subject string, body string) error
}

var Client EmailSender

func InitEmailer() {
	if config.Env.EmailProvider == "resend" {
		Client = NewResendEmailer()
	} else {
		Client = NewSMTPConfig()
	}
}
