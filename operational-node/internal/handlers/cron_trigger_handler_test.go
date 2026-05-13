package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func setupCronRouter(mockSvc *servicemocks.MockRebalanceService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewCronTriggerHandler(mockSvc)
	r.POST("/cron/rebalance", handler.RunRebalance)
	r.POST("/cron/cleanup", handler.RunCleanup)
	return r
}

func TestRunRebalanceHandler(t *testing.T) {
	t.Run("RunRebalance_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockRebalanceService)
		r := setupCronRouter(mockSvc)
		mockSvc.On("RunMonthlyRebalance").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/cron/rebalance", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RunRebalance_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockRebalanceService)
		r := setupCronRouter(mockSvc)
		mockSvc.On("RunMonthlyRebalance").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/cron/rebalance", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

