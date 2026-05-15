package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestInvest(t *testing.T) {
	t.Run("Invest_walletError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		userRepo.On("FindWalletByUserID", uint(1)).Return((*models.Wallet)(nil), ErrInternal).Once()

		err := svc.Invest(1, 500.0)
		assert.ErrorIs(t, err, ErrInternal)
		userRepo.AssertExpectations(t)
	})

	t.Run("Invest_insufficientBalance_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		wallet := &models.Wallet{Balance: 100.0}
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()

		err := svc.Invest(1, 500.0)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
		userRepo.AssertExpectations(t)
	})

	t.Run("Invest_noExistingRound_success", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		wallet := &models.Wallet{Balance: 1000.0}
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), nil).Once()
		portfolioRepo.On("ExecuteInvestTransaction",
			mock.AnythingOfType("*models.Wallet"),
			mock.AnythingOfType("*models.Transaction"),
			(*models.InvestmentRound)(nil),
			mock.AnythingOfType("*models.InvestmentRound"),
		).Return(nil).Once()

		err := svc.Invest(1, 500.0)
		assert.NoError(t, err)
		userRepo.AssertExpectations(t)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("Invest_existingRound_addsToUSD", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		wallet := &models.Wallet{Balance: 1000.0}
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()

		existingRound := &models.InvestmentRound{
			UserID:     1,
			TotalValue: 500.0,
			IsActive:   true,
			Holdings: []models.Holding{
				{UserID: 1, Ticker: "USD", Shares: 500.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 500.0},
			},
		}
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(existingRound, nil).Once()
		portfolioRepo.On("ExecuteInvestTransaction",
			mock.AnythingOfType("*models.Wallet"),
			mock.AnythingOfType("*models.Transaction"),
			mock.AnythingOfType("*models.InvestmentRound"),
			mock.AnythingOfType("*models.InvestmentRound"),
		).Return(nil).Once()

		err := svc.Invest(1, 300.0)
		assert.NoError(t, err)
		portfolioRepo.AssertExpectations(t)
	})
}

func TestSell(t *testing.T) {
	t.Run("Sell_noActiveRound_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), nil).Once()

		err := svc.Sell(1, 100.0)
		assert.ErrorIs(t, err, ErrNoActivePortfolio)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("Sell_exceedsPortfolioValue_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "USD", Shares: 100.0},
			},
		}
		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		portfolioRepo.On("GetLatestPrices", mock.Anything).Return(map[string]float64{}, nil).Once()

		err := svc.Sell(1, 99999.0)
		assert.ErrorIs(t, err, ErrSellExceedsPortfolioValue)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("Sell_USDOnly_success", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "USD", Shares: 500.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 500.0},
			},
		}
		wallet := &models.Wallet{UserID: 1, Balance: 0.0}

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		portfolioRepo.On("GetLatestPrices", mock.Anything).Return(map[string]float64{}, nil).Once()
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()
		portfolioRepo.On("ExecuteSellTransaction",
			mock.AnythingOfType("*models.Wallet"),
			mock.AnythingOfType("*models.Transaction"),
			mock.AnythingOfType("*models.InvestmentRound"),
			mock.Anything,
		).Return(nil).Once()

		err := svc.Sell(1, 100.0)
		assert.NoError(t, err)
		assert.Equal(t, 100.0, wallet.Balance)
		portfolioRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("Sell_ETFHoldings_proportionalSell", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		// portfolio: only ETF holdings, no USD
		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "SPY", Shares: 2.0, Weight: 1.0, PurchasePrice: 400.0, AllocatedAmount: 800.0},
			},
		}
		wallet := &models.Wallet{UserID: 1, Balance: 0.0}
		prices := map[string]float64{"SPY": 500.0}

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		portfolioRepo.On("GetLatestPrices", mock.Anything).Return(prices, nil).Once()
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()
		portfolioRepo.On("ExecuteSellTransaction",
			mock.AnythingOfType("*models.Wallet"),
			mock.AnythingOfType("*models.Transaction"),
			mock.AnythingOfType("*models.InvestmentRound"),
			mock.Anything,
		).Return(nil).Once()

		err := svc.Sell(1, 200.0) // sell 200 from 1000 total (SPY @ 500)
		assert.NoError(t, err)
		portfolioRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("Sell_fullLiquidation_returnsNilNewRound", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "USD", Shares: 100.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 100.0},
			},
		}
		wallet := &models.Wallet{UserID: 1, Balance: 0.0}

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		portfolioRepo.On("GetLatestPrices", mock.Anything).Return(map[string]float64{}, nil).Once()
		userRepo.On("FindWalletByUserID", uint(1)).Return(wallet, nil).Once()
		portfolioRepo.On("ExecuteSellTransaction",
			mock.AnythingOfType("*models.Wallet"),
			mock.AnythingOfType("*models.Transaction"),
			mock.AnythingOfType("*models.InvestmentRound"),
			(*models.InvestmentRound)(nil),
		).Return(nil).Once()

		err := svc.Sell(1, 100.0) // sell everything
		assert.NoError(t, err)
		portfolioRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})
}

func TestGetPortfolioSummary(t *testing.T) {
	t.Run("GetPortfolioSummary_roundError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), ErrInternal).Once()

		resp, err := svc.GetPortfolioSummary(1)
		assert.Nil(t, resp)
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioSummary_noActiveRound_returnsZeroSummary", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return((*models.InvestmentRound)(nil), nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()

		resp, err := svc.GetPortfolioSummary(1)
		assert.NoError(t, err)
		assert.Equal(t, 0.0, resp.LiveTotalValue)
		assert.Empty(t, resp.Holdings)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioSummary_withActiveRound_returnsLiveValue", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		round := &models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{
				{Ticker: "USD", Shares: 200.0, Weight: 0.4, PurchasePrice: 1.0},
				{Ticker: "SPY", Shares: 1.0, Weight: 0.6, PurchasePrice: 400.0},
			},
		}
		txs := []models.Transaction{
			{Type: "INVEST", Amount: 600.0},
		}

		portfolioRepo.On("GetRoundWithHoldingsByStatus", uint(1), true).Return(round, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return(txs, nil).Once()
		portfolioRepo.On("GetLatestPrices", mock.Anything).Return(map[string]float64{"SPY": 450.0}, nil).Once()

		resp, err := svc.GetPortfolioSummary(1)
		assert.NoError(t, err)
		assert.Equal(t, 650.0, resp.LiveTotalValue) // 200 USD + 1*450 SPY
		assert.Equal(t, 600.0, resp.NetContributions)
		assert.Len(t, resp.Holdings, 2)
		portfolioRepo.AssertExpectations(t)
	})
}

func TestGetPortfolioHistory(t *testing.T) {
	t.Run("GetPortfolioHistory_roundsError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return(nil, ErrInternal).Once()

		_, err := svc.GetPortfolioHistory(1, "1M")
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_noRounds_returnsEmptyHistory", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1M")
		assert.NoError(t, err)
		assert.Equal(t, "1M", resp.Range)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_intraDayRange_callsGetPricingDataWithIntraday", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		// empty intraday -> triggers fallback GetPricingData call
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string][]models.AssetPricePoint{}, nil).Maybe()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), true).Return([]time.Time{}, nil).Once()
		// empty intraday market timestamps -> triggers fallback
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), true).Return([]time.Time{}, nil).Maybe()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1D")
		assert.NoError(t, err)
		assert.Equal(t, "1D", resp.Range)
	})

	t.Run("GetPortfolioHistory_pricingDataError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(nil, ErrInternal).Once()

		_, err := svc.GetPortfolioHistory(1, "1M")
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_getInvestTransactionsError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return(nil, ErrInternal).Once()

		_, err := svc.GetPortfolioHistory(1, "1M")
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_getMarketTimestampsError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return(nil, ErrInternal).Once()

		_, err := svc.GetPortfolioHistory(1, "1M")
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_getPricesBeforeWindowError_returnsError", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(nil, ErrInternal).Once()

		_, err := svc.GetPortfolioHistory(1, "1M")
		assert.Error(t, err)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_1W_range_setsIntradayTrue", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string][]models.AssetPricePoint{}, nil).Maybe()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), true).Return([]time.Time{}, nil).Maybe()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1W")
		assert.NoError(t, err)
		assert.Equal(t, "1W", resp.Range)
	})

	t.Run("GetPortfolioHistory_6M_range", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "6M")
		assert.NoError(t, err)
		assert.Equal(t, "6M", resp.Range)
	})

	t.Run("GetPortfolioHistory_1Y_range", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1Y")
		assert.NoError(t, err)
		assert.Equal(t, "1Y", resp.Range)
	})

	t.Run("GetPortfolioHistory_YTD_range", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "YTD")
		assert.NoError(t, err)
		assert.Equal(t, "YTD", resp.Range)
	})

	t.Run("GetPortfolioHistory_5Y_range", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "5Y")
		assert.NoError(t, err)
		assert.Equal(t, "5Y", resp.Range)
	})

	t.Run("GetPortfolioHistory_unknownRange_defaultsTo1M", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return([]time.Time{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "INVALID")
		assert.NoError(t, err)
		assert.Equal(t, "INVALID", resp.Range)
	})

	t.Run("GetPortfolioHistory_nonUSD_intradayFallback_buildsTimeSeries", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		now := time.Now()
		latestDay := now.AddDate(0, 0, -3).Truncate(24 * time.Hour)
		ts1 := latestDay.Add(14 * time.Hour) // some intraday ts

		round := models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{{Ticker: "SPY", Shares: 2.0, PurchasePrice: 400.0}},
		}

		// first round fetch
		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{round}, nil).Once()
		// first GetPricingData -> empty (simulates intraday weekend/holiday gap)
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		// fallback GetPricingData -> returns data for latestDay
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(
			map[string][]models.AssetPricePoint{
				"SPY": {{Timestamp: ts1, Price: 410.0}},
			}, nil).Once()
		// re-fetch rounds after effectiveSince update
		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{round}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string]float64{"SPY": 400.0}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1D")
		assert.NoError(t, err)
		assert.Equal(t, "1D", resp.Range)
		portfolioRepo.AssertExpectations(t)
	})

	t.Run("GetPortfolioHistory_USDOnly_intradayFallback_emptyTimestamps", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		// USD-only round -> tickers empty -> calls GetMarketTimestamps with isIntraday=true
		round := models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{{Ticker: "USD", Shares: 200.0}},
		}
		// first GetMarketTimestamps (intraday=true) returns empty -> triggers fallback
		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{round}, nil).Maybe()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string][]models.AssetPricePoint{}, nil).Maybe()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), true).Return([]time.Time{}, nil).Maybe()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), true).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1D")
		assert.NoError(t, err)
		assert.Equal(t, "1D", resp.Range)
	})

	t.Run("GetPortfolioHistory_withUSDRoundAndMarketTimestamps_buildsTimeSeries", func(t *testing.T) {
		portfolioRepo := new(repomocks.MockPortfolioRepository)
		userRepo := new(repomocks.MockUserRepository)
		svc := NewPortfolioService(portfolioRepo, userRepo)

		now := time.Now()

		round := models.InvestmentRound{
			UserID:   1,
			IsActive: true,
			Holdings: []models.Holding{{Ticker: "USD", Shares: 500.0}},
		}

		tx := models.Transaction{Type: "INVEST", Amount: 500.0}
		marketTS := []time.Time{now.AddDate(0, 0, -3), now.AddDate(0, 0, -2)}

		portfolioRepo.On("GetHistoricalRounds", uint(1), mock.AnythingOfType("time.Time")).Return([]models.InvestmentRound{round}, nil).Once()
		portfolioRepo.On("GetPricingData", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string][]models.AssetPricePoint{}, nil).Once()
		portfolioRepo.On("GetInvestTransactions", uint(1)).Return([]models.Transaction{tx}, nil).Once()
		portfolioRepo.On("GetMarketTimestamps", mock.AnythingOfType("time.Time"), false).Return(marketTS, nil).Once()
		portfolioRepo.On("GetPricesBeforeWindow", mock.Anything, mock.AnythingOfType("time.Time"), false).Return(map[string]float64{}, nil).Once()

		resp, err := svc.GetPortfolioHistory(1, "1M")
		assert.NoError(t, err)
		assert.Equal(t, "1M", resp.Range)
		assert.NotEmpty(t, resp.Data)
		portfolioRepo.AssertExpectations(t)
	})
}
