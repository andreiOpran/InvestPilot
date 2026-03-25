package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type UserService interface {
	GetUserProfile(userID uint) (*models.User, error)
	UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error
	DepositFunds(userID uint, amount float64) (float64, error)
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

func (s *userService) DepositFunds(userID uint, amount float64) (float64, error) {
	// atomic update to prevent race conditions delegated to repository
	if err := s.userRepo.AddWalletBalance(userID, amount); err != nil {
		return 0, ErrInternal
	}

	// fetch updated wallet to return new balance
	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return 0, ErrUserNotFound
	}

	return wallet.Balance, nil
}
