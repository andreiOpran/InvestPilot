package services

import (
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/utils/crypto"
)

func TestRegisterUser_existingUser_returnsSuccessQuietly(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)
	req := models.RegisterRequest{Email: "reg@test.com", Password: "ValidPass1!", TurnstileToken: "tok"}

	mockRepo.On("FindUserByEmail", req.Email).Return(&models.User{}, nil).Once()

	err := service.RegisterUser(req)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRegisterUser_newUser_registersSuccessfully(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindUserByEmail", "new@test.com").Return((*models.User)(nil), errors.New("not found")).Once()
	mockRepo.On("CreateUser", mock.AnythingOfType("*models.User")).Return(nil).Once()
	mockRepo.On("CreateActionToken", mock.AnythingOfType("*models.ActionToken")).Return(nil).Once()

	err := service.RegisterUser(models.RegisterRequest{Email: "new@test.com", Password: "ValidPass1!", TurnstileToken: "tok"})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRegisterUser_createActionTokenError_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindUserByEmail", "new2@test.com").Return((*models.User)(nil), errors.New("not found")).Once()
	mockRepo.On("CreateUser", mock.AnythingOfType("*models.User")).Return(nil).Once()
	mockRepo.On("CreateActionToken", mock.AnythingOfType("*models.ActionToken")).Return(ErrInternal).Once()

	err := service.RegisterUser(models.RegisterRequest{Email: "new2@test.com", Password: "ValidPass1!", TurnstileToken: "tok"})
	assert.ErrorIs(t, err, ErrInternal)
	mockRepo.AssertExpectations(t)
}

func TestRegisterUser_weakPassword_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	err := service.RegisterUser(models.RegisterRequest{Email: "new@test.com", Password: "weak", TurnstileToken: "tok"})
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestVerifyEmail_invalidToken_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindActionToken", "invalid", "verify_email").Return((*models.ActionToken)(nil), errors.New("not found")).Once()
	err := service.VerifyEmail("invalid")
	assert.ErrorIs(t, err, ErrTokenInvalid)
	mockRepo.AssertExpectations(t)
}

func TestVerifyEmail_expiredToken_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	expiredToken := &models.ActionToken{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	mockRepo.On("FindActionToken", "expired", "verify_email").Return(expiredToken, nil).Once()
	mockRepo.On("DeleteActionToken", expiredToken).Return(nil).Once()

	err := service.VerifyEmail("expired")
	assert.ErrorIs(t, err, ErrTokenInvalid)
	mockRepo.AssertExpectations(t)
}

func TestVerifyEmail_validToken_verifiesSuccessfully(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validToken := &models.ActionToken{UserID: 1, ID: 1, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindActionToken", "valid", "verify_email").Return(validToken, nil).Once()
	mockRepo.On("VerifyEmailTx", uint(1), uint(1)).Return(nil).Once()

	err := service.VerifyEmail("valid")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_ghostUser_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindUserByEmail", "ghost@test.com").Return((*models.User)(nil), errors.New("not found")).Once()
	_, err := service.AuthenticateUser("ghost@test.com", "pass", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_validCredentials_returnsTokens(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsEmailVerified: true}

	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(0, time.Time{}, nil).Once()
	mockRepo.On("CreateLoginAttempt", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	res, err := service.AuthenticateUser("ok@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.False(t, res.Requires2FA)
	assert.NotEmpty(t, res.AccessToken)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_2faRequired_returnsStatus(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "2fa@test.com", Password: string(hashed), IsEmailVerified: true, IsTwoFactorEnable: true}
	mockRepo.On("FindUserByEmail", "2fa@test.com").Return(user, nil).Once()
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(0, time.Time{}, nil).Once()
	mockRepo.On("CreateLoginAttempt", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

	res, err := service.AuthenticateUser("2fa@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.True(t, res.Requires2FA)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_wrongPassword_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsEmailVerified: true}

	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(0, time.Time{}, nil).Once()
	mockRepo.On("CreateLoginAttempt", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

	_, err := service.AuthenticateUser("ok@test.com", "WrongPass1!", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestVerify2FA_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindUserByEmail", "ghost@test.com").Return((*models.User)(nil), errors.New("not found")).Once()
	_, _, err := service.Verify2FA("ghost@test.com", "pass", "123456", "ip", "agent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestVerify2FA_wrongPassword_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed)}
	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()

	_, _, err := service.Verify2FA("ok@test.com", "WrongPass!", "123456", "ip", "agent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestVerify2FA_2faNotEnabled_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsTwoFactorEnable: false}
	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()

	_, _, err := service.Verify2FA("ok@test.com", "ValidPass1!", "123456", "ip", "agent")
	assert.ErrorIs(t, err, Err2FANotEnabled)
	mockRepo.AssertExpectations(t)
}

func TestVerify2FA_invalidTOTP_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	key, _ := totp.Generate(totp.GenerateOpts{Issuer: "Test", AccountName: "ok@test.com"})
	encSecret, _ := crypto.EncryptAES(key.Secret(), []byte("0123456789abcdef0123456789abcdef"))
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsTwoFactorEnable: true, TwoFactorSecret: encSecret}
	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()

	_, _, err := service.Verify2FA("ok@test.com", "ValidPass1!", "000000", "ip", "agent")
	assert.ErrorIs(t, err, ErrInvalid2FAToken)
	mockRepo.AssertExpectations(t)
}

func TestVerify2FA_validToken_returnsTokens(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsTwoFactorEnable: true}

	key, _ := totp.Generate(totp.GenerateOpts{Issuer: "Test", AccountName: "ok@test.com"})
	encSecret, _ := crypto.EncryptAES(key.Secret(), []byte("0123456789abcdef0123456789abcdef"))
	user.TwoFactorSecret = encSecret

	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	validCode, _ := totp.GenerateCode(key.Secret(), time.Now())
	acc, ref, err := service.Verify2FA("ok@test.com", "ValidPass1!", validCode, "ip", "agent")
	assert.NoError(t, err)
	assert.NotEmpty(t, acc)
	assert.NotEmpty(t, ref)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_invalidToken_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("FindSessionByToken", "invalid").Return((*models.Session)(nil), errors.New("not found")).Once()
	_, _, err := service.RefreshToken("invalid", "ip", "agent")
	assert.ErrorIs(t, err, ErrTokenInvalid)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_tokenReuse_invalidatesFamily(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	reusedSession := &models.Session{FamilyID: "fam1", IsUsed: true}
	mockRepo.On("FindSessionByToken", "reused").Return(reusedSession, nil).Once()
	mockRepo.On("DeleteSessionsByFamily", "fam1").Return(nil).Once()

	_, _, err := service.RefreshToken("reused", "ip", "agent")
	assert.ErrorIs(t, err, ErrTokenReuseDetected)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_validToken_rotatesSuccessfully(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validSession := &models.Session{ID: 1, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindSessionByToken", "valid").Return(validSession, nil).Once()
	mockRepo.On("MarkSessionAsUsed", uint(1), validSession.UpdatedAt).Return(int64(1), nil).Once()
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	newAccess, newRefresh, err := service.RefreshToken("valid", "ip", "agent")
	assert.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	mockRepo.AssertExpectations(t)
}

func TestForgotPassword_validEmail_sendsEmail(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	user := &models.User{Email: "reset@test.com", IsEmailVerified: true}
	mockRepo.On("FindUserByEmail", "reset@test.com").Return(user, nil).Once()
	mockRepo.On("CreateActionToken", mock.AnythingOfType("*models.ActionToken")).Return(nil).Once()

	err := service.ForgotPassword("reset@test.com")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestResetPassword_validToken_updatesPassword(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validToken := &models.ActionToken{UserID: 1, ID: 1, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindActionToken", "valid", "reset_password").Return(validToken, nil).Once()
	mockRepo.On("FindUserByID", uint(1)).Return(&models.User{Email: "user@test.com"}, nil).Once()
	mockRepo.On("ResetPasswordTx", uint(1), uint(1), mock.AnythingOfType("string")).Return(nil).Once()

	err := service.ResetPassword("valid", "ValidNewPass1!")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_accountLocked_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "locked@test.com", Password: string(hashed), IsEmailVerified: true}
	mockRepo.On("FindUserByEmail", "locked@test.com").Return(user, nil).Once()
	// return enough failures with recent lastAttemptTime to trigger lockout
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(4, time.Now(), nil).Once()

	_, err := service.AuthenticateUser("locked@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrAccountLocked)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_emailNotVerified_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "unverified@test.com", Password: string(hashed), IsEmailVerified: false}
	mockRepo.On("FindUserByEmail", "unverified@test.com").Return(user, nil).Once()
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(0, time.Time{}, nil).Once()
	mockRepo.On("CreateLoginAttempt", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()

	_, err := service.AuthenticateUser("unverified@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	mockRepo.AssertExpectations(t)
}

func TestResetPassword_expiredToken_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	expiredToken := &models.ActionToken{UserID: 1, ID: 1, ExpiresAt: time.Now().Add(-1 * time.Hour)}
	mockRepo.On("FindActionToken", "expired", "reset_password").Return(expiredToken, nil).Once()
	mockRepo.On("DeleteActionToken", expiredToken).Return(nil).Once()

	err := service.ResetPassword("expired", "ValidNewPass1!")
	assert.ErrorIs(t, err, ErrTokenExpired)
	mockRepo.AssertExpectations(t)
}

func TestResetPassword_weakPassword_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validToken := &models.ActionToken{UserID: 1, ID: 1, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindActionToken", "valid", "reset_password").Return(validToken, nil).Once()
	mockRepo.On("FindUserByID", uint(1)).Return(&models.User{Email: "user@test.com"}, nil).Once()

	err := service.ResetPassword("valid", "weak")
	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_expiredSession_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	expiredSession := &models.Session{ID: 1, ExpiresAt: time.Now().Add(-1 * time.Hour)}
	mockRepo.On("FindSessionByToken", "expired").Return(expiredSession, nil).Once()
	mockRepo.On("DeleteSession", expiredSession).Return(nil).Once()

	_, _, err := service.RefreshToken("expired", "ip", "agent")
	assert.ErrorIs(t, err, ErrTokenExpired)
	mockRepo.AssertExpectations(t)
}

func TestRefreshToken_concurrentRequest_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	session := &models.Session{ID: 1, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindSessionByToken", "concurrent").Return(session, nil).Once()
	mockRepo.On("MarkSessionAsUsed", uint(1), session.UpdatedAt).Return(int64(0), nil).Once()

	_, _, err := service.RefreshToken("concurrent", "ip", "agent")
	assert.ErrorIs(t, err, ErrConcurrentRequest)
	mockRepo.AssertExpectations(t)
}

func TestLogoutUser_validToken_deletesSession(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	mockRepo.On("DeleteSessionByToken", "token").Return(nil).Once()
	err := service.LogoutUser("token")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_lockoutThreshold2_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "locked2@test.com", Password: string(hashed), IsEmailVerified: true}
	mockRepo.On("FindUserByEmail", "locked2@test.com").Return(user, nil).Once()
	// threshold2=5 fails, duration2=3min, lastAttemptTime=now -> still locked
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(5, time.Now(), nil).Once()

	_, err := service.AuthenticateUser("locked2@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrAccountLocked)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_lockoutThreshold3_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "locked3@test.com", Password: string(hashed), IsEmailVerified: true}
	mockRepo.On("FindUserByEmail", "locked3@test.com").Return(user, nil).Once()
	// threshold3=6 fails, duration3=15min, lastAttemptTime=now -> still locked
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(6, time.Now(), nil).Once()

	_, err := service.AuthenticateUser("locked3@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.ErrorIs(t, err, ErrAccountLocked)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_lockoutExpired_allowsLogin(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("ValidPass1!"), 4)
	user := &models.User{Email: "expired@test.com", Password: string(hashed), IsEmailVerified: true, IsTwoFactorEnable: false}
	mockRepo.On("FindUserByEmail", "expired@test.com").Return(user, nil).Once()
	// 4 failures but lastAttempt was 10 minutes ago (> LockoutDuration1=1min)
	mockRepo.On("GetConsecutiveFailedAttempts", user.ID).Return(4, time.Now().Add(-10*time.Minute), nil).Once()
	mockRepo.On("CreateLoginAttempt", mock.AnythingOfType("*models.LoginAttempt")).Return(nil).Once()
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	result, err := service.AuthenticateUser("expired@test.com", "ValidPass1!", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestResetPassword_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validToken := &models.ActionToken{UserID: 99, ID: 5, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindActionToken", "tok99", "reset_password").Return(validToken, nil).Once()
	mockRepo.On("FindUserByID", uint(99)).Return(nil, ErrInternal).Once()

	err := service.ResetPassword("tok99", "AnyPass1!")
	assert.ErrorIs(t, err, ErrInternal)
	mockRepo.AssertExpectations(t)
}

func TestResetPassword_repoTxError_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	validToken := &models.ActionToken{UserID: 2, ID: 2, ExpiresAt: time.Now().Add(1 * time.Hour)}
	mockRepo.On("FindActionToken", "tok2", "reset_password").Return(validToken, nil).Once()
	mockRepo.On("FindUserByID", uint(2)).Return(&models.User{Email: "u2@test.com"}, nil).Once()
	mockRepo.On("ResetPasswordTx", uint(2), uint(2), mock.AnythingOfType("string")).Return(ErrInternal).Once()

	err := service.ResetPassword("tok2", "ValidNewPass1!")
	assert.ErrorIs(t, err, ErrInternal)
	mockRepo.AssertExpectations(t)
}
