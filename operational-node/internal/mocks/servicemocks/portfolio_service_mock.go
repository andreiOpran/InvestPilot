package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockPortfolioService struct {
	mock.Mock
}

func (m *MockPortfolioService) Invest(userID uint, amount float64) error {
	args := m.Called(userID, amount)
	return args.Error(0)
}

func (m *MockPortfolioService) Sell(userID uint, amount float64) error {
	args := m.Called(userID, amount)
	return args.Error(0)
}

func (m *MockPortfolioService) GetPortfolioSummary(userID uint) (*models.PortfolioSummaryResponse, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PortfolioSummaryResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockPortfolioService) GetPortfolioHistory(userID uint, timeRange string) (models.PortfolioHistoryResponse, error) {
	args := m.Called(userID, timeRange)
	return args.Get(0).(models.PortfolioHistoryResponse), args.Error(1)
}
