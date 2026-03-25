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
	AddWalletBalance(userID uint, amount float64) error
	FindWalletByUserID(userID uint) (*models.Wallet, error)
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

func (r *userRepository) AddWalletBalance(userID uint, amount float64) error {
	// atomic update to prevent race conditions directly via gorm.Expr
	return r.db.Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *userRepository) FindWalletByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.Where("user_id = ?", userID).First(&wallet).Error
	return &wallet, err
}
