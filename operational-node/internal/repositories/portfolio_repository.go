package repositories

import (
	"errors"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type PortfolioRepository interface {
	GetRoundWithHoldingsByStatus(userID uint, isActive bool) (*models.InvestmentRound, error)
	ExecuteInvestTransaction(
		wallet *models.Wallet,
		txRecord *models.Transaction,
		oldRound *models.InvestmentRound,
		newRound *models.InvestmentRound,
	) error
}

type portfolioRepository struct {
	db *gorm.DB
}

func NewPortfolioRepository(db *gorm.DB) PortfolioRepository {
	return &portfolioRepository{db: db}
}

func (r *portfolioRepository) GetRoundWithHoldingsByStatus(userID uint, isActive bool) (*models.InvestmentRound, error) {
	var round models.InvestmentRound
	err := r.db.Preload("Holdings").Where("user_id = ? AND is_active = ?", userID, isActive).First(&round).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil // gracefully return nil if no existing matching round
	}
	if err != nil {
		return nil, err
	}
	return &round, nil
}

func (r *portfolioRepository) ExecuteInvestTransaction(
	wallet *models.Wallet,
	txRecord *models.Transaction,
	oldRound *models.InvestmentRound,
	newRound *models.InvestmentRound,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// update wallet balance
		if err := tx.Save(wallet).Error; err != nil {
			return err
		}

		// save transaction record
		if err := tx.Create(txRecord).Error; err != nil {
			return err
		}

		// deactivate old round
		if oldRound != nil {
			if err := tx.Save(oldRound).Error; err != nil {
				return err
			}
		}

		// create new round
		if err := tx.Create(newRound).Error; err != nil {
			return err
		}

		return nil
	})
}

// TODO

// 3. File: portfolio_repository.go
// Change the hardcoded function signature in your repository to accept a parametrized status boolean.
