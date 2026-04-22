package repositories

import (
	"errors"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/gorm"
)

type PortfolioRepository interface {
	GetRoundWithHoldingsByStatus(userID uint, isActive bool) (*models.InvestmentRound, error)
	GetHistoricalRounds(userID uint, since time.Time) ([]models.InvestmentRound, error)
	GetHistoricalFundings(userID uint) ([]models.Funding, error)
	GetLatestPrices(tickers []string) (map[string]float64, error)
	GetPricingData(tickers []string, since time.Time, isIntraday bool) (map[string][]models.AssetPricePoint, error)
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

func (r *portfolioRepository) GetHistoricalRounds(userID uint, since time.Time) ([]models.InvestmentRound, error) {
	var rounds []models.InvestmentRound

	// time condition (get the active one, or the non-active that were created after `since`)
	timeCondition := r.db.
		Where("is_active = ?", true).
		Or("id IN (SELECT id FROM investment_rounds WHERE user_id = ? AND created_at >= ?)", userID, since)

	err := r.db.Preload("Holdings").
		Where("user_id = ?", userID).
		Where(timeCondition).
		Order("created_at asc").
		Find(&rounds).Error

	return rounds, err
}

func (r *portfolioRepository) GetHistoricalFundings(userID uint) ([]models.Funding, error) {
	var fundings []models.Funding

	// fetch all completed fundings
	err := r.db.Where("user_id = ? AND status = ?", userID, "COMPLETED").
		Order("created_at asc").
		Find(&fundings).Error
	return fundings, err
}

func (r *portfolioRepository) GetLatestPrices(tickers []string) (map[string]float64, error) {
	prices := make(map[string]float64)
	if len(tickers) == 0 {
		return prices, nil
	}

	var results []struct {
		Ticker string
		Price  float64
	}

	// get latest intraday price for each requested ticker
	err := r.db.Table("intraday_market_data").
		Select("DISTINCT ON (ticker) ticker, price").
		Where("ticker IN ?", tickers).
		Order("ticker, timestamp DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		prices[res.Ticker] = res.Price
	}

	return prices, nil
}

func (r *portfolioRepository) GetPricingData(
	tickers []string,
	since time.Time,
	isIntraday bool,
) (map[string][]models.AssetPricePoint, error) {
	var results []models.AssetPricePoint
	var err error

	if isIntraday {
		// look in IntradayMarketData table
		err = r.db.Model(&models.IntradayMarketData{}).
			Where("ticker IN ? AND timestamp >= ?", tickers, since).
			Select("ticker, timestamp, price").
			Order("timestamp asc").
			Scan(&results).Error
	} else {
		// look in DailyMarketData table
		err = r.db.Model(&models.DailyMarketData{}).
			Where("ticker IN ? AND date >= ?", tickers, since).
			Select("ticker, date AS timestamp, close_price AS price").
			Order("date asc").
			Scan(&results).Error
	}

	grouped := make(map[string][]models.AssetPricePoint)
	for _, p := range results {
		grouped[p.Ticker] = append(grouped[p.Ticker], p)
	}
	return grouped, err
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
