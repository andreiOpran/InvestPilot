package repomocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockForecastRepository struct {
	mock.Mock
}

func (m *MockForecastRepository) CreateForecast(taskID string, userID uint, forecast *models.ForecastResult) error {
	args := m.Called(taskID, userID, forecast)
	return args.Error(0)
}

func (m *MockForecastRepository) GetForecast(taskID string, userID uint) (*models.ForecastResult, error) {
	args := m.Called(taskID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ForecastResult), args.Error(1)
	}
	return nil, args.Error(1)
}
