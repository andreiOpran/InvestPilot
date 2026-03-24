package handlers

import (
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// PingHandler simple health check
func PingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Go node works"})
}

// StatusHandler reports rudimentary status
func StatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "Server is running",
		"database": "Connected",
	})
}

// TestEmailHandler triggers a test email send
func TestEmailHandler(c *gin.Context) {
	testEmail := os.Getenv("SMTP_TEST_DESTINATION")

	err := mailer.Client.SendEmail(
		testEmail,
		"Test",
		"Test for SMTP",
	)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Test email sent successfully"})
}

// SimulateInvestmentHandler proxies a request to python-engine service
func SimulateInvestmentHandler(c *gin.Context) {
	// make a request to the py container using the name of the service from docker-compose
	resp, err := http.Post("http://python-engine:5000/generate-models", "application/json", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error commincating with Py node"})
		return
	}
	// close the response body to avoid memory leaks
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// forward response to frontend
	c.Data(http.StatusOK, "application/json", body)
}

// GetUserHandler returns basic profile and wallet balance
func GetUserHandler(c *gin.Context) {
	var user models.User
	userID := c.MustGet("userID").(uint)

	// Preload("Wallet") tells GORM to also fetch the attached Wallet data
	if err := database.DB.Preload("Wallet").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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

// DepositHandler adds simulated funds to user's wallet
func DepositHandler(c *gin.Context) {
	var req models.DepositRequest
	userID := c.MustGet("userID").(uint)

	// 1. read and validate the JSON body from the request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid amount greater than 0"})
		return
	}

	var user models.User
	// 2. find the authenticated user and their attached wallet
	if err := database.DB.Preload("Wallet").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 3. add simulated money to the wallet
	user.Wallet.Balance += req.Amount

	user.Wallet.UserId = user.ID

	// 4. save updated walet to the database
	database.DB.Save(&user.Wallet)

	// 5. send a succes response back
	c.JSON(http.StatusOK, gin.H{
		"message":     "Paper trading deposit successful.",
		"added":       req.Amount,
		"new_balance": user.Wallet.Balance,
	})
}
