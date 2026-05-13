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

func setupDataPipelineRouter(mockSvc *servicemocks.MockDataPipelineService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewDataPipelineHandler(mockSvc)
	r.POST("/pipeline/daily", handler.RunDataPipeline)
	r.POST("/pipeline/intraday", handler.RunIntradayPipeline)
	return r
}

func TestRunDataPipelineHandler(t *testing.T) {
	t.Run("RunDataPipeline_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockDataPipelineService)
		r := setupDataPipelineRouter(mockSvc)
		mockSvc.On("RunDailyPipeline").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/pipeline/daily", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RunDataPipeline_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockDataPipelineService)
		r := setupDataPipelineRouter(mockSvc)
		mockSvc.On("RunDailyPipeline").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/pipeline/daily", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestRunIntradayPipelineHandler(t *testing.T) {
	t.Run("RunIntradayPipeline_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockDataPipelineService)
		r := setupDataPipelineRouter(mockSvc)
		mockSvc.On("RunIntradayPipeline").Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/pipeline/intraday", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RunIntradayPipeline_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockDataPipelineService)
		r := setupDataPipelineRouter(mockSvc)
		mockSvc.On("RunIntradayPipeline").Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/pipeline/intraday", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
