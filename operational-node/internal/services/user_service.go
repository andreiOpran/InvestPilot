package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

func GetUserProfile(userID uint) (*models.User, error) {
	var user models.User
	// Preload("Wallet") tells GORM to also fetch the attached Wallet data
	if err := database.DB.Preload("Wallet").First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error {
	var user models.User

	if err := database.DB.First(&user, userID).Error; err != nil {
		return ErrUserNotFound
	}

	// update the financial profile
	user.RiskTolerance = req.RiskTolerance
	user.InvestmentHorizon = req.InvestmentHorizon

	// save changes to db
	if err := database.DB.Save(&user).Error; err != nil {
		return ErrInternal
	}

	return nil
}

func DepositFunds(userID uint, amount float64) (float64, error) {
	var wallet models.Wallet

	// atomic update to prevent race conditions
	err := database.DB.Model(&wallet).
		Where("user_id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error

	if err != nil {
		return 0, ErrInternal
	}

	// fetch updated wallet to return new balance
	if err := database.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return 0, ErrUserNotFound
	}

	return wallet.Balance, nil
}
