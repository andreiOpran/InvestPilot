package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockForecastService struct {
	mock.Mock
}

func (m *MockForecastService) RequestForecast(userID uint, req models.ForecastRequest) (string, error) {
	args := m.Called(userID, req)
	return args.String(0), args.Error(1)
}

func (m *MockForecastService) GetForecast(taskID string, userID uint) (*models.ForecastResult, error) {
	args := m.Called(taskID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ForecastResult), args.Error(1)
	}
	return nil, args.Error(1)
}
