package repomocks

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockPortfolioRepository struct {
	mock.Mock
}

func (m *MockPortfolioRepository) GetRoundWithHoldingsByStatus(userID uint, isActive bool) (*models.InvestmentRound, error) {
	args := m.Called(userID, isActive)
	if args.Get(0) != nil {
		return args.Get(0).(*models.InvestmentRound), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetHistoricalRounds(userID uint, since time.Time) ([]models.InvestmentRound, error) {
	args := m.Called(userID, since)
	if args.Get(0) != nil {
		return args.Get(0).([]models.InvestmentRound), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetHistoricalFundings(userID uint) ([]models.Funding, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Funding), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetInvestTransactions(userID uint) ([]models.Transaction, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).([]models.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetLatestPrices(tickers []string) (map[string]float64, error) {
	args := m.Called(tickers)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]float64), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetPricingData(tickers []string, since time.Time, isIntraday bool) (map[string][]models.AssetPricePoint, error) {
	args := m.Called(tickers, since, isIntraday)
	if args.Get(0) != nil {
		return args.Get(0).(map[string][]models.AssetPricePoint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetPricesBeforeWindow(tickers []string, since time.Time, isIntraday bool) (map[string]float64, error) {
	args := m.Called(tickers, since, isIntraday)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]float64), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) GetMarketTimestamps(since time.Time, isIntraday bool) ([]time.Time, error) {
	args := m.Called(since, isIntraday)
	if args.Get(0) != nil {
		return args.Get(0).([]time.Time), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioRepository) ExecuteInvestTransaction(
	wallet *models.Wallet,
	txRecord *models.Transaction,
	oldRound *models.InvestmentRound,
	newRound *models.InvestmentRound,
) error {
	args := m.Called(wallet, txRecord, oldRound, newRound)
	return args.Error(0)
}

func (m *MockPortfolioRepository) ExecuteSellTransaction(
	wallet *models.Wallet,
	txRecord *models.Transaction,
	oldRound *models.InvestmentRound,
	newRound *models.InvestmentRound,
) error {
	args := m.Called(wallet, txRecord, oldRound, newRound)
	return args.Error(0)
}
