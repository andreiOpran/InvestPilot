package repositories

import (
	"gorm.io/gorm"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// UserRepository handles database logic for the user domain
type UserRepository interface {
	FindByID(userID uint) (*models.User, error)
	FindByIDWithWallet(userID uint) (*models.User, error)
	Save(user *models.User) error
	FindWalletByUserID(userID uint) (*models.Wallet, error)
	DepositTx(userID uint, amount float64, stripeID string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(userID uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, userID).Error
	return &user, err
}

func (r *userRepository) FindByIDWithWallet(userID uint) (*models.User, error) {
	var user models.User
	// Preload("Wallet") tells gorm to also fetch the attached wallet data
	err := r.db.Preload("Wallet").First(&user, userID).Error
	return &user, err
}

func (r *userRepository) Save(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) FindWalletByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.Where("user_id = ?", userID).First(&wallet).Error
	return &wallet, err
}

func (r *userRepository) DepositTx(userID uint, amount float64, stripeID string) error {
	// begin gorm transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		// atomically update wallet balance
		err := tx.Model(&models.Wallet{}).
			Where("user_id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", amount)).Error
		if err != nil {
			return err
		}

		// create immutable funding ledger record
		funding := models.Funding{
			UserID:          userID,
			Type:            "DEPOSIT",
			Amount:          amount,
			StripePaymentID: stripeID,
			Status:          "COMPLETED",
		}
		if err := tx.Create(&funding).Error; err != nil {
			return err
		}

		// return nil to trigger COMMIT
		return nil
	})
}
