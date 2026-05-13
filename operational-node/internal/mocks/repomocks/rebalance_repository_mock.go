package repomocks

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockRebalanceRepository struct {
	mock.Mock
}

func (m *MockRebalanceRepository) GetLatestMarketDataDate() (time.Time, error) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *MockRebalanceRepository) GetLatestModelPortfolios() ([]models.ModelPortfolio, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]models.ModelPortfolio), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRebalanceRepository) GetMaxRoundID() (uint, error) {
	args := m.Called()
	return args.Get(0).(uint), args.Error(1)
}

func (m *MockRebalanceRepository) GetInvestmentRoundsBatchByStatus(isActive bool, lastID uint, maxID uint, batchSize int) ([]models.InvestmentRound, error) {
	args := m.Called(isActive, lastID, maxID, batchSize)
	if args.Get(0) != nil {
		return args.Get(0).([]models.InvestmentRound), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRebalanceRepository) GetLatestPrices() ([]models.DailyMarketData, error) {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]models.DailyMarketData), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRebalanceRepository) ExecuteBatchRebalanceTransaction(newRounds []models.InvestmentRound, oldRoundIDs []uint) error {
	args := m.Called(newRounds, oldRoundIDs)
	return args.Error(0)
}
