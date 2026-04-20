package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/andreiOpran/licenta/operational-node/utils/cookie"
	"github.com/andreiOpran/licenta/operational-node/utils/validator"
)

type AuthHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterHandler handles user registration
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.RegisterUser(req)
	if err != nil {
		if validator.IsPasswordValidationError(err) {
			// return specific password requirement that was not met
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
func (h *AuthHandler) VerifyEmailHandler(c *gin.Context) {
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	err := h.authService.VerifyEmail(tokenString)
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
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authService.AuthenticateUser(req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		if errors.Is(err, services.ErrAccountLocked) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// if the user has 2fa enabled, stop and tell client to prompt for code
	if result.Requires2FA {
		c.JSON(http.StatusOK, gin.H{
			"status":  "2fa_required",
			"message": "Please submit your TOTP code.",
			"email":   result.Email,
		})
		return
	}

	// set refresh token as HttpOnly cookie
	cookie.SetHttpOnly(
		c,                   // context
		"refresh_token",     // name
		result.RefreshToken, // value
		config.Env.RefreshTokenLifetimeSecondsInt, // maxAge
		"/api/v1/refresh-token",                   // path
	)

	// frontend will store just the access token
	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"access_token": result.AccessToken,
	})
}

// LogoutHandler handles logout by revoking refresh token
func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	// read refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		// if cookie is missing (session is already deleted) log success
		cookie.Clear(c, "refresh_token", "/api/v1/refresh-token")
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
		return
	}

	// revoke session
	_ = h.authService.LogoutUser(refreshToken)

	cookie.Clear(c, "refresh_token", "/api/v1/refresh-token")
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Verify2FAHandler verifies TOTP and generates session tokens
func (h *AuthHandler) Verify2FAHandler(c *gin.Context) {
	var req models.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accessToken, refreshToken, err := h.authService.Verify2FA(req.Email, req.Password, req.Token, c.ClientIP(), c.Request.UserAgent())
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

	// set refresh token as HttpOnly cookie
	cookie.SetHttpOnly(
		c,               // context
		"refresh_token", // name
		refreshToken,    // value
		config.Env.RefreshTokenLifetimeSecondsInt, // maxAge
		"/api/v1/refresh-token",                   // path
	)

	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"access_token": accessToken,
	})
}

// RefreshTokenHandler rotates refresh token
func (h *AuthHandler) RefreshTokenHandler(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
		return
	}

	accessToken, refreshToken, err := h.authService.RefreshToken(req.RefreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, services.ErrTokenInvalid) || errors.Is(err, services.ErrTokenExpired) {
			cookie.Clear(c, "refresh_token", "/api/v1/refresh-token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token. Please log in again."})
			return
		}
		if errors.Is(err, services.ErrTokenReuseDetected) {
			cookie.Clear(c, "refresh_token", "/api/v1/refresh-token")
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

	// rotate token, set new refresh token as a new cookie
	cookie.SetHttpOnly(
		c,               // context
		"refresh_token", // name
		refreshToken,    // value
		config.Env.RefreshTokenLifetimeSecondsInt, // maxAge
		"/api/v1/refresh-token",                   // path
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// ForgotPasswordHandler handles generating recovery token and sending email
func (h *AuthHandler) ForgotPasswordHandler(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.authService.ForgotPassword(req.Email)

	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

// ResetPasswordHandler processes reset password requests
func (h *AuthHandler) ResetPasswordHandler(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrTokenInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired recovery token"})
			return
		}
		if errors.Is(err, services.ErrTokenExpired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Recovery token has expired"})
			return
		}
		if validator.IsPasswordValidationError(err) {
			// return specific password requirement that was not met
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete reset process"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password successfully reset. You can now log in with your new password."})
}
