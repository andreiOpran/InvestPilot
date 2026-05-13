package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestRunMonthlyRebalance(t *testing.T) {
	t.Run("RunMonthlyRebalance_marketDataDateError_returnsError", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		rebalanceRepo.On("GetLatestMarketDataDate").Return(time.Time{}, ErrInternal).Once()

		err := svc.RunMonthlyRebalance()
		assert.Error(t, err)
		rebalanceRepo.AssertExpectations(t)
	})

	t.Run("RunMonthlyRebalance_staleMarketData_returnsError", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		// very old date to trigger staleness check
		staleDate := time.Now().AddDate(-1, 0, 0)
		rebalanceRepo.On("GetLatestMarketDataDate").Return(staleDate, nil).Once()

		err := svc.RunMonthlyRebalance()
		assert.ErrorIs(t, err, ErrRebalancePausedStaleMarketData)
		rebalanceRepo.AssertExpectations(t)
	})

	t.Run("RunMonthlyRebalance_getModelPortfoliosError_returnsError", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		rebalanceRepo.On("GetLatestMarketDataDate").Return(time.Now(), nil).Once()
		rebalanceRepo.On("GetLatestModelPortfolios").Return(nil, ErrInternal).Once()

		err := svc.RunMonthlyRebalance()
		assert.Error(t, err)
		rebalanceRepo.AssertExpectations(t)
	})

	t.Run("RunMonthlyRebalance_getLatestPricesError_returnsError", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		rebalanceRepo.On("GetLatestMarketDataDate").Return(time.Now(), nil).Once()
		rebalanceRepo.On("GetLatestModelPortfolios").Return([]models.ModelPortfolio{}, nil).Once()
		rebalanceRepo.On("GetLatestPrices").Return(nil, ErrInternal).Once()

		err := svc.RunMonthlyRebalance()
		assert.Error(t, err)
		rebalanceRepo.AssertExpectations(t)
	})

	t.Run("RunMonthlyRebalance_getMaxRoundIDError_returnsError", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		rebalanceRepo.On("GetLatestMarketDataDate").Return(time.Now(), nil).Once()
		rebalanceRepo.On("GetLatestModelPortfolios").Return([]models.ModelPortfolio{}, nil).Once()
		rebalanceRepo.On("GetLatestPrices").Return([]models.DailyMarketData{}, nil).Once()
		rebalanceRepo.On("GetMaxRoundID").Return(uint(0), ErrInternal).Once()

		err := svc.RunMonthlyRebalance()
		assert.Error(t, err)
		rebalanceRepo.AssertExpectations(t)
	})

	t.Run("RunMonthlyRebalance_emptyBatch_returnsNil", func(t *testing.T) {
		rebalanceRepo := new(repomocks.MockRebalanceRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewRebalanceService(rebalanceRepo, userRepo)

		rebalanceRepo.On("GetLatestMarketDataDate").Return(time.Now(), nil).Once()
		rebalanceRepo.On("GetLatestModelPortfolios").Return([]models.ModelPortfolio{}, nil).Once()
		rebalanceRepo.On("GetLatestPrices").Return([]models.DailyMarketData{}, nil).Once()
		rebalanceRepo.On("GetMaxRoundID").Return(uint(100), nil).Once()
		rebalanceRepo.On("GetInvestmentRoundsBatchByStatus", true, uint(0), uint(100), mock.AnythingOfType("int")).
			Return([]models.InvestmentRound{}, nil).Once()

		err := svc.RunMonthlyRebalance()
		assert.NoError(t, err)
		rebalanceRepo.AssertExpectations(t)
	})
}
