package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestRequestForecast(t *testing.T) {
	t.Run("RequestForecast_roundError_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), ErrInternal).Once()

		req := models.ForecastRequest{Years: 5}
		_, err := svc.RequestForecast(1, req)
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("RequestForecast_noActivePortfolio_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), nil).Once()

		req := models.ForecastRequest{Years: 5}
		_, err := svc.RequestForecast(1, req)
		assert.ErrorIs(t, err, ErrForecastUserNoActivePortfolio)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("RequestForecast_emptyHoldings_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		round := &models.InvestmentRound{UserID: 1, IsActive: true, Holdings: []models.Holding{}}
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()

		req := models.ForecastRequest{Years: 5}
		_, err := svc.RequestForecast(1, req)
		assert.ErrorIs(t, err, ErrForecastUserNoActivePortfolio)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("RequestForecast_onlyCash_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "USD", Shares: 1000.0, AllocatedAmount: 1000.0},
			},
		}
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()

		req := models.ForecastRequest{Years: 5}
		_, err := svc.RequestForecast(1, req)
		assert.ErrorIs(t, err, ErrForecastNoAssetsOnlyCash)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("RequestForecast_createForecastError_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		initialInvestment := 1000.0
		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "SPY", Shares: 1.0, AllocatedAmount: 450.0},
			},
		}
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		forecastRepo.On("CreateForecast",
			mock.AnythingOfType("string"),
			uint(1),
			mock.AnythingOfType("*models.ForecastResult"),
		).Return(ErrInternal).Once()

		req := models.ForecastRequest{InitialInvestment: &initialInvestment, Years: 5}
		_, err := svc.RequestForecast(1, req)
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
		forecastRepo.AssertExpectations(t)
	})
}

func TestGetForecast(t *testing.T) {
	t.Run("GetForecast_delegatesToRepo", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		result := &models.ForecastResult{TaskID: "task-abc", UserID: 1, Status: "pending"}
		forecastRepo.On("GetForecast", "task-abc", uint(1)).Return(result, nil).Once()

		got, err := svc.GetForecast("task-abc", 1)
		assert.NoError(t, err)
		assert.Equal(t, "pending", got.Status)
		forecastRepo.AssertExpectations(t)
	})

	t.Run("GetForecast_repoError_returnsError", func(t *testing.T) {
		forecastRepo := new(repomocks.MockForecastRepository)
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		svc := NewForecastService(forecastRepo, portfolioRepo)

		forecastRepo.On("GetForecast", "bad-task", uint(1)).Return((*models.ForecastResult)(nil), ErrInternal).Once()

		got, err := svc.GetForecast("bad-task", 1)
		assert.Error(t, err)
		assert.Nil(t, got)
		forecastRepo.AssertExpectations(t)
	})
}
