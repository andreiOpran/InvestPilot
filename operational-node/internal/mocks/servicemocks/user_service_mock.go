package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// MockUserService implementation of services.UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserProfile(userID uint) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdateUserProfile(userID uint, req models.UpdateProfileRequest) error {
	args := m.Called(userID, req)
	return args.Error(0)
}

func (m *MockUserService) DepositFunds(userID uint, amount float64) (float64, error) {
	args := m.Called(userID, amount)
	return args.Get(0).(float64), args.Error(1)
}
