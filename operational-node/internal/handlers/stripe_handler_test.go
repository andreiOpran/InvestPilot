package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stripe/stripe-go/v85"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func setupStripeRouter(ss *servicemocks.MockStripeService, us *servicemocks.MockUserService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewStripeHandler(ss, us)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.POST("/deposit/intent", handler.CreateIntentHandler)
	r.POST("/webhook/stripe", handler.WebhookHandler)
	return r
}

func TestCreateIntentHandler(t *testing.T) {
	t.Run("CreateIntentHandler_badJSON_returns400", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		req, _ := http.NewRequest(http.MethodPost, "/deposit/intent", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("CreateIntentHandler_getUserProfileError_returns500", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		payload := models.DepositIntentRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		us.On("GetUserProfile", uint(1)).Return((*models.User)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/deposit/intent", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		us.AssertExpectations(t)
	})

	t.Run("CreateIntentHandler_createIntentError_returns500", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		payload := models.DepositIntentRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		user := &models.User{Email: "user@test.com"}
		us.On("GetUserProfile", uint(1)).Return(user, nil).Once()
		ss.On("CreatePaymentIntent", uint(1), 100.0, "user@test.com").Return("", services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/deposit/intent", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		us.AssertExpectations(t)
		ss.AssertExpectations(t)
	})

	t.Run("CreateIntentHandler_success_returns200WithClientSecret", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		payload := models.DepositIntentRequest{Amount: 50.0}
		body, _ := json.Marshal(payload)
		user := &models.User{Email: "user@test.com"}
		us.On("GetUserProfile", uint(1)).Return(user, nil).Once()
		ss.On("CreatePaymentIntent", uint(1), 50.0, "user@test.com").Return("pi_secret_abc123", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/deposit/intent", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "pi_secret_abc123")
		us.AssertExpectations(t)
		ss.AssertExpectations(t)
	})
}

func TestWebhookHandler(t *testing.T) {
	t.Run("WebhookHandler_invalidSignature_returns400", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		payload := []byte(`{"type":"payment_intent.succeeded"}`)
		ss.On("VerifyWebhookSignature", payload, "bad-sig").Return(stripe.Event{}, services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Stripe-Signature", "bad-sig")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		ss.AssertExpectations(t)
	})

	t.Run("WebhookHandler_nonPaymentEvent_returns200", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		payload := []byte(`{}`)
		event := stripe.Event{Type: "customer.created"}
		ss.On("VerifyWebhookSignature", payload, "sig").Return(event, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Stripe-Signature", "sig")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ss.AssertExpectations(t)
	})

	t.Run("WebhookHandler_paymentSucceeded_noUserID_returns200", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		piJSON, _ := json.Marshal(stripe.PaymentIntent{
			Amount: 1000,
			ID:     "pi_test",
		})
		payload := []byte(`{}`)
		event := stripe.Event{
			Type: "payment_intent.succeeded",
			Data: &stripe.EventData{Raw: piJSON},
		}
		ss.On("VerifyWebhookSignature", payload, "sig").Return(event, nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Stripe-Signature", "sig")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ss.AssertExpectations(t)
	})

	t.Run("WebhookHandler_paymentSucceeded_processError_returns500", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		piJSON, _ := json.Marshal(stripe.PaymentIntent{
			Amount:   5000,
			ID:       "pi_test_abc",
			Metadata: map[string]string{"user_id": "1"},
		})
		payload := []byte(`{}`)
		event := stripe.Event{
			Type: "payment_intent.succeeded",
			Data: &stripe.EventData{Raw: piJSON},
		}
		ss.On("VerifyWebhookSignature", payload, "sig").Return(event, nil).Once()
		us.On("ProcessWebhookDeposit", uint(1), int64(5000), "pi_test_abc").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Stripe-Signature", "sig")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		ss.AssertExpectations(t)
		us.AssertExpectations(t)
	})

	t.Run("WebhookHandler_paymentSucceeded_success_returns200", func(t *testing.T) {
		ss := new(servicemocks.MockStripeService)
		us := new(servicemocks.MockUserService)
		r := setupStripeRouter(ss, us)

		piJSON, _ := json.Marshal(stripe.PaymentIntent{
			Amount:   2000,
			ID:       "pi_success_xyz",
			Metadata: map[string]string{"user_id": "1"},
		})
		payload := []byte(`{}`)
		event := stripe.Event{
			Type: "payment_intent.succeeded",
			Data: &stripe.EventData{Raw: piJSON},
		}
		ss.On("VerifyWebhookSignature", payload, "sig").Return(event, nil).Once()
		us.On("ProcessWebhookDeposit", uint(1), int64(2000), "pi_success_xyz").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/webhook/stripe", bytes.NewBuffer(payload))
		req.Header.Set("Stripe-Signature", "sig")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		ss.AssertExpectations(t)
		us.AssertExpectations(t)
	})
}
