package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// RegisterHandler handles user registration
func RegisterHandler(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.RegisterUser(req)
	if err != nil {
		if errors.Is(err, services.ErrEmailExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// same success response as real registration
	c.JSON(http.StatusOK, gin.H{
		"message": "If the email is valid, a verification link has been sent.",
	})
}

// VerifyEmailHandler handles email verification via token
func VerifyEmailHandler(c *gin.Context) {
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	err := services.VerifyEmail(tokenString)
	if err != nil {
		if errors.Is(err, services.ErrTokenInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete verification process"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email successfully verified. You can now log in."})
}

// LoginHandler handles user login
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := services.AuthenticateUser(req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// if the user has 2FA enabled, stop and tell client to prompt for code
	if result.Requires2FA {
		c.JSON(http.StatusOK, gin.H{
			"status":  "2fa_required",
			"message": "Please submit your TOTP code.",
			"email":   result.Email,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

// LogoutHandler handles logout by revoking refresh token
func LogoutHandler(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required for logout"})
		return
	}

	_ = services.LogoutUser(req.RefreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Verify2FAHandler verifies TOTP and generates session tokens
func Verify2FAHandler(c *gin.Context) {
	var req models.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := services.Verify2FA(req.Email, req.Password, req.Token, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials or token"})
			return
		}
		if errors.Is(err, services.Err2FANotEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled on this account"})
			return
		}
		if errors.Is(err, services.ErrInvalid2FAToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA token"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RefreshTokenHandler rotates refresh token
func RefreshTokenHandler(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}

	accessToken, refreshToken, err := services.RefreshToken(req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, services.ErrTokenInvalid) || errors.Is(err, services.ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token. Please log in again."})
			return
		}
		if errors.Is(err, services.ErrTokenReuseDetected) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Security Alert: Token reuse detected. To protect your account, all devices have been logged out.",
			})
			return
		}
		if errors.Is(err, services.ErrConcurrentRequest) {
			c.JSON(http.StatusConflict, gin.H{"error": "Concurrent request detected. Please try again."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// ForgotPasswordHandler handles generating recovery token and sending email
func ForgotPasswordHandler(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = services.ForgotPassword(req.Email)

	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

// ResetPasswordHandler processes reset password requests
func ResetPasswordHandler(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := services.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrTokenInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired recovery token"})
			return
		}
		if errors.Is(err, services.ErrTokenExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Recovery token has expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete reset process"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password successfully reset. You can now log in with your new password."})
}
