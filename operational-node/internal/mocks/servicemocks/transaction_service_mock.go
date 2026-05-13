package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) GetTransactionHistory(userID uint, page, limit int) (*models.PaginatedTransactionsResponse, error) {
	args := m.Called(userID, page, limit)
	if args.Get(0) != nil {
		return args.Get(0).(*models.PaginatedTransactionsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}
