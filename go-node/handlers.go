package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
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
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
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
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// helper for generating JWT
func generateSessionToken(userID uint) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
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

			// if the use has 2FA enabled, stop and tell client to prompt for code
			if user.IsTwoFactorEnable {
				c.JSON(http.StatusOK, gin.H{
					"status":  "2fa_required",
					"message": "Please submit your TOTP code.",
					"email":   user.Email,
				})
				return
			}

			// if 2FA is not enabled, in log normally
			tokenString, err := generateSessionToken(user.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"token": tokenString,
			})
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
			tokenString, err := generateSessionToken(user.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status": "success",
				"token":  tokenString,
			})
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
