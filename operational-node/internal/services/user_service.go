package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type UserService interface {
	GetUserProfile(userID uint) (*models.User, error)
	UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error
	DepositFunds(userID uint, amount float64) (float64, error)
}

type userService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) UserService {
	return &userService{
		db: db,
	}
}

func (s *userService) GetUserProfile(userID uint) (*models.User, error) {
	var user models.User
	// Preload("Wallet") tells GORM to also fetch the attached wallet data
	if err := s.db.Preload("Wallet").First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (s *userService) UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error {
	var user models.User

	if err := s.db.First(&user, userID).Error; err != nil {
		return ErrUserNotFound
	}

	// update the financial profile
	user.RiskTolerance = req.RiskTolerance
	user.InvestmentHorizon = req.InvestmentHorizon

	// save changes to db
	if err := s.db.Save(&user).Error; err != nil {
		return ErrInternal
	}

	return nil
}

func (s *userService) DepositFunds(userID uint, amount float64) (float64, error) {
	var wallet models.Wallet

	// atomic update to prevent race conditions (lost update anomaly)
	err := s.db.Model(&wallet).
		Where("user_id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error

	if err != nil {
		return 0, ErrInternal
	}

	// fetch updated wallet to return new balance
	if err := s.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return 0, ErrUserNotFound
	}

	return wallet.Balance, nil
}
