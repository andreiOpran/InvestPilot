package services

import (
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestSetup2FA_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	mockRepo.On("FindByID", uint(999)).Return((*models.User)(nil), errors.New("not found")).Once()
	_, _, _, err := service.Setup2FA(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestSetup2FA_alreadyEnabled_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	enabledUser := &models.User{Email: "1@test.com", IsTwoFactorEnable: true}
	mockRepo.On("FindByID", uint(2)).Return(enabledUser, nil).Once()
	_, _, _, err := service.Setup2FA(2)
	assert.ErrorIs(t, err, Err2FAAlreadyEnabled)
	mockRepo.AssertExpectations(t)
}

func TestSetup2FA_validRequest_returnsSecretAndQR(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	normalUser := &models.User{Email: "2@test.com", IsTwoFactorEnable: false}
	mockRepo.On("FindByID", uint(1)).Return(normalUser, nil).Once()
	mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(nil).Once()

	secret, uri, qr, err := service.Setup2FA(1)
	assert.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Contains(t, uri, "otpauth://")
	assert.NotEmpty(t, qr)
	mockRepo.AssertExpectations(t)
}

func TestEnable2FA_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	mockRepo.On("FindByID", uint(999)).Return((*models.User)(nil), errors.New("not found")).Once()
	err := service.Enable2FA(999, "123456")
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestEnable2FA_invalidToken_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	user := &models.User{Email: "2@test.com", IsTwoFactorEnable: false}

	// mock calls for setup phase
	mockRepo.On("FindByID", uint(1)).Return(user, nil).Twice()
	mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(nil).Times(1)

	// generate secret first via setup
	_, _, _, _ = service.Setup2FA(1)

	// test with wrong code
	err := service.Enable2FA(1, "000000")
	assert.ErrorIs(t, err, ErrInvalid2FAToken)
	mockRepo.AssertExpectations(t)
}

func TestEnable2FA_validToken_enablesSuccessfully(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewSecurityService(mockRepo)

	user := &models.User{Email: "3@test.com", IsTwoFactorEnable: false}

	mockRepo.On("FindByID", uint(1)).Return(user, nil).Twice()
	mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(nil).Times(2)

	secret, _, _, _ := service.Setup2FA(1)

	validToken, _ := totp.GenerateCode(secret, time.Now())
	err := service.Enable2FA(1, validToken)
	assert.NoError(t, err)
	assert.True(t, user.IsTwoFactorEnable)
	mockRepo.AssertExpectations(t)
}
