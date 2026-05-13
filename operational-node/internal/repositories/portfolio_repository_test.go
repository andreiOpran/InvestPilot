package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func seedPortfolioUser(t *testing.T, db interface{ Create(interface{}) interface{ Error() error } }) {
	t.Helper()
}

func TestPortfolioRepository_GetRoundWithHoldingsByStatus(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	t.Run("notFound_returnsNil", func(t *testing.T) {
		round, err := repo.GetRoundWithHoldingsByStatus(999, true)
		assert.NoError(t, err)
		assert.Nil(t, round)
	})

	t.Run("found_returnsRound", func(t *testing.T) {
		user := &models.User{Email: "port-round@test.com", Password: "hash"}
		db.Create(user)

		r := &models.InvestmentRound{
			UserID:     user.ID,
			TotalValue: 1000.0,
			IsActive:   true,
		}
		db.Create(r)
		db.Create(&models.Holding{UserID: user.ID, InvestmentRoundID: r.ID, Ticker: "USD", Shares: 1000.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 1000.0})

		result, err := repo.GetRoundWithHoldingsByStatus(user.ID, true)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1000.0, result.TotalValue)
		assert.Len(t, result.Holdings, 1)
	})
}

func TestPortfolioRepository_GetInvestTransactions(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-tx@test.com", Password: "hash"}
	db.Create(user)

	db.Create(&models.Transaction{UserID: user.ID, Type: "INVEST", Amount: 500.0})
	db.Create(&models.Transaction{UserID: user.ID, Type: "SELL", Amount: 100.0})
	db.Create(&models.Transaction{UserID: user.ID, Type: "OTHER", Amount: 10.0})

	txs, err := repo.GetInvestTransactions(user.ID)
	assert.NoError(t, err)
	assert.Len(t, txs, 2)
	for _, tx := range txs {
		assert.Contains(t, []string{"INVEST", "SELL"}, tx.Type)
	}
}

func TestPortfolioRepository_GetHistoricalFundings(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-funding@test.com", Password: "hash"}
	db.Create(user)

	db.Create(&models.Funding{UserID: user.ID, Type: "DEPOSIT", Amount: 200.0, Status: "COMPLETED"})
	db.Create(&models.Funding{UserID: user.ID, Type: "DEPOSIT", Amount: 50.0, Status: "PENDING"})

	fundings, err := repo.GetHistoricalFundings(user.ID)
	assert.NoError(t, err)
	assert.Len(t, fundings, 1)
	assert.Equal(t, 200.0, fundings[0].Amount)
}

func TestPortfolioRepository_ExecuteInvestTransaction(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-invest@test.com", Password: "hash"}
	db.Create(user)
	wallet := &models.Wallet{UserID: user.ID, Balance: 1000.0}
	db.Create(wallet)

	wallet.Balance = 500.0
	txRecord := &models.Transaction{UserID: user.ID, Type: "INVEST", Amount: 500.0}
	newRound := &models.InvestmentRound{
		UserID:     user.ID,
		TotalValue: 500.0,
		IsActive:   true,
		Holdings: []models.Holding{
			{UserID: user.ID, Ticker: "USD", Shares: 500.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 500.0},
		},
	}

	err := repo.ExecuteInvestTransaction(wallet, txRecord, nil, newRound)
	assert.NoError(t, err)
	assert.NotZero(t, newRound.ID)

	// verify wallet was updated
	var updatedWallet models.Wallet
	db.First(&updatedWallet, wallet.ID)
	assert.Equal(t, 500.0, updatedWallet.Balance)
}

func TestPortfolioRepository_ExecuteInvestTransaction_withOldRound(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-invest2@test.com", Password: "hash"}
	db.Create(user)
	wallet := &models.Wallet{UserID: user.ID, Balance: 500.0}
	db.Create(wallet)

	oldRound := &models.InvestmentRound{UserID: user.ID, TotalValue: 500.0, IsActive: true}
	db.Create(oldRound)
	oldRound.IsActive = false

	wallet.Balance = 200.0
	txRecord := &models.Transaction{UserID: user.ID, Type: "INVEST", Amount: 200.0}
	newRound := &models.InvestmentRound{
		UserID:     user.ID,
		TotalValue: 700.0,
		IsActive:   true,
		Holdings: []models.Holding{
			{UserID: user.ID, Ticker: "USD", Shares: 700.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 700.0},
		},
	}

	err := repo.ExecuteInvestTransaction(wallet, txRecord, oldRound, newRound)
	assert.NoError(t, err)

	var updatedOld models.InvestmentRound
	db.First(&updatedOld, oldRound.ID)
	assert.False(t, updatedOld.IsActive)
}

func TestPortfolioRepository_ExecuteSellTransaction(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-sell@test.com", Password: "hash"}
	db.Create(user)
	wallet := &models.Wallet{UserID: user.ID, Balance: 0.0}
	db.Create(wallet)

	oldRound := &models.InvestmentRound{UserID: user.ID, TotalValue: 500.0, IsActive: true}
	db.Create(oldRound)
	oldRound.IsActive = false

	wallet.Balance = 100.0
	txRecord := &models.Transaction{UserID: user.ID, Type: "SELL", Amount: 100.0}

	err := repo.ExecuteSellTransaction(wallet, txRecord, oldRound, nil)
	assert.NoError(t, err)

	// verify wallet updated
	var updatedWallet models.Wallet
	db.First(&updatedWallet, wallet.ID)
	assert.Equal(t, 100.0, updatedWallet.Balance)

	// verify old round deactivated
	var updatedRound models.InvestmentRound
	db.First(&updatedRound, oldRound.ID)
	assert.False(t, updatedRound.IsActive)
}

func TestPortfolioRepository_ExecuteSellTransaction_withNewRound(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-sell2@test.com", Password: "hash"}
	db.Create(user)
	wallet := &models.Wallet{UserID: user.ID, Balance: 0.0}
	db.Create(wallet)

	oldRound := &models.InvestmentRound{UserID: user.ID, TotalValue: 500.0, IsActive: true}
	db.Create(oldRound)
	oldRound.IsActive = false

	wallet.Balance = 100.0
	txRecord := &models.Transaction{UserID: user.ID, Type: "SELL", Amount: 100.0}
	newRound := &models.InvestmentRound{
		UserID:     user.ID,
		TotalValue: 400.0,
		IsActive:   true,
		Holdings: []models.Holding{
			{UserID: user.ID, Ticker: "USD", Shares: 400.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 400.0},
		},
	}

	err := repo.ExecuteSellTransaction(wallet, txRecord, oldRound, newRound)
	assert.NoError(t, err)
	assert.NotZero(t, newRound.ID)
}

func TestPortfolioRepository_GetHistoricalRounds(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-history@test.com", Password: "hash"}
	db.Create(user)

	// create active round
	r := &models.InvestmentRound{UserID: user.ID, TotalValue: 500.0, IsActive: true}
	db.Create(r)

	since := time.Now().AddDate(0, -1, 0)
	rounds, err := repo.GetHistoricalRounds(user.ID, since)
	assert.NoError(t, err)
	assert.NotEmpty(t, rounds)
}

func TestPortfolioRepository_GetPricingData(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	// seed a daily market data entry
	db.Create(&models.DailyMarketData{
		Ticker:     "AAPL",
		Date:       time.Now().AddDate(0, 0, -1),
		ClosePrice: 180.0,
	})

	since := time.Now().AddDate(0, 0, -7)
	pricing, err := repo.GetPricingData([]string{"AAPL"}, since, false)
	assert.NoError(t, err)
	assert.Contains(t, pricing, "AAPL")
	assert.NotEmpty(t, pricing["AAPL"])
}

func TestPortfolioRepository_GetMarketTimestamps(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	db.Create(&models.DailyMarketData{
		Ticker:     "SPY",
		Date:       time.Now().AddDate(0, 0, -2),
		ClosePrice: 450.0,
	})

	since := time.Now().AddDate(0, 0, -7)
	timestamps, err := repo.GetMarketTimestamps(since, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, timestamps)
}

func TestPortfolioRepository_GetMarketTimestamps_intraday(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	db.Create(&models.IntradayMarketData{
		Ticker:    "SPY",
		Timestamp: time.Now().AddDate(0, 0, -1),
		Price:     450.0,
	})

	since := time.Now().AddDate(0, 0, -7)
	timestamps, err := repo.GetMarketTimestamps(since, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, timestamps)
}

func TestPortfolioRepository_GetInvestTransactions_empty(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewPortfolioRepository(db)

	user := &models.User{Email: "port-txempty@test.com", Password: "hash"}
	db.Create(user)

	txs, err := repo.GetInvestTransactions(user.ID)
	assert.NoError(t, err)
	assert.Empty(t, txs)
}
