package services

import (
	"errors"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/google/uuid"
)

type UserService interface {
	GetUserProfile(userID uint) (*models.User, error)
	UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error
	DepositFunds(userID uint, amount float64) (float64, error)
	Cashout(userID uint, amount float64) (float64, error)
	ProcessWebhookDeposit(userID uint, amount int64, stripeID string) error
}

type userService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) GetUserProfile(userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByIDWithWallet(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *userService) UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// update the financial profile
	user.RiskTolerance = req.RiskTolerance
	user.InvestmentHorizon = req.InvestmentHorizon

	// save changes to db
	if err := s.userRepo.Save(user); err != nil {
		return ErrInternal
	}

	return nil
}

// This is now unused, as deposit is made by stripe webhooks, which calls the DepositTx()
func (s *userService) DepositFunds(userID uint, amount float64) (float64, error) {
	stripeID := "sim_paper_trading_deposit"
	funding := &models.Funding{
		UserID:          userID,
		Type:            "DEPOSIT",
		Amount:          amount,
		StripePaymentID: stripeID,
		Status:          "COMPLETED",
	}

	// atomic update to prevent race conditions and log the funding via transaction
	if err := s.userRepo.DepositTx(userID, amount, funding); err != nil {
		return 0, ErrInternal
	}

	// fetch updated wallet to return new balance
	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return 0, ErrUserNotFound
	}

	return wallet.Balance, nil
}

func (s *userService) ProcessWebhookDeposit(userID uint, paymentIntentAmount int64, stripeID string) error {
	amount := float64(paymentIntentAmount) / 100.0 // convert cents back to flat dollar float

	// generate funding ledger log
	funding := &models.Funding{
		UserID:          userID,
		Type:            "DEPOSIT",
		Amount:          amount,
		StripePaymentID: stripeID,
		Status:          "COMPLETED",
	}
	return s.userRepo.DepositTx(userID, amount, funding)
}

func (s *userService) Cashout(userID uint, amount float64) (float64, error) {
	if amount <= 0 {
		return 0, ErrAmountNegative
	}

	// generate cashout log with mock Stripe ID
	mockStripeID := "sim_out_" + uuid.New().String()
	funding := &models.Funding{
		UserID:          userID,
		Type:            "WITHDRAWAL",
		Amount:          amount,
		StripePaymentID: mockStripeID,
		Status:          "COMPLETED",
	}

	err := s.userRepo.CashoutTx(userID, amount, funding)
	if err != nil {
		if errors.Is(err, repositories.ErrUserCashoutInsufficientFunds) {
			return 0, ErrInsufficientBalance
		}
		return 0, err
	}

	// fetch updated wallet balance
	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return 0, err
	}
	return wallet.Balance, nil
}
