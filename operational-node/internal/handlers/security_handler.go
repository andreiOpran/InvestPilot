package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// Setup2FAHandler generates TOTP secret and QR code
func Setup2FAHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	secret, uri, qrCodeB64, err := services.Setup2FA(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if errors.Is(err, services.Err2FAAlreadyEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled for this account"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secret": secret,
		"uri":    uri, // app deep link (otpauth://...)
		// send qr codes as base63 string for easy frontend rendering
		"qr_code_b64": "data:image/png;base64," + qrCodeB64,
	})
}

// Enable2FAHandler confirms TOTP and enables 2FA for user
func Enable2FAHandler(c *gin.Context) {
	var req models.Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)
	err := services.Enable2FA(userID, req.Token)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		if errors.Is(err, services.Err2FAAlreadyEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
			return
		}
		if errors.Is(err, services.ErrInvalid2FAToken) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code. 2FA not enabled."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "2FA successfully enabled"})
}
