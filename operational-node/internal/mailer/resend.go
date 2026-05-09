package mailer

import (
	"fmt"

	resend "github.com/resend/resend-go/v2"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
)

// ResendEmailer implements EmailSender using the Resend API.
type ResendEmailer struct {
	client *resend.Client
	from   string
}

func NewResendEmailer() *ResendEmailer {
	return &ResendEmailer{
		client: resend.NewClient(config.Env.ResendAPIKey),
		from:   config.Env.ResendFrom,
	}
}

func (r *ResendEmailer) SendEmail(to, subject, body string) error {
	params := &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}
	_, err := r.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("resend: failed to send email: %w", err)
	}
	return nil
}
