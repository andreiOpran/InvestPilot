package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v85"
)

type StripeHandler struct {
	stripeService services.StripeService
	userService   services.UserService
}

func NewStripeHandler(ss services.StripeService, us services.UserService) *StripeHandler {
	return &StripeHandler{stripeService: ss, userService: us}
}

// POST /deposit/intent (protected)
func (h *StripeHandler) CreateIntentHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var req models.DepositIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	user, err := h.userService.GetUserProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize payment gateway"})
		return
	}

	clientSecret, err := h.stripeService.CreatePaymentIntent(userID, req.Amount, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize payment gateway"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"client_secret": clientSecret})
}

// POST /webhook/stripe (unprotected)
func (h *StripeHandler) WebhookHandler(c *gin.Context) {
	const MaxBodyBytes = int64(65536) // stripe suggest 65KB max bound
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error reading request body"})
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	event, err := h.stripeService.VerifyWebhookSignature(payload, sigHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature"})
		return
	}

	// route the different events type
	if event.Type == "payment_intent.succeeded" {
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Error parsing webhook event"})
			return
		}

		// extract metadata "user_id"
		userIDStr, ok := paymentIntent.Metadata["user_id"]
		if !ok {
			// not really error, could be test payment, just ignore and 200 OK
			c.Status(http.StatusOK)
			return
		}

		userID, _ := strconv.ParseUint(userIDStr, 10, 32)

		// atomically log and fund user wallet
		if err := h.userService.ProcessWebhookDeposit(uint(userID), paymentIntent.Amount, paymentIntent.ID); err != nil {
			// if it fails, returning 500 will tell stripe to retry the webhook later
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process funding"})
			return
		}
	}
	c.Status(http.StatusOK)
}
