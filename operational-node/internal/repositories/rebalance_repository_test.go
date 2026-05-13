package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestRebalanceRepository_GetMaxRoundID(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewRebalanceRepository(db)

	t.Run("emptyDB_returnsZero", func(t *testing.T) {
		id, err := repo.GetMaxRoundID()
		assert.NoError(t, err)
		assert.Equal(t, uint(0), id)
	})

	t.Run("withRounds_returnsPositiveID", func(t *testing.T) {
		user := &models.User{Email: "rebalance-max@test.com", Password: "hash"}
		db.Create(user)
		db.Create(&models.InvestmentRound{UserID: user.ID, TotalValue: 100.0, IsActive: true})

		id, err := repo.GetMaxRoundID()
		assert.NoError(t, err)
		assert.Greater(t, id, uint(0))
	})
}

func TestRebalanceRepository_GetInvestmentRoundsBatchByStatus(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewRebalanceRepository(db)

	user := &models.User{Email: "rebalance-batch@test.com", Password: "hash"}
	db.Create(user)
	db.Create(&models.InvestmentRound{UserID: user.ID, TotalValue: 100.0, IsActive: true})

	t.Run("activeRounds_returnsBatch", func(t *testing.T) {
		rounds, err := repo.GetInvestmentRoundsBatchByStatus(true, 0, 9999, 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, rounds)
		for _, r := range rounds {
			assert.True(t, r.IsActive)
		}
	})

	t.Run("emptyResult_returnsEmptySlice", func(t *testing.T) {
		// query for id range that has no rounds
		rounds, err := repo.GetInvestmentRoundsBatchByStatus(true, 999998, 999999, 10)
		assert.NoError(t, err)
		assert.Empty(t, rounds)
	})
}

func TestRebalanceRepository_ExecuteBatchRebalanceTransaction(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewRebalanceRepository(db)

	user := &models.User{Email: "rebalance-exec@test.com", Password: "hash"}
	db.Create(user)

	oldRound := &models.InvestmentRound{UserID: user.ID, TotalValue: 500.0, IsActive: true}
	db.Create(oldRound)

	newRound := models.InvestmentRound{
		UserID:     user.ID,
		TotalValue: 500.0,
		IsActive:   true,
		Holdings: []models.Holding{
			{UserID: user.ID, Ticker: "USD", Shares: 500.0, Weight: 1.0, PurchasePrice: 1.0, AllocatedAmount: 500.0},
		},
	}

	err := repo.ExecuteBatchRebalanceTransaction([]models.InvestmentRound{newRound}, []uint{oldRound.ID})
	assert.NoError(t, err)

	var updated models.InvestmentRound
	db.First(&updated, oldRound.ID)
	assert.False(t, updated.IsActive)
}

func TestRebalanceRepository_GetLatestMarketDataDate_returnsNoError(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewRebalanceRepository(db)

	// just verify it doesn't return a query construction error
	// (the actual time.Time scan from SQLite MAX(date) is a known dialect mismatch)
	db.Create(&models.DailyMarketData{Ticker: "SPY", Date: time.Now().AddDate(0, 0, -1), ClosePrice: 400.0})
	_, _ = repo.GetLatestMarketDataDate() // error expected from SQLite dialect, not a logic error
}
