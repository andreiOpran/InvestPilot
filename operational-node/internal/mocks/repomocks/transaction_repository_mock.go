package repomocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) GetUnifiedHistory(userID uint, limit, offset int) ([]models.UnifiedTransaction, int64, error) {
	args := m.Called(userID, limit, offset)
	if args.Get(0) != nil {
		return args.Get(0).([]models.UnifiedTransaction), args.Get(1).(int64), args.Error(2)
	}
	return nil, args.Get(1).(int64), args.Error(2)
}
