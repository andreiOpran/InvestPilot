package servicemocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v85"
)

type MockStripeService struct {
	mock.Mock
}

func (m *MockStripeService) CreatePaymentIntent(userID uint, amount float64, receiptEmail string) (string, error) {
	args := m.Called(userID, amount, receiptEmail)
	return args.String(0), args.Error(1)
}

func (m *MockStripeService) VerifyWebhookSignature(payload []byte, signature string) (stripe.Event, error) {
	args := m.Called(payload, signature)
	return args.Get(0).(stripe.Event), args.Error(1)
}
