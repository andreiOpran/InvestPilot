package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

type UserHandler struct {
	userService services.UserService
}

func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// TestEmailHandler triggers a test email send
func TestEmailHandler(c *gin.Context) {
	testEmail := config.Env.SMTPTestDestination
	err := mailer.Client.SendEmail(testEmail, "Test", "Test for SMTP")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Test email sent successfully"})
}

// GetUserHandler returns basic profile and wallet balance
func (h *UserHandler) GetUserHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	user, err := h.userService.GetUserProfile(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":            user.ID,
		"email":              user.Email,
		"risk_tolerance":     user.RiskTolerance,
		"investment_horizon": user.InvestmentHorizon,
		"wallet_balance":     user.Wallet.Balance,
	})
}

// UpdateProfileHandler processes the onboarding form for financial details
func (h *UserHandler) UpdateProfileHandler(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk tolerance (1-5) or investment horizon (1-50)"})
		return
	}

	userID := c.MustGet("userID").(uint)

	err := h.userService.UpdateUserProfile(userID, req)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Financial profile updated successfully."})
}

// DepositHandler adds simulated funds to user's wallet
func (h *UserHandler) DepositHandler(c *gin.Context) {
	var req models.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid amount greater than 0"})
		return
	}

	userID := c.MustGet("userID").(uint)
	newBalance, err := h.userService.DepositFunds(userID, req.Amount)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Paper trading deposit successful.",
		"added":       req.Amount,
		"new_balance": newBalance,
	})
}

// CashoutHandler extracts simulated funds from user's wallet
func (h *UserHandler) CashoutHandler(c *gin.Context) {
	var req models.CashoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)
	newBalance, err := h.userService.Cashout(userID, req.Amount)
	if err != nil {
		if errors.Is(err, services.ErrInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient wallet balance."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process withdrawal: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Withdrawal processed successfully",
		"new_balance": newBalance,
	})
}
