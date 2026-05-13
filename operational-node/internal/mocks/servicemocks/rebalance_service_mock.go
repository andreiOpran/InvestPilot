package servicemocks

import "github.com/stretchr/testify/mock"

type MockRebalanceService struct {
	mock.Mock
}

func (m *MockRebalanceService) RunMonthlyRebalance() error {
	args := m.Called()
	return args.Error(0)
}
