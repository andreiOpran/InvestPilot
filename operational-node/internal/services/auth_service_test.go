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
	req := models.RegisterRequest{Email: "reg@test.com", Password: "password123"}

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

	err := service.RegisterUser(models.RegisterRequest{Email: "new@test.com", Password: "password123"})
	assert.NoError(t, err)
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

	hashed, _ := bcrypt.GenerateFromPassword([]byte("pass123"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsEmailVerified: true}

	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()
	// now we expect createsession because the service saves it via the repo
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	res, err := service.AuthenticateUser("ok@test.com", "pass123", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.False(t, res.Requires2FA)
	assert.NotEmpty(t, res.AccessToken)
	mockRepo.AssertExpectations(t)
}

func TestAuthenticateUser_2faRequired_returnsStatus(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("pass123"), 4)
	user := &models.User{Email: "2fa@test.com", Password: string(hashed), IsEmailVerified: true, IsTwoFactorEnable: true}
	mockRepo.On("FindUserByEmail", "2fa@test.com").Return(user, nil).Once()

	res, err := service.AuthenticateUser("2fa@test.com", "pass123", "127.0.0.1", "agent")
	assert.NoError(t, err)
	assert.True(t, res.Requires2FA)
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

func TestVerify2FA_validToken_returnsTokens(t *testing.T) {
	mockRepo := new(repomocks.MockAuthRepository)
	service := NewAuthService(mockRepo)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("pass123"), 4)
	user := &models.User{Email: "ok@test.com", Password: string(hashed), IsTwoFactorEnable: true}

	// generate real otp secret for the test
	key, _ := totp.Generate(totp.GenerateOpts{Issuer: "Test", AccountName: "ok@test.com"})
	encSecret, _ := crypto.EncryptAES(key.Secret(), []byte("0123456789abcdef0123456789abcdef"))
	user.TwoFactorSecret = encSecret

	mockRepo.On("FindUserByEmail", "ok@test.com").Return(user, nil).Once()
	mockRepo.On("CreateSession", mock.AnythingOfType("*models.Session")).Return(nil).Once()

	// get valid code to trigger success
	validCode, _ := totp.GenerateCode(key.Secret(), time.Now())
	acc, ref, err := service.Verify2FA("ok@test.com", "pass123", validCode, "ip", "agent")

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
	mockRepo.On("ResetPasswordTx", uint(1), uint(1), mock.AnythingOfType("string")).Return(nil).Once()

	err := service.ResetPassword("valid", "new-pass-123")
	assert.NoError(t, err)
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
