package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/bcrypt"
)

var encryptionKey = os.Getenv("AES_MASTER_KEY")

// encrypt plain string into hex-encoded string useing AES-GCM
func EncryptAES(plainText string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

// decrypt a hex-encoded cipher string back to plain text using AES-GCM
func DecryptAES(encryptedText string, key []byte) (string, error) {
	cipherText, err := hex.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherText) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherBytes := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}

// helper for generating random string
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// helper for generating short-lived Access JWT and a long lived Refresh Token
func generateTokensAndSession(c *gin.Context, userID uint) (string, string, error) {
	// generate short-lived JWT (10 minutes)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// generate long-lived refresh token (consists of random hex available for 7 days)
	refreshToken, err := generateSecureToken(32)
	if err != nil {
		return "", "", err
	}

	// generate familyid for the token
	familyID, err := generateSecureToken(16)
	if err != nil {
		return "", "", err
	}

	// save session to database
	session := Session{
		UserID:       userID,
		FamilyID:     familyID,
		RefreshToken: refreshToken,
		IsUsed:       false,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	if err := DB.Create(&session).Error; err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func StartTokenCleanupJob() {
	// init scheduler
	c := cron.New()

	// "0 3 * * *" - minute 0, hour 3, every day, every month, every day of the week
	_, err := c.AddFunc("0 3 * * *", func() {
		log.Println("[CRON JOB 03:00 AM] Cleaning up expired tokens...")
		now := time.Now()

		// clean ActionTokens (few, so we use regular deletion)
		res1 := DB.Where("expires_at < ?", now).Delete(&ActionToken{})
		if res1.Error != nil {
			log.Printf("[CRON JOB 03:00 AM] Error deleting ActionTokens: %v\n", res1.Error)
		}
		if res1.RowsAffected > 0 {
			log.Printf("[CRON JOB 03:00 AM] Deleted %d expired ActionTokens.\n", res1.RowsAffected)
		} else {
			log.Println("[CRON JOB 03:00 AM] No expired ActionTokens found for deletion.")
		}

		// clean expired sessions using batching (big count of sessions, compared to the ActionTokens)
		var totalDeleted int64
		batchSize := 1000

		for {
			// we can use DELETE w/ LIMIT, so we retrieve 1000 ids,
			// and then delete the sessions with ids that are in that set
			subQuery := DB.Table("sessions").Select("id").Where("expires_at < ?", now).Limit(batchSize)

			res2 := DB.Where("id IN (?)", subQuery).Delete(&Session{})

			if res2.Error != nil {
				log.Printf("[CRON JOB 03:00 AM] Error deleting Sessions (Batch): %v\n", res2.Error)
				break
			}

			rowsAffected := res2.RowsAffected
			totalDeleted += res2.RowsAffected

			// if we deleted less than 1000, means we are done
			if rowsAffected < int64(batchSize) {
				break
			}

			// sleep for a bit to let the db receive requests from the users
			time.Sleep(100 * time.Millisecond)
		}

		if totalDeleted > 0 {
			log.Printf("[CRON JOB 03:00 AM] Deleted %d expired Sessions (Batch).\n", totalDeleted)
		} else {
			log.Println("[CRON JOB 03:00 AM] No expired Sessions found for deletion.")
		}
	})

	if err != nil {
		log.Fatalf("Error initializing CRON job: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] TokenCleanupJob scheduled successfully.")
}

func RegisterRoutes(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go node works"})
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "Server is running",
			"database": "Connected",
		})
	})

	r.GET("/test-email", func(c *gin.Context) {
		testEmail := os.Getenv("SMTP_TEST_DESTINATION")

		err := emailClient.SendEmail(
			testEmail,
			"Test",
			"Test for SMTP",
		)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Test email sent successfully"})
	})

	v1 := r.Group("/api/v1")
	{
		// endpoint that shows vpc communication
		v1.POST("/simulate-investment", func(c *gin.Context) {
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
		})

		v1.POST("/register", func(c *gin.Context) {
			var req RegisterRequest

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
			var existingUser User
			userExists := false
			if err := DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
				userExists = true
			}

			// if user exists, pretend registration was successful to avoid user enumeration
			if userExists {
				// generate dummy token to simulate time taken by rand ops
				_, _ = generateSecureToken(32)

				// same success response as real registration
				c.JSON(http.StatusOK, gin.H{
					"message": "If the email is valid, a verification link has been sent.",
				})
				return
			}

			// if user does not exist, procees with creation
			// build user with an empty wallet and IsEmailVerified=false
			user := User{
				Email:             req.Email,
				Password:          string(hashedPassword),
				RiskTolerance:     req.RiskTolerance,
				InvestmentHorizon: req.InvestmentHorizon,
				Wallet:            Wallet{Balance: 0.0},
			}

			// save to DB (will fail if email already exists)
			if err := DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
				return
			}

			// generate ActionToken for email verification
			verificationToken, err := generateSecureToken(32)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate verification token"})
				return
			}

			actionToken := ActionToken{
				UserID:    user.ID,
				Token:     verificationToken,
				Type:      "verify_email",
				ExpiresAt: time.Now().Add(24 * time.Hour), // time available to verify
			}

			// save ActionToken for email verification to database
			if err := DB.Create(&actionToken).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save verification token"})
				return
			}

			// send email
			// TODO, in prod get BaseURL from env
			verificationURL := fmt.Sprintf("http://localhost:8080/api/v1/verify-email?token=%s", verificationToken)
			emailBody := fmt.Sprintf("Welcome to Robo-Advisory application.\n\nPlease verify your email clicking the link below:\n%s\n\nNote: link expires in 24 hours.", verificationURL)

			// send email in goroutine so SMTP server network latency does not affect API response time
			go func() {
				if err := emailClient.SendEmail(user.Email, "Verify Your Robo-Advisory Account", emailBody); err != nil {
					fmt.Printf("Failed to send verification email to %s: %v\n", user.Email, err)
				}
			}()

			c.JSON(http.StatusCreated, gin.H{
				"message": "If the email is valid, a verification link has been sent.",
			})
		})

		v1.GET("/verify-email", func(c *gin.Context) {
			tokenString := c.Query("token")
			if tokenString == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
				return
			}

			var actionToken ActionToken

			// find token and preload user
			if err := DB.Where("token = ? AND type = ?", tokenString, "verify_email").First(&actionToken).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
				return
			}

			// check expiration
			if time.Now().After(actionToken.ExpiresAt) {
				// cleanup expired token
				DB.Delete(&actionToken)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
				return
			}

			// transaction - update user and delete token
			tx := DB.Begin()

			// update user to verified
			if err := tx.Model(&User{}).Where("id = ?", actionToken.UserID).Update("is_email_verified", true).Error; err != nil {
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
		})

		v1.POST("/login", func(c *gin.Context) {
			var req LoginRequest

			// validate incoming json
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// look up user by email
			var user User
			userExists := true
			if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
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
			accessToken, refreshToken, err := generateTokensAndSession(c, user.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":        "success",
				"access_token":  accessToken,
				"refresh_token": refreshToken,
			})
		})

		v1.POST("/logout", func(c *gin.Context) {
			var req RefreshRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required for logout"})
				return
			}

			// delete session from the db (access token is nto deleted because it has short lifetime)
			if err := DB.Where("refresh_token = ?", req.RefreshToken).Delete(&Session{}).Error; err != nil {
				// return succes even if we have an error here
				// client will clear local state anyway
			}

			c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
		})

		v1.POST("/verify-2fa", func(c *gin.Context) {
			var req Verify2FARequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}

			// re-authenticate user (stateless flow)
			var user User
			if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
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

			plainSecret, err := DecryptAES(user.TwoFactorSecret, []byte(encryptionKey))
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
			accessToken, refreshToken, err := generateTokensAndSession(c, user.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status":        "success",
				"access_token":  accessToken,
				"refresh_token": refreshToken,
			})
		})

		v1.POST("/refresh-token", func(c *gin.Context) {
			var req RefreshRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
				return
			}

			// look up session in the DB, preload user to make sure he hasn't been deleted
			var session Session
			if err := DB.Preload("User").Where("refresh_token = ?", req.RefreshToken).First(&session).Error; err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
				return
			}
			// remember the last UpdatedAt when we retrieve the session, to prevent race conditions (optimistic locking)
			originalUpdatedAt := session.UpdatedAt

			// token reuse detection
			// if someone tries to use a token that has already been changed by the legitimate user, we invalidate all sessions
			if session.IsUsed {
				DB.Where("family_id = ?", session.FamilyID).Delete(&Session{})

				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Security Alert: Token reuse detected. To protect your account, all devices have been logged out.",
				})
				return
			}

			// check expiration
			if time.Now().After(session.ExpiresAt) {
				// cleanup expired session
				DB.Delete(&session)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh session expired. Please log in again."})
				return
			}

			// refresh token rotation with optimistic concurrency control
			// mark current token as being used, and keep it in the db as a trap for potential attackers
			// only if it hasn't been modified by another conucrrent request
			result := DB.Model(&Session{}).
				Where("id = ? AND updated_at = ?", session.ID, originalUpdatedAt).
				Update("is_used", true)
			// if 0 rows were affected, means another request just updated this token
			if result.RowsAffected == 0 {
				c.JSON(http.StatusConflict, gin.H{"error": "Concurrent request detected. Please try again."})
				return
			}

			// generate new refresh token
			newRefreshToken, err := generateSecureToken(32)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
				return
			}

			// send new 10 minute access token
			claims := Claims{
				UserID: session.UserID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}
			newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			newAccessToken, err := newToken.SignedString(jwtSecret)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate new token"})
				return
			}

			// save new token, with the same familyid as the previous one
			newSession := Session{
				UserID:       session.UserID,
				FamilyID:     session.FamilyID, // same faimlyid as the previous session
				RefreshToken: newRefreshToken,
				IsUsed:       false,
				ClientIP:     c.ClientIP(),
				UserAgent:    c.Request.UserAgent(),
				ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
			}
			DB.Create(&newSession)

			c.JSON(http.StatusOK, gin.H{
				"access_token":  newAccessToken,
				"refresh_token": newRefreshToken,
			})
		})

		v1.POST("/forgot-password", func(c *gin.Context) {
			// record actual logic computing time to standardize response times to avoid timing attacks
			startTime := time.Now()

			var req ForgotPasswordRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// look up user
			var user User
			userExists := true
			if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
				userExists = false
			}

			// generate recovery token, even if user is not found, to combat timing attacks
			recoveryToken, err := generateSecureToken(32)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}

			if userExists && user.IsEmailVerified {
				// generate and save action token
				actionToken := ActionToken{
					UserID:    user.ID,
					Token:     recoveryToken,
					Type:      "reset_password",
					ExpiresAt: time.Now().Add(15 * time.Minute),
				}

				if err := DB.Create(&actionToken).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate recovery process"})
					return
				}

				// send recovery email
				// TODO: In prod, get BaseURL from env
				recoveryURL := fmt.Sprintf("http://localhost:8081/reset-password?token=%s", recoveryToken)
				emailBody := fmt.Sprintf("You requested a password reset for your Robo-Advisory account.\n\nPlease click the link below to set a new password:\n%s\n\nThis link expires in 15 minutes. If you did not request this, please ignore this email.", recoveryURL)

				// send email in goroutine so SMTP server network latency does not affect API response time
				go func() {
					if err := emailClient.SendEmail(user.Email, "Robo-Advisory Password Reset", emailBody); err != nil {
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
			noise := time.Duration(mathrand.Intn(20)) * time.Millisecond

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
		})

		v1.POST("/reset-password", func(c *gin.Context) {
			var req ResetPasswordRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			var actionToken ActionToken

			// find token and check type
			if err := DB.Where("token = ? AND type = ?", req.Token, "reset_password").First(&actionToken).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired recovery token"})
				return
			}

			// check expiration
			if time.Now().After(actionToken.ExpiresAt) {
				// cleanup expired token
				DB.Delete(&actionToken)
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
			tx := DB.Begin()
			if err := tx.Model(&User{}).Where("id = ?", actionToken.UserID).Update("password", string(hashedPassword)).Error; err != nil {
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
			if err := tx.Where("user_id = ?", actionToken.UserID).Delete(&Session{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not invalidate old sessions"})
				return
			}

			tx.Commit()

			c.JSON(http.StatusOK, gin.H{"message": "Password successfully reset. You can now log in with your new password."})
		})

		// protected: JWT required for all routes inside
		protected := v1.Group("/", authMiddleware())
		{
			protected.GET("/user", func(c *gin.Context) {
				var user User
				userID := c.MustGet("userID").(uint)

				// Preload("Wallet") tells GORM to also fetch the attached Wallet data
				if err := DB.Preload("Wallet").First(&user, userID).Error; err != nil {
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
			})

			protected.GET("/2fa/setup", func(c *gin.Context) {
				userID := c.MustGet("userID").(uint)
				var user User
				if err := DB.First(&user, userID).Error; err != nil {
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

				encryptedSecret, err := EncryptAES(key.Secret(), []byte(encryptionKey))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt secret"})
					return
				}
				// temp save secret (user must confirm it to enable)
				user.TwoFactorSecret = encryptedSecret
				DB.Save(&user)

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
			})

			// confirm that token to permanently enable 2FA
			protected.POST("/2fa/enable", func(c *gin.Context) {
				var req Enable2FARequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				userID := c.MustGet("userID").(uint)
				var user User
				if err := DB.First(&user, userID).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
					return
				}

				if user.IsTwoFactorEnable {
					c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is already enabled"})
					return
				}

				plainSecret, err := DecryptAES(user.TwoFactorSecret, []byte(encryptionKey))
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
				DB.Save(&user)

				c.JSON(http.StatusOK, gin.H{"message": "2FA successfully enabled"})
			})

			protected.POST("/deposit", func(c *gin.Context) {
				var req DepositRequst
				userID := c.MustGet("userID").(uint)

				// 1. read and validate the JSON body from the request
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid amount greater than 0"})
					return
				}

				var user User
				// 2. find the authenticated user and their attached wallet
				if err := DB.Preload("Wallet").First(&user, userID).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
					return
				}

				// 3. add simulated money to the wallet
				user.Wallet.Balance += req.Amount

				user.Wallet.UserId = user.ID

				// 4. save updated walet to the database
				DB.Save(&user.Wallet)

				// 5. send a succes response back
				c.JSON(http.StatusOK, gin.H{
					"message":     "Paper trading deposit successful.",
					"added":       req.Amount,
					"new_balance": user.Wallet.Balance,
				})
			})
		}
	}
}
