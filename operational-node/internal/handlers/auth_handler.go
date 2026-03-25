package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
	"github.com/andreiOpran/licenta/operational-node/utils/token"
)

// RegisterHandler handles user registration
func RegisterHandler(c *gin.Context) {
	var req models.RegisterRequest

	// validate incoming json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// even if the user already exists, we do the heavy bcrypt hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	// check if the user exists
	var existingUser models.User
	userExists := false
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		userExists = true
	}

	// if user exists, pretend registration was successful to avoid user enumeration
	if userExists {
		// generate dummy token to simulate time taken by rand ops
		_, _ = token.GenerateSecureToken(32)

		// same success response as real registration
		c.JSON(http.StatusOK, gin.H{
			"message": "If the email is valid, a verification link has been sent.",
		})
		return
	}

	// if user does not exist, procees with creation
	// build user with an empty wallet and IsEmailVerified=false
	user := models.User{
		Email:             req.Email,
		Password:          string(hashedPassword),
		RiskTolerance:     req.RiskTolerance,
		InvestmentHorizon: req.InvestmentHorizon,
		Wallet:            models.Wallet{Balance: 0.0},
	}

	// save to DB (will fail if email already exists)
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// generate ActionToken for email verification
	verificationToken, err := token.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate verification token"})
		return
	}

	actionToken := models.ActionToken{
		UserID:    user.ID,
		Token:     verificationToken,
		Type:      "verify_email",
		ExpiresAt: time.Now().Add(config.Env.VerifyEmailLifetime), // time available to verify
	}

	// save ActionToken for email verification to database
	if err := database.DB.Create(&actionToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save verification token"})
		return
	}

	// send email
	// TODO, in prod get BaseURL from env
	verificationURL := fmt.Sprintf("http://localhost:8080/api/v1/verify-email?token=%s", verificationToken)
	emailBody := fmt.Sprintf("Welcome to Robo-Advisory application.\n\nPlease verify your email clicking the link below:\n%s\n\nNote: link expires in 24 hours.", verificationURL)

	// send email in goroutine so SMTP server network latency does not affect API response time
	go func() {
		if err := mailer.Client.SendEmail(user.Email, "Verify Your Robo-Advisory Account", emailBody); err != nil {
			fmt.Printf("Failed to send verification email to %s: %v\n", user.Email, err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
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

	var actionToken models.ActionToken

	// find token and preload user
	if err := database.DB.Where("token = ? AND type = ?", tokenString, "verify_email").First(&actionToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
		return
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		database.DB.Delete(&actionToken)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
		return
	}

	// transaction - update user and delete token
	tx := database.DB.Begin()

	// update user to verified
	if err := tx.Model(&models.User{}).Where("id = ?", actionToken.UserID).Update("is_email_verified", true).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete verification process"})
		return
	}

	// delete used token
	if err := tx.Delete(&actionToken).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete verification process"})
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Email successfully verified. You can now log in."})
}

// LoginHandler handles user login
func LoginHandler(c *gin.Context) {
	var req models.LoginRequest

	// validate incoming json
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// look up user by email
	var user models.User
	userExists := true
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		userExists = false
		// do not return here, continue to dummy bcrypt comparison to avoid timing attacks
	}

	// compare provided password against stored bcrypt hash, also dummy verifications for nonexistent user
	if userExists {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			// vague, do not reveal whether email exists
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		// check verification only is password is correct,
		// but return same vague error message to protect against enuemration
		if !user.IsEmailVerified {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid email or password"})
			return
		}
	} else {
		// dummy comparison
		// declared to not compute a random cost 14 hash
		const dummyBcryptHash = "$2a$14$1AB05scB8KFNDuDWpgvzkO6GYYf62uSGJr445WX6x2jHkWpcySpjW"
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(req.Password))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// if the user has 2FA enabled, stop and tell client to prompt for code
	if user.IsTwoFactorEnable {
		c.JSON(http.StatusOK, gin.H{
			"status":  "2fa_required",
			"message": "Please submit your TOTP code.",
			"email":   user.Email,
		})
		return
	}

	// if 2FA is not enabled, log in normally
	accessToken, refreshToken, err := token.GenerateTokensAndSession(c, user.ID, []byte(config.Env.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// LogoutHandler handles logout by revoking refresh token
func LogoutHandler(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required for logout"})
		return
	}

	// delete session from the db (access token is nto deleted because it has short lifetime)
	if err := database.DB.Where("refresh_token = ?", req.RefreshToken).Delete(&models.Session{}).Error; err != nil {
		// return succes even if we have an error here
		// client will clear local state anyway
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Verify2FAHandler verifies TOTP and generates session tokens
func Verify2FAHandler(c *gin.Context) {
	var req models.Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	// re-authenticate user (stateless flow)
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials or token"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials or token"})
		return
	}

	if !user.IsTwoFactorEnable {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled on this account"})
		return
	}

	plainSecret, err := crypto.DecryptAES(user.TwoFactorSecret, []byte(config.Env.AESMasterKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt secret"})
		return
	}

	// validate TOTP code
	valid := totp.Validate(req.Token, plainSecret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA token"})
		return
	}

	// generate session
	accessToken, refreshToken, err := token.GenerateTokensAndSession(c, user.ID, []byte(config.Env.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
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

	// look up session in the DB, preload user to make sure he hasn't been deleted
	var session models.Session
	if err := database.DB.Preload("User").Where("refresh_token = ?", req.RefreshToken).First(&session).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}
	// remember the last UpdatedAt when we retrieve the session, to prevent race conditions (optimistic locking)
	originalUpdatedAt := session.UpdatedAt

	// token reuse detection
	// if someone tries to use a token that has already been changed by the legitimate user, we invalidate all sessions
	if session.IsUsed {
		database.DB.Where("family_id = ?", session.FamilyID).Delete(&models.Session{})

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Security Alert: Token reuse detected. To protect your account, all devices have been logged out.",
		})
		return
	}

	// check expiration
	if time.Now().After(session.ExpiresAt) {
		// cleanup expired session
		database.DB.Delete(&session)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh session expired. Please log in again."})
		return
	}

	// refresh token rotation with optimistic concurrency control
	// mark current token as being used, and keep it in the db as a trap for potential attackers
	// only if it hasn't been modified by another conucrrent request
	result := database.DB.Model(&models.Session{}).
		Where("id = ? AND updated_at = ?", session.ID, originalUpdatedAt).
		Update("is_used", true)
	// if 0 rows were affected, means another request just updated this token
	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Concurrent request detected. Please try again."})
		return
	}

	// generate new refresh token
	newRefreshToken, err := token.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
		return
	}

	// send new 10 minute access token
	claims := models.Claims{
		UserID: session.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newAccessToken, err := newToken.SignedString([]byte(config.Env.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
		return
	}

	// save new token, with the same familyid as the previous one
	newSession := models.Session{
		UserID:       session.UserID,
		FamilyID:     session.FamilyID, // same faimlyid as the previous session
		RefreshToken: newRefreshToken,
		IsUsed:       false,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	database.DB.Create(&newSession)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// ForgotPasswordHandler handles generating recovery token and sending email
func ForgotPasswordHandler(c *gin.Context) {
	// record actual logic computing time to standardize response times to avoid timing attacks
	startTime := time.Now()

	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// look up user
	var user models.User
	userExists := true
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		userExists = false
	}

	// generate recovery token, even if user is not found, to combat timing attacks
	recoveryToken, err := token.GenerateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if userExists && user.IsEmailVerified {
		// generate and save action token
		actionToken := models.ActionToken{
			UserID:    user.ID,
			Token:     recoveryToken,
			Type:      "reset_password",
			ExpiresAt: time.Now().Add(config.Env.ResetPasswordLifetime),
		}

		if err := database.DB.Create(&actionToken).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate recovery process"})
			return
		}

		// send recovery email
		// TODO: In prod, get BaseURL from env
		recoveryURL := fmt.Sprintf("http://localhost:8081/reset-password?token=%s", recoveryToken)
		emailBody := fmt.Sprintf("You requested a password reset for your Robo-Advisory account.\n\nPlease click the link below to set a new password:\n%s\n\nThis link expires in 15 minutes. If you did not request this, please ignore this email.", recoveryURL)

		// send email in goroutine so SMTP server network latency does not affect API response time
		go func() {
			if err := mailer.Client.SendEmail(user.Email, "Robo-Advisory Password Reset", emailBody); err != nil {
				fmt.Printf("Failed to send recovery email to %s: %v\n", user.Email, err)
			}
		}()
	}

	// timing attack avoidance logic
	// stop timer to see how long it took to compute real logic
	elapsed := time.Since(startTime)
	// we set a target time of 100ms that all /forgot-password should achieve
	targetTime := 100 * time.Millisecond
	// generate random noise
	noise := time.Duration(rand.Intn(20)) * time.Millisecond

	// level actual response time with the target time
	if elapsed < targetTime {
		// the request was too fast, so we sleep until the target time + noise to prevent patterns
		time.Sleep((targetTime - elapsed) + noise)
	} else {
		// if we surpassed target time, still sleep a bit to prevent patterns
		time.Sleep(noise)
	}

	// return vague success message
	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

// ResetPasswordHandler processes reset password requests
func ResetPasswordHandler(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var actionToken models.ActionToken

	// find token and check type
	if err := database.DB.Where("token = ? AND type = ?", req.Token, "reset_password").First(&actionToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired recovery token"})
		return
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		database.DB.Delete(&actionToken)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recovery token has expired"})
		return
	}

	// hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not process new password"})
		return
	}

	// transaction - update user pass and delete token
	// transaction is used in case of token deletion failure, then the password will not be updated
	tx := database.DB.Begin()
	if err := tx.Model(&models.User{}).Where("id = ?", actionToken.UserID).Update("password", string(hashedPassword)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete reset process"})
		return
	}

	// delete used token, it should be used only once
	if err := tx.Delete(&actionToken).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete reset process"})
		return
	}

	// invalidate all existing sessions for this user
	if err := tx.Where("user_id = ?", actionToken.UserID).Delete(&models.Session{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not invalidate old sessions"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Password successfully reset. You can now log in with your new password."})
}
