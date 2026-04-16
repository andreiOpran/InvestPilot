package services

import (
	"fmt"
	"strconv"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/paymentintent"
	"github.com/stripe/stripe-go/v85/webhook"
)

type StripeService interface {
	CreatePaymentIntent(userID uint, amount float64) (string, error)
	VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error)
}

type stripeService struct{}

func NewStripeService() StripeService {
	stripe.Key = config.Env.StripeKey
	return &stripeService{}
}

func (s *stripeService) CreatePaymentIntent(userID uint, amount float64) (string, error) {
	// stripe expects cents, so we multiply by 100
	amountCents := int64(amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Metadata: map[string]string{
			"user_id": strconv.FormatUint(uint64(userID), 10), // for mapping webhooks back to user
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create payment intent: %w", err)
	}

	// client secret goes to the frontend Elements widget
	return pi.ClientSecret, nil
}

func (s *stripeService) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, signature, config.Env.StripeWebhookKey)
}
