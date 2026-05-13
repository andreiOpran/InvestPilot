package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func setupTransactionRouter(mockSvc *servicemocks.MockTransactionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewTransactionHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.GET("/transactions", handler.GetTransactionsHandler)
	return r
}

func TestGetTransactionsHandler(t *testing.T) {
	t.Run("GetTransactionsHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockTransactionService)
		r := setupTransactionRouter(mockSvc)
		mockSvc.On("GetTransactionHistory", uint(1), 1, 10).Return((*models.PaginatedTransactionsResponse)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetTransactionsHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockTransactionService)
		r := setupTransactionRouter(mockSvc)
		resp := &models.PaginatedTransactionsResponse{
			Data:       []models.UnifiedTransaction{},
			TotalCount: 0,
			Page:       1,
			Limit:      10,
		}
		mockSvc.On("GetTransactionHistory", uint(1), 1, 10).Return(resp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetTransactionsHandler_customPagination_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockTransactionService)
		r := setupTransactionRouter(mockSvc)
		resp := &models.PaginatedTransactionsResponse{
			Data:       []models.UnifiedTransaction{},
			TotalCount: 25,
			Page:       2,
			Limit:      5,
		}
		mockSvc.On("GetTransactionHistory", uint(1), 2, 5).Return(resp, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/transactions?page=2&limit=5", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
