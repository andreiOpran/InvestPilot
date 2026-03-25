package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func GetUserProfile(userID uint) (*models.User, error) {
	var user models.User
	// Preload("Wallet") tells GORM to also fetch the attached Wallet data
	if err := database.DB.Preload("Wallet").First(&user, userID).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func DepositFunds(userID uint, amount float64) (float64, error) {
	var user models.User
	// find the authenticated user and their attached wallet
	if err := database.DB.Preload("Wallet").First(&user, userID).Error; err != nil {
		return 0, ErrUserNotFound
	}

	// add simulated money to the wallet
	user.Wallet.Balance += amount
	user.Wallet.UserId = user.ID

	// save updated walet to the database
	if err := database.DB.Save(&user.Wallet).Error; err != nil {
		return 0, ErrInternal
	}

	return user.Wallet.Balance, nil
}
