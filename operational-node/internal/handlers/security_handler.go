package handlers

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"

	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
)

// Setup2FAHandler generates TOTP secret and QR code
func Setup2FAHandler(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.IsTwoFactorEnable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled for this account"})
		return
	}

	// generate OTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Robo-Advisory",
		AccountName: user.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	encryptedSecret, err := crypto.EncryptAES(key.Secret(), []byte(crypto.EncryptionKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt secret"})
		return
	}
	// temp save secret (user must confirm it to enable)
	user.TwoFactorSecret = encryptedSecret
	database.DB.Save(&user)

	// generate QR code image
	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	png.Encode(&buf, img)

	b64String := base64.StdEncoding.EncodeToString(buf.Bytes())
	c.JSON(http.StatusOK, gin.H{
		"secret": key.Secret(),
		"uri":    key.URL(), // app deep link (otpauth://...)
		// send qr codes as base63 string for easy frontend rendering
		// frontend: <img src="data:image/png;base64,+b64String"/>
		"qr_code_b64": "data:image/png;base64," + b64String,
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
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.IsTwoFactorEnable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
		return
	}

	plainSecret, err := crypto.DecryptAES(user.TwoFactorSecret, []byte(crypto.EncryptionKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt secret"})
		return
	}

	// validate the code agains the secret we saved during /setup
	valid := totp.Validate(req.Token, plainSecret)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code. 2FA not enabled."})
		return
	}

	user.IsTwoFactorEnable = true
	database.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{"message": "2FA successfully enabled"})
}
