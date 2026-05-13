package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestGetTransactionHistory(t *testing.T) {
	t.Run("GetTransactionHistory_repoError_returnsError", func(t *testing.T) {
		mockRepo := new(repomocks.MockTransactionRepository)
		svc := NewTransactionService(mockRepo)

		mockRepo.On("GetUnifiedHistory", uint(1), 10, 0).Return(nil, int64(0), ErrInternal).Once()

		resp, err := svc.GetTransactionHistory(1, 1, 10)
		assert.Nil(t, resp)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetTransactionHistory_defaultPagination_returnsData", func(t *testing.T) {
		mockRepo := new(repomocks.MockTransactionRepository)
		svc := NewTransactionService(mockRepo)

		data := []models.UnifiedTransaction{{Type: "INVEST", Amount: 500.0}}
		mockRepo.On("GetUnifiedHistory", uint(1), 10, 0).Return(data, int64(1), nil).Once()

		resp, err := svc.GetTransactionHistory(1, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.Limit)
		assert.Equal(t, int64(1), resp.TotalCount)
		assert.Len(t, resp.Data, 1)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetTransactionHistory_page2_calculatesOffsetCorrectly", func(t *testing.T) {
		mockRepo := new(repomocks.MockTransactionRepository)
		svc := NewTransactionService(mockRepo)

		mockRepo.On("GetUnifiedHistory", uint(1), 5, 5).Return([]models.UnifiedTransaction{}, int64(8), nil).Once()

		resp, err := svc.GetTransactionHistory(1, 2, 5)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.Page)
		assert.Equal(t, 5, resp.Limit)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetTransactionHistory_invalidPage_defaultsToPage1", func(t *testing.T) {
		mockRepo := new(repomocks.MockTransactionRepository)
		svc := NewTransactionService(mockRepo)

		mockRepo.On("GetUnifiedHistory", uint(1), 10, 0).Return([]models.UnifiedTransaction{}, int64(0), nil).Once()

		resp, err := svc.GetTransactionHistory(1, 0, 10)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.Page)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetTransactionHistory_nilData_returnsEmptySlice", func(t *testing.T) {
		mockRepo := new(repomocks.MockTransactionRepository)
		svc := NewTransactionService(mockRepo)

		mockRepo.On("GetUnifiedHistory", uint(1), 10, 0).Return(nil, int64(0), nil).Once()

		resp, err := svc.GetTransactionHistory(1, 1, 10)
		assert.NoError(t, err)
		assert.NotNil(t, resp.Data)
		assert.Empty(t, resp.Data)
		mockRepo.AssertExpectations(t)
	})
}
