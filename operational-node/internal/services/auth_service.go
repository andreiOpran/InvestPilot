package services

import (
	"fmt"
	"math/rand"
	"time"

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

type LoginResult struct {
	Requires2FA  bool
	Email        string
	AccessToken  string
	RefreshToken string
}

func RegisterUser(req models.RegisterRequest) error {
	// even if the user already exists, we do the heavy bcrypt hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), config.Env.BcryptCost)
	if err != nil {
		return ErrInternal
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
		_, _ = token.GenerateSecureToken(config.Env.SecureTokenBytes)
		return nil // Success from the client's perspective
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
		return ErrEmailExists
	}

	// generate ActionToken for email verification
	verificationToken, err := token.GenerateSecureToken(config.Env.SecureTokenBytes)
	if err != nil {
		return ErrInternal
	}

	actionToken := models.ActionToken{
		UserID:    user.ID,
		Token:     verificationToken,
		Type:      "verify_email",
		ExpiresAt: time.Now().Add(config.Env.VerifyEmailLifetime), // time available to verify
	}

	// save ActionToken for email verification to database
	if err := database.DB.Create(&actionToken).Error; err != nil {
		return ErrInternal
	}

	// send email using embedded templates
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", config.Env.APIBaseURL, verificationToken)
	data := struct{ VerificationURL string }{VerificationURL: verificationURL}

	subject, body, tmplErr := mailer.BuildEmailContent("verify_email", data)
	if tmplErr == nil {
		// send email in goroutine so SMTP server network latency does not affect API response time
		go func() {
			_ = mailer.Client.SendEmail(user.Email, subject, body)
		}()
	}

	return nil
}

func VerifyEmail(tokenString string) error {
	var actionToken models.ActionToken

	// find token and preload user
	if err := database.DB.Where("token = ? AND type = ?", tokenString, "verify_email").First(&actionToken).Error; err != nil {
		return ErrTokenInvalid
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		database.DB.Delete(&actionToken)
		return ErrTokenInvalid
	}

	// transaction - update user and delete token
	tx := database.DB.Begin()

	// update user to verified
	if err := tx.Model(&models.User{}).Where("id = ?", actionToken.UserID).Update("is_email_verified", true).Error; err != nil {
		tx.Rollback()
		return ErrInternal
	}

	// delete used token
	if err := tx.Delete(&actionToken).Error; err != nil {
		tx.Rollback()
		return ErrInternal
	}

	tx.Commit()
	return nil
}

func AuthenticateUser(email, password, clientIP, userAgent string) (*LoginResult, error) {
	// look up user by email
	var user models.User
	userExists := true
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		userExists = false
		// do not return here, continue to dummy bcrypt comparison to avoid timing attacks
	}

	// compare provided password against stored bcrypt hash, also dummy verifications for nonexistent user
	if userExists {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			// vague, do not reveal whether email exists
			return nil, ErrInvalidCredentials
		}

		// check verification only is password is correct,
		// but return same vague error message to protect against enuemration
		if !user.IsEmailVerified {
			return nil, ErrInvalidCredentials
		}
	} else {
		// dummy comparison
		// declared to not compute a random cost 14 hash
		const dummyBcryptHash = "$2a$14$1AB05scB8KFNDuDWpgvzkO6GYYf62uSGJr445WX6x2jHkWpcySpjW"
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		return nil, ErrInvalidCredentials
	}

	// if the user has 2FA enabled, stop and tell client to prompt for code
	if user.IsTwoFactorEnable {
		return &LoginResult{Requires2FA: true, Email: user.Email}, nil
	}

	// if 2FA is not enabled, log in normally
	accessToken, refreshToken, err := token.GenerateTokensAndSession(
		user.ID,
		clientIP,
		userAgent,
		[]byte(config.Env.JWTSecret),
	)
	if err != nil {
		return nil, ErrInternal
	}

	return &LoginResult{
		Requires2FA:  false,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func Verify2FA(email, password, totpToken, clientIP, userAgent string) (string, string, error) {
	// re-authenticate user (stateless flow)
	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	if !user.IsTwoFactorEnable {
		return "", "", Err2FANotEnabled
	}

	plainSecret, err := crypto.DecryptAES(user.TwoFactorSecret, []byte(config.Env.AESMasterKey))
	if err != nil {
		return "", "", ErrInternal
	}

	// validate TOTP code
	valid := totp.Validate(totpToken, plainSecret)
	if !valid {
		return "", "", ErrInvalid2FAToken
	}

	// generate session
	accessToken, refreshToken, err := token.GenerateTokensAndSession(
		user.ID,
		clientIP,
		userAgent,
		[]byte(config.Env.JWTSecret),
	)
	if err != nil {
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}

func RefreshToken(refreshTokenStr, clientIP, userAgent string) (string, string, error) {
	// look up session in the DB, preload user to make sure he hasn't been deleted
	var session models.Session
	if err := database.DB.Preload("User").Where("refresh_token = ?", refreshTokenStr).First(&session).Error; err != nil {
		return "", "", ErrTokenInvalid
	}
	// remember the last UpdatedAt when we retrieve the session, to prevent race conditions (optimistic locking)
	originalUpdatedAt := session.UpdatedAt

	// token reuse detection
	// if someone tries to use a token that has already been changed by the legitimate user, we invalidate all sessions
	if session.IsUsed {
		database.DB.Where("family_id = ?", session.FamilyID).Delete(&models.Session{})
		return "", "", ErrTokenReuseDetected
	}

	// check expiration
	if time.Now().After(session.ExpiresAt) {
		// cleanup expired session
		database.DB.Delete(&session)
		return "", "", ErrTokenExpired
	}

	// refresh token rotation with optimistic concurrency control
	// mark current token as being used, and keep it in the db as a trap for potential attackers
	// only if it hasn't been modified by another conucrrent request
	result := database.DB.Model(&models.Session{}).
		Where("id = ? AND updated_at = ?", session.ID, originalUpdatedAt).
		Update("is_used", true)
	// if 0 rows were affected, means another request just updated this token
	if result.RowsAffected == 0 {
		return "", "", ErrConcurrentRequest
	}

	// generate new refresh token
	newRefreshToken, err := token.GenerateSecureToken(config.Env.SecureTokenBytes)
	if err != nil {
		return "", "", ErrInternal
	}

	// send new access token with configured lifetime
	claims := models.Claims{
		UserID: session.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.Env.AccessTokenLifetime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newAccessToken, err := newToken.SignedString([]byte(config.Env.JWTSecret))
	if err != nil {
		return "", "", ErrInternal
	}

	// save new token, with the same familyid as the previous one
	newSession := models.Session{
		UserID:       session.UserID,
		FamilyID:     session.FamilyID, // same faimlyid as the previous session
		RefreshToken: newRefreshToken,
		IsUsed:       false,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(config.Env.RefreshTokenLifetime),
	}
	database.DB.Create(&newSession)

	return newAccessToken, newRefreshToken, nil
}

func LogoutUser(refreshToken string) error {
	// delete session from the db (access token is nto deleted because it has short lifetime)
	// return succes even if we have an error here
	// client will clear local state anyway
	database.DB.Where("refresh_token = ?", refreshToken).Delete(&models.Session{})
	return nil
}

func ForgotPassword(email string) error {
	// record actual logic computing time to standardize response times to avoid timing attacks
	startTime := time.Now()

	// look up user
	var user models.User
	userExists := true
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		userExists = false
	}

	// generate recovery token, even if user is not found, to combat timing attacks
	recoveryToken, err := token.GenerateSecureToken(config.Env.SecureTokenBytes)
	if err != nil {
		return ErrInternal
	}

	if userExists && user.IsEmailVerified {
		// generate and save action token
		actionToken := models.ActionToken{
			UserID:    user.ID,
			Token:     recoveryToken,
			Type:      "reset_password",
			ExpiresAt: time.Now().Add(config.Env.ResetPasswordLifetime),
		}

		if err := database.DB.Create(&actionToken).Error; err == nil {
			// send recovery email using embedded templates
			recoveryURL := fmt.Sprintf("%s/reset-password?token=%s", config.Env.FrontendBaseURL, recoveryToken)
			data := struct{ RecoveryURL string }{RecoveryURL: recoveryURL}

			subject, body, tmplErr := mailer.BuildEmailContent("reset_password", data)
			if tmplErr == nil {
				// send email in goroutine so SMTP server network latency does not affect API response time
				go func() {
					_ = mailer.Client.SendEmail(user.Email, subject, body)
				}()
			}
		}
	}

	// timing attack avoidance logic
	// stop timer to see how long it took to compute real logic
	elapsed := time.Since(startTime)
	// use configured target time for request leveling
	targetTime := config.Env.TimingAttackTarget
	// generate random noise
	noise := time.Duration(rand.Intn(config.Env.TimingAttackNoise)) * time.Millisecond

	// level actual response time with the target time
	if elapsed < targetTime {
		// the request was too fast, so we sleep until the target time + noise to prevent patterns
		time.Sleep((targetTime - elapsed) + noise)
	} else {
		// if we surpassed target time, still sleep a bit to prevent patterns
		time.Sleep(noise)
	}

	return nil
}

func ResetPassword(tokenStr, newPassword string) error {
	var actionToken models.ActionToken

	// find token and check type
	if err := database.DB.Where("token = ? AND type = ?", tokenStr, "reset_password").First(&actionToken).Error; err != nil {
		return ErrTokenInvalid
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		database.DB.Delete(&actionToken)
		return ErrTokenExpired
	}

	// hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), config.Env.BcryptCost)
	if err != nil {
		return ErrInternal
	}

	// transaction - update user pass and delete token
	// transaction is used in case of token deletion failure, then the password will not be updated
	tx := database.DB.Begin()
	if err := tx.Model(&models.User{}).Where("id = ?", actionToken.UserID).Update("password", string(hashedPassword)).Error; err != nil {
		tx.Rollback()
		return ErrInternal
	}

	// delete used token, it should be used only once
	if err := tx.Delete(&actionToken).Error; err != nil {
		tx.Rollback()
		return ErrInternal
	}

	// invalidate all existing sessions for this user
	if err := tx.Where("user_id = ?", actionToken.UserID).Delete(&models.Session{}).Error; err != nil {
		tx.Rollback()
		return ErrInternal
	}

	tx.Commit()
	return nil
}
