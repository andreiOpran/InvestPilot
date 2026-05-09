package services

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
	"github.com/andreiOpran/licenta/operational-node/utils/token"
	"github.com/andreiOpran/licenta/operational-node/utils/validator"
)

type AuthService interface {
	RegisterUser(req models.RegisterRequest) error
	VerifyEmail(tokenString string) error
	AuthenticateUser(email, password, clientIP, userAgent string) (*LoginResult, error)
	Verify2FA(email, password, totpToken, clientIP, userAgent string) (string, string, error)
	RefreshToken(refreshTokenStr, clientIP, userAgent string) (string, string, error)
	LogoutUser(refreshToken string) error
	ForgotPassword(email string) error
	ResetPassword(tokenStr, newPassword string) error
}

type authService struct {
	authRepo repositories.AuthRepository
}

func NewAuthService(authRepo repositories.AuthRepository) AuthService {
	return &authService{
		authRepo: authRepo,
	}
}

type LoginResult struct {
	Requires2FA  bool
	Email        string
	AccessToken  string
	RefreshToken string
}

func (s *authService) RegisterUser(req models.RegisterRequest) error {
	// password policy check
	userInputs := []string{req.Email}
	if err := validator.ValidatePassword(req.Password, userInputs); err != nil {
		return err
	}

	// even if the user already exists, we do the heavy bcrypt hashing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), config.Env.BcryptCost)
	if err != nil {
		return ErrInternal
	}

	// check if the user exists
	_, err = s.authRepo.FindUserByEmail(req.Email)
	userExists := err == nil // if no error, user exists

	// if user exists, pretend registration was successful to avoid user enumeration
	if userExists {
		// generate dummy token to simulate time taken by rand ops
		_, _ = token.GenerateSecureToken(config.Env.SecureTokenBytes)
		return nil // success from the client's perspective
	}

	// if user does not exist, proceed with creation
	// build user with an empty wallet and isemailverified=false
	user := models.User{
		Email:             req.Email,
		Password:          string(hashedPassword),
		RiskTolerance:     0, // will be updated later by onboarding form
		InvestmentHorizon: 0, // will be updated later by onboarding form
	}

	// save to db via repo
	// wallet is handled by repo
	if err := s.authRepo.CreateUser(&user); err != nil {
		// if insert fails because on unique constraing on email,
		// we pretend it worked to block enumeration
		return nil
	}

	// generate actiontoken for email verification
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

	// save actiontoken for email verification to database
	if err := s.authRepo.CreateActionToken(&actionToken); err != nil {
		return ErrInternal
	}

	// send email using embedded templates
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", config.Env.FrontendBaseURL, verificationToken)
	data := struct{ VerificationURL string }{VerificationURL: verificationURL}

	subject, body, tmplErr := mailer.BuildEmailContent("verify_email", data)
	if tmplErr == nil {
		// send email in goroutine so smtp server network latency does not affect api response time
		go func() {
			_ = mailer.Client.SendEmail(user.Email, subject, body)
		}()
	}

	return nil
}

func (s *authService) VerifyEmail(tokenString string) error {
	// find token
	actionToken, err := s.authRepo.FindActionToken(tokenString, "verify_email")
	if err != nil {
		return ErrTokenInvalid
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		s.authRepo.DeleteActionToken(actionToken)
		return ErrTokenInvalid
	}

	// transaction handled internally by repository
	if err := s.authRepo.VerifyEmailTx(actionToken.UserID, actionToken.ID); err != nil {
		return ErrInternal
	}

	return nil
}

func (s *authService) AuthenticateUser(email, password, clientIP, userAgent string) (*LoginResult, error) {
	// look up user by email
	user, err := s.authRepo.FindUserByEmail(email)
	userExists := err == nil

	// evaluate progressive lockout
	var consecutiveFails int
	var lastAttemptTime time.Time

	if userExists {
		consecutiveFails, lastAttemptTime, _ = s.authRepo.GetConsecutiveFailedAttempts(user.ID)

		if consecutiveFails >= config.Env.LockoutThreshold1 {
			var lockoutDuration time.Duration
			if consecutiveFails >= config.Env.LockoutThreshold3 {
				lockoutDuration = config.Env.LockoutDuration3
			} else if consecutiveFails >= config.Env.LockoutThreshold2 {
				lockoutDuration = config.Env.LockoutDuration2
			} else {
				lockoutDuration = config.Env.LockoutDuration1
			}

			// check if we are still within the penalty box time
			if time.Since(lastAttemptTime) < lockoutDuration {
				return nil, ErrAccountLocked
			}
		}
	}

	// validate password (fall back to dummy hash for nonexistent user to prevent timing attacks)
	var passwordOk bool
	if userExists {
		passwordOk = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil
	} else {
		// dummy comparison
		// dummyBcryptHash is declared to not compute a random cost 14 hash
		const dummyBcryptHash = "$2a$14$1AB05scB8KFNDuDWpgvzkO6GYYf62uSGJr445WX6x2jHkWpcySpjW"
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
	}

	// create login attempt records for real users
	if userExists {
		attempt := models.LoginAttempt{
			UserID:    user.ID,
			IsSuccess: passwordOk,
			IPAddress: clientIP,
		}
		_ = s.authRepo.CreateLoginAttempt(&attempt)

		if !passwordOk {
			newFails := consecutiveFails + 1

			// send warning email on exactly the first threshold breach
			if newFails == config.Env.LockoutThreshold1 {
				subject, body, tmplErr := mailer.BuildEmailContent("lockout_alert", nil)
				if tmplErr == nil {
					// send email in goroutine so smtp server network latency does not affect api response time
					go func() {
						_ = mailer.Client.SendEmail(user.Email, subject, body)
					}()
				}
			}
			// vague, do not reveal whether email exists
			return nil, ErrInvalidCredentials
		}

		// check verification only if password is correct,
		// but return same vague error message to protect against enuemration
		if !user.IsEmailVerified {
			return nil, ErrInvalidCredentials
		}
	} else {
		// vague, do not reveal whether email exists
		return nil, ErrInvalidCredentials
	}

	// if the user has 2fa enabled, stop and tell client to prompt for code
	if user.IsTwoFactorEnable {
		return &LoginResult{Requires2FA: true, Email: user.Email}, nil
	}

	// if 2fa is not enabled, log in normally
	// get tokens
	accessToken, refreshToken, familyID, err := token.GenerateTokens(user.ID, []byte(config.Env.JWTSecret))
	if err != nil {
		return nil, ErrInternal
	}

	// generate and save session
	session := models.Session{
		UserID:       user.ID,
		FamilyID:     familyID,
		RefreshToken: refreshToken,
		IsUsed:       false,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(config.Env.RefreshTokenLifetimeHours),
	}
	if err := s.authRepo.CreateSession(&session); err != nil {
		return nil, ErrInternal
	}

	return &LoginResult{
		Requires2FA:  false,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *authService) Verify2FA(email, password, totpToken, clientIP, userAgent string) (string, string, error) {
	// re-authenticate user (stateless flow)
	user, err := s.authRepo.FindUserByEmail(email)
	if err != nil {
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

	// validate totp code
	valid := totp.Validate(totpToken, plainSecret)
	if !valid {
		return "", "", ErrInvalid2FAToken
	}

	// get tokens
	accessToken, refreshToken, familyID, err := token.GenerateTokens(user.ID, []byte(config.Env.JWTSecret))
	if err != nil {
		return "", "", ErrInternal
	}

	// generate and save session
	session := models.Session{
		UserID:       user.ID,
		FamilyID:     familyID,
		RefreshToken: refreshToken,
		IsUsed:       false,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(config.Env.RefreshTokenLifetimeHours),
	}
	if err := s.authRepo.CreateSession(&session); err != nil {
		return "", "", ErrInternal
	}

	return accessToken, refreshToken, nil
}

func (s *authService) RefreshToken(refreshTokenStr, clientIP, userAgent string) (string, string, error) {
	// look up session in the db
	session, err := s.authRepo.FindSessionByToken(refreshTokenStr)
	if err != nil {
		return "", "", ErrTokenInvalid
	}
	// remember the last updatedat when we retrieve the session, to prevent race conditions (optimistic locking)
	originalUpdatedAt := session.UpdatedAt

	// token reuse detection
	// if someone tries to use a token that has already been changed by the legitimate user, we invalidate all sessions
	if session.IsUsed {
		s.authRepo.DeleteSessionsByFamily(session.FamilyID)
		return "", "", ErrTokenReuseDetected
	}

	// check expiration
	if time.Now().After(session.ExpiresAt) {
		// cleanup expired session
		s.authRepo.DeleteSession(session)
		return "", "", ErrTokenExpired
	}

	// refresh token rotation with optimistic concurrency control
	rowsAffected, err := s.authRepo.MarkSessionAsUsed(session.ID, originalUpdatedAt)
	// if 0 rows were affected, means another request just updated this token
	if err != nil || rowsAffected == 0 {
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
		ExpiresAt:    time.Now().Add(config.Env.RefreshTokenLifetimeHours),
	}

	if err := s.authRepo.CreateSession(&newSession); err != nil {
		return "", "", ErrInternal
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *authService) LogoutUser(refreshToken string) error {
	// delete session from the db (access token is nto deleted because it has short lifetime)
	// return succes even if we have an error here
	// client will clear local state anyway
	s.authRepo.DeleteSessionByToken(refreshToken)
	return nil
}

func (s *authService) ForgotPassword(email string) error {
	// record actual logic computing time to standardize response times to avoid timing attacks
	startTime := time.Now()

	// look up user
	user, err := s.authRepo.FindUserByEmail(email)
	userExists := err == nil

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

		if err := s.authRepo.CreateActionToken(&actionToken); err == nil {
			// send recovery email using embedded templates
			recoveryURL := fmt.Sprintf("%s/reset-password?token=%s", config.Env.FrontendBaseURL, recoveryToken)
			data := struct{ RecoveryURL string }{RecoveryURL: recoveryURL}

			subject, body, tmplErr := mailer.BuildEmailContent("reset_password", data)
			if tmplErr == nil {
				// send email in goroutine so smtp server network latency does not affect api response time
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

func (s *authService) ResetPassword(tokenStr, newPassword string) error {
	// find token and check type
	actionToken, err := s.authRepo.FindActionToken(tokenStr, "reset_password")
	if err != nil {
		return ErrTokenInvalid
	}

	// check expiration
	if time.Now().After(actionToken.ExpiresAt) {
		// cleanup expired token
		s.authRepo.DeleteActionToken(actionToken)
		return ErrTokenExpired
	}

	// extract user data from DB based on userID from token, to use it in password validation
	user, err := s.authRepo.FindUserByID(actionToken.UserID)
	if err != nil {
		return ErrInternal
	}

	// password policy check
	userInputs := []string{user.Email}
	if err := validator.ValidatePassword(newPassword, userInputs); err != nil {
		return err
	}

	// hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), config.Env.BcryptCost)
	if err != nil {
		return ErrInternal
	}

	// pass to repository to handle the transaction securely
	if err := s.authRepo.ResetPasswordTx(actionToken.UserID, actionToken.ID, string(hashedPassword)); err != nil {
		return ErrInternal
	}

	return nil
}
