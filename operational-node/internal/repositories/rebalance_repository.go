package repositories

import (
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type RebalanceRepository interface {
	GetLatestMarketDataDate() (time.Time, error)
	GetLatestModelPortfolios() ([]models.ModelPortfolio, error)
	GetMaxRoundID() (uint, error)
	GetInvestmentRoundsBatchByStatus(isActive bool, lastID uint, maxID uint, batchSize int) ([]models.InvestmentRound, error)
	GetLatestPrices() ([]models.DailyMarketData, error)
	ExecuteBatchRebalanceTransaction(newRounds []models.InvestmentRound, oldRoundIDs []uint) error
}

type rebalanceRepository struct {
	db *gorm.DB
}

func NewRebalanceRepository(db *gorm.DB) RebalanceRepository {
	return &rebalanceRepository{db: db}
}

func (r *rebalanceRepository) GetLatestMarketDataDate() (time.Time, error) {
	var maxDate time.Time
	err := r.db.Model(&models.DailyMarketData{}).Select("MAX(date)").Scan(&maxDate).Error
	return maxDate, err
}

func (r *rebalanceRepository) GetLatestModelPortfolios() ([]models.ModelPortfolio, error) {
	var portfolios []models.ModelPortfolio
	err := r.db.Raw(`
		SELECT DISTINCT ON (bucket_key) * FROM model_portfolios
		ORDER BY bucket_key, computed_at DESC
	`).Scan(&portfolios).Error

	return portfolios, err
}

func (r *rebalanceRepository) GetMaxRoundID() (uint, error) {
	var maxID uint
	err := r.db.Model(&models.InvestmentRound{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

func (r *rebalanceRepository) GetInvestmentRoundsBatchByStatus(isActive bool, lastID uint, maxID uint, batchSize int) ([]models.InvestmentRound, error) {
	var rounds []models.InvestmentRound
	err := r.db.
		Preload("Holdings").
		Preload("User").
		Where("is_active = ? AND id > ? AND id <= ?", isActive, lastID, maxID).
		Order("id ASC").
		Limit(batchSize).
		Find(&rounds).Error

	return rounds, err
}

func (r *rebalanceRepository) GetLatestPrices() ([]models.DailyMarketData, error) {
	var results []models.DailyMarketData
	err := r.db.Raw(`
		SELECT DISTINCT ON (ticker) ticker, close_price
		FROM daily_market_data
		ORDER BY ticker, date DESC
	`).Scan(&results).Error

	return results, err
}

func (r *rebalanceRepository) ExecuteBatchRebalanceTransaction(
	newRounds []models.InvestmentRound,
	oldRoundIDs []uint,
) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// deactivate old rounds locally
		if len(oldRoundIDs) > 0 {
			if err := tx.Model(&models.InvestmentRound{}).Where("id IN ?", oldRoundIDs).Update("is_active", false).Error; err != nil {
				return err
			}
		}

		// insert new rounds
		if len(newRounds) > 0 {
			if err := tx.Create(&newRounds).Error; err != nil {
				return err
			}
		}
		return nil
	})

}
