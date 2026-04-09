package repositories

import (
	"encoding/json"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type RebalanceRepository interface {
	GetLatestMarketDataDate() (time.Time, error)
	GetLatestModelPortfolios() (map[string]map[string]float64, error)
	GetMaxRoundID() (uint, error)
	GetActiveInvestmentRoundsBatch(lastID uint, maxID uint, batchSize int) ([]models.InvestmentRound, error)
	GetLatestPrices() (map[string]float64, error)
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
	err := r.db.Model(&models.HistoricalMarketData{}).Select("MAX(date)").Scan(&maxDate).Error
	return maxDate, err
}

func (r *rebalanceRepository) GetLatestModelPortfolios() (map[string]map[string]float64, error) {
	var portfolios []models.ModelPortfolio

	// fetch latest model portfolios
	err := r.db.Raw(`
		SELECT DISTINCT ON (bucket_key) * FROM model_portfolios
		ORDER BY bucket_key, computed_at DESC
	`).Scan(&portfolios).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for _, m := range portfolios {
		var weights map[string]float64
		if err := json.Unmarshal([]byte(m.Weights), &weights); err != nil {
			return nil, err
		}
		result[m.BucketKey] = weights
	}

	return result, nil
}

func (r *rebalanceRepository) GetMaxRoundID() (uint, error) {
	var maxID uint
	err := r.db.Model(&models.InvestmentRound{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error
	return maxID, err
}

func (r *rebalanceRepository) GetActiveInvestmentRoundsBatch(lastID uint, maxID uint, batchSize int) ([]models.InvestmentRound, error) {
	var rounds []models.InvestmentRound
	err := r.db.
		Preload("Holdings").
		Preload("User").
		Where("is_active = ? AND id > ? AND id <= ?", true, lastID, maxID).
		Order("id ASC"). // for cursor pagination
		Limit(batchSize).
		Find(&rounds).Error

	return rounds, err
}

func (r *rebalanceRepository) GetLatestPrices() (map[string]float64, error) {
	type priceResult struct {
		Ticker     string
		ClosePrice float64
	}
	var results []priceResult
	err := r.db.Raw(`
		SELECT DISTINCT ON (ticker) ticker, close_price
		FROM historical_market_data
		ORDER BY ticker, date DESC
	`).Scan(&results).Error

	prices := make(map[string]float64)
	for _, res := range results {
		prices[res.Ticker] = res.ClosePrice
	}
	prices["USD"] = 1.0
	return prices, err
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
