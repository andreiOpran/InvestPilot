package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

func TestGetUserProfile_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("FindByIDWithWallet", uint(999)).Return((*models.User)(nil), errors.New("db error")).Once()

	_, err := service.GetUserProfile(999)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestGetUserProfile_validId_returnsUser(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	expectedUser := &models.User{Email: "test@example.com", Wallet: models.Wallet{Balance: 150.5}}
	mockRepo.On("FindByIDWithWallet", uint(1)).Return(expectedUser, nil).Once()

	foundUser, err := service.GetUserProfile(1)
	assert.NoError(t, err)
	assert.Equal(t, 150.5, foundUser.Wallet.Balance)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserProfile_userNotFound_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)
	req := models.UpdateProfileRequest{RiskTolerance: 4, InvestmentHorizon: 20}

	mockRepo.On("FindByID", uint(999)).Return((*models.User)(nil), errors.New("db error")).Once()

	err := service.UpdateUserProfile(999, req)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestUpdateUserProfile_validRequest_updatesSuccessfully(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
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
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("DepositTx", uint(1), 50.5, mock.AnythingOfType("*models.Funding")).Return(nil).Once()
	mockRepo.On("FindWalletByUserID", uint(1)).Return(&models.Wallet{Balance: 150.5}, nil).Once()

	newBalance, err := service.DepositFunds(1, 50.5)
	assert.NoError(t, err)
	assert.Equal(t, 150.5, newBalance)
	mockRepo.AssertExpectations(t)
}

func TestDepositFunds_depositTxError_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("DepositTx", uint(1), 100.0, mock.AnythingOfType("*models.Funding")).Return(ErrInternal).Once()

	_, err := service.DepositFunds(1, 100.0)
	assert.ErrorIs(t, err, ErrInternal)
	mockRepo.AssertExpectations(t)
}

func TestDepositFunds_walletFetchError_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("DepositTx", uint(1), 100.0, mock.AnythingOfType("*models.Funding")).Return(nil).Once()
	mockRepo.On("FindWalletByUserID", uint(1)).Return(nil, ErrInternal).Once()

	_, err := service.DepositFunds(1, 100.0)
	assert.ErrorIs(t, err, ErrUserNotFound)
	mockRepo.AssertExpectations(t)
}

func TestCashout_insufficientBalance_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("CashoutTx", uint(1), 500.0, mock.AnythingOfType("*models.Funding")).Return(repositories.ErrUserCashoutInsufficientFunds).Once()

	_, err := service.Cashout(1, 500.0)
	assert.ErrorIs(t, err, ErrInsufficientBalance)
	mockRepo.AssertExpectations(t)
}

func TestCashout_validRequest_returnsNewBalance(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("CashoutTx", uint(1), 50.0, mock.AnythingOfType("*models.Funding")).Return(nil).Once()
	mockRepo.On("FindWalletByUserID", uint(1)).Return(&models.Wallet{Balance: 50.0}, nil).Once()

	newBalance, err := service.Cashout(1, 50.0)
	assert.NoError(t, err)
	assert.Equal(t, 50.0, newBalance)
	mockRepo.AssertExpectations(t)
}

func TestCashout_negativeAmount_returnsError(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	_, err := service.Cashout(1, -10.0)
	assert.ErrorIs(t, err, ErrAmountNegative)
	mockRepo.AssertExpectations(t)
}

func TestProcessWebhookDeposit_validRequest_depositsSuccessfully(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	service := NewUserService(mockRepo)

	mockRepo.On("DepositTx", uint(1), 10.0, mock.AnythingOfType("*models.Funding")).Return(nil).Once()
	mockRepo.On("FindByID", uint(1)).Return(&models.User{Email: "user@test.com"}, nil).Once()

	err := service.ProcessWebhookDeposit(1, 1000, "pi_test_123") // 1000 cents = $10.00
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
