package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func setupForecastRouter(mockSvc *servicemocks.MockForecastService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewForecastHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.POST("/forecast", handler.RequestForecastHandler)
	r.GET("/forecast/status/:task_id", handler.GetForecastStatusHandler)
	return r
}

func float64Ptr(v float64) *float64 { return &v }

func TestRequestForecastHandler(t *testing.T) {
	t.Run("RequestForecastHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/forecast", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RequestForecastHandler_noActivePortfolio_returns422", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		payload := models.ForecastRequest{InitialInvestment: float64Ptr(1000), Years: 5}
		body, _ := json.Marshal(payload)
		mockSvc.On("RequestForecast", uint(1), payload).Return("", services.ErrForecastUserNoActivePortfolio).Once()

		req, _ := http.NewRequest(http.MethodPost, "/forecast", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RequestForecastHandler_onlyCash_returns422", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		payload := models.ForecastRequest{InitialInvestment: float64Ptr(1000), Years: 5}
		body, _ := json.Marshal(payload)
		mockSvc.On("RequestForecast", uint(1), payload).Return("", services.ErrForecastNoAssetsOnlyCash).Once()

		req, _ := http.NewRequest(http.MethodPost, "/forecast", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RequestForecastHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		payload := models.ForecastRequest{InitialInvestment: float64Ptr(1000), Years: 5}
		body, _ := json.Marshal(payload)
		mockSvc.On("RequestForecast", uint(1), payload).Return("", services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/forecast", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("RequestForecastHandler_success_returns202", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		payload := models.ForecastRequest{InitialInvestment: float64Ptr(1000), Years: 5}
		body, _ := json.Marshal(payload)
		mockSvc.On("RequestForecast", uint(1), payload).Return("task-uuid-123", nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/forecast", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.Contains(t, w.Body.String(), "task-uuid-123")
		mockSvc.AssertExpectations(t)
	})
}

func TestGetForecastStatusHandler(t *testing.T) {
	t.Run("GetForecastStatusHandler_notFound_returns404", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		mockSvc.On("GetForecast", "task-123", uint(1)).Return((*models.ForecastResult)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/forecast/status/task-123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetForecastStatusHandler_pending_returns200WithStatus", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		result := &models.ForecastResult{TaskID: "task-123", Status: "pending"}
		mockSvc.On("GetForecast", "task-123", uint(1)).Return(result, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/forecast/status/task-123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "pending")
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetForecastStatusHandler_error_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		result := &models.ForecastResult{TaskID: "task-123", Status: "error"}
		mockSvc.On("GetForecast", "task-123", uint(1)).Return(result, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/forecast/status/task-123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetForecastStatusHandler_complete_returns200WithPayload", func(t *testing.T) {
		mockSvc := new(servicemocks.MockForecastService)
		r := setupForecastRouter(mockSvc)
		result := &models.ForecastResult{TaskID: "task-123", Status: "complete", Payload: `{"p50":[1000,1100]}`}
		mockSvc.On("GetForecast", "task-123", uint(1)).Return(result, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/forecast/status/task-123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "complete")
		mockSvc.AssertExpectations(t)
	})
}
