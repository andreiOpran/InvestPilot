package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestGetUserProfile_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("FindByIDWithWallet", uint(999)).Return((*models.User)(nil), errors.New("db error")).Once()

	_, err := service.GetUserProfile(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestGetUserProfile_validId_returnsUser(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewUserService(mockRepo)

	expectedUser := &models.User{Email: "test@example.com", Wallet: models.Wallet{Balance: 150.5}}
	mockRepo.On("FindByIDWithWallet", uint(1)).Return(expectedUser, nil).Once()

	foundUser, err := service.GetUserProfile(1)
	assert.NoError(t, err)
	assert.Equal(t, 150.5, foundUser.Wallet.Balance)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserProfile_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewUserService(mockRepo)
	req := models.UpdateProfileRequest{RiskTolerance: 4, InvestmentHorizon: 20}

	mockRepo.On("FindByID", uint(999)).Return((*models.User)(nil), errors.New("db error")).Once()

	err := service.UpdateUserProfile(999, req)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserProfile_validRequest_updatesSuccessfully(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewUserService(mockRepo)
	req := models.UpdateProfileRequest{RiskTolerance: 4, InvestmentHorizon: 20}

	user := &models.User{Email: "update@example.com"}
	mockRepo.On("FindByID", uint(1)).Return(user, nil).Once()
	mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(nil).Once()

	err := service.UpdateUserProfile(1, req)
	assert.NoError(t, err)
	assert.Equal(t, 4, user.RiskTolerance)
	assert.Equal(t, 20, user.InvestmentHorizon)
	mockRepo.AssertExpectations(t)
}

func TestDepositFunds_validRequest_returnsNewBalance(t *testing.T) {
	mockRepo := new(mocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("AddWalletBalance", uint(1), 50.5).Return(nil).Once()
	mockRepo.On("FindWalletByUserID", uint(1)).Return(&models.Wallet{Balance: 150.5}, nil).Once()

	newBalance, err := service.DepositFunds(1, 50.5)
	assert.NoError(t, err)
	assert.Equal(t, 150.5, newBalance)
	mockRepo.AssertExpectations(t)
}
