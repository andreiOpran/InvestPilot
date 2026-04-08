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
	DepositTx(userID uint, amount float64, fundingLog *models.Funding) error
	CashoutTx(userID uint, amount float64, fundingLog *models.Funding) error
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

func (r *userRepository) DepositTx(userID uint, amount float64, fundingLog *models.Funding) error {
	// begin gorm transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		// atomically update wallet balance
		err := tx.Model(&models.Wallet{}).
			Where("user_id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", amount)).Error
		if err != nil {
			return err
		}

		return tx.Create(fundingLog).Error
	})
}

func (r *userRepository) CashoutTx(userID uint, amount float64, fundingLog *models.Funding) error {
	// begin gorm transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		// atomically update wallet balance only if there are enough funds
		result := tx.Model(&models.Wallet{}).
			Where("user_id = ? AND balance >= ?", userID, amount).
			Update("balance", gorm.Expr("balance - ?", amount))
		if result.Error != nil {
			return result.Error
		}

		// if no rows affected, means balance >= amount did not pass
		if result.RowsAffected == 0 {
			return ErrUserCashoutInsufficientFunds
		}

		return tx.Create(fundingLog).Error
	})
}
