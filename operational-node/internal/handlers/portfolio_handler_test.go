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

func setupPortfolioRouter(mockSvc *servicemocks.MockPortfolioService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewPortfolioHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.POST("/invest", handler.InvestHandler)
	r.POST("/sell", handler.SellHandler)
	r.GET("/portfolio", handler.GetPortfolioSummaryHandler)
	r.GET("/portfolio/history", handler.GetPortfolioHistoryHandler)
	return r
}

func TestInvestHandler(t *testing.T) {
	t.Run("InvestHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/invest", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InvestHandler_insufficientBalance_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 1000.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Invest", uint(1), 1000.0).Return(services.ErrInsufficientBalance).Once()

		req, _ := http.NewRequest(http.MethodPost, "/invest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("InvestHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 1000.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Invest", uint(1), 1000.0).Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/invest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("InvestHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 500.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Invest", uint(1), 500.0).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/invest", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestSellHandler(t *testing.T) {
	t.Run("SellHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/sell", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("SellHandler_noActivePortfolio_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Sell", uint(1), 100.0).Return(services.ErrNoActivePortfolio).Once()

		req, _ := http.NewRequest(http.MethodPost, "/sell", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("SellHandler_exceedsPortfolioValue_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 99999.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Sell", uint(1), 99999.0).Return(services.ErrSellExceedsPortfolioValue).Once()

		req, _ := http.NewRequest(http.MethodPost, "/sell", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("SellHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		payload := models.InvestRequest{Amount: 100.0}
		body, _ := json.Marshal(payload)
		mockSvc.On("Sell", uint(1), 100.0).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/sell", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestGetPortfolioSummaryHandler(t *testing.T) {
	t.Run("GetPortfolioSummaryHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		mockSvc.On("GetPortfolioSummary", uint(1)).Return((*models.PortfolioSummaryResponse)(nil), services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/portfolio", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetPortfolioSummaryHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		summary := &models.PortfolioSummaryResponse{
			LiveTotalValue:   1500.0,
			NetContributions: 1000.0,
			Holdings:         []models.HoldingResponse{},
		}
		mockSvc.On("GetPortfolioSummary", uint(1)).Return(summary, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/portfolio", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "1500")
		mockSvc.AssertExpectations(t)
	})
}

func TestGetPortfolioHistoryHandler(t *testing.T) {
	t.Run("GetPortfolioHistoryHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		mockSvc.On("GetPortfolioHistory", uint(1), "1M").Return(models.PortfolioHistoryResponse{}, services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodGet, "/portfolio/history", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistoryHandler_withRange_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockPortfolioService)
		r := setupPortfolioRouter(mockSvc)
		history := models.PortfolioHistoryResponse{Range: "1W", Data: []models.PortfolioHistoryPoint{}}
		mockSvc.On("GetPortfolioHistory", uint(1), "1W").Return(history, nil).Once()

		req, _ := http.NewRequest(http.MethodGet, "/portfolio/history?range=1W", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
