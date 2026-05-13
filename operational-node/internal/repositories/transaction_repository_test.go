package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestTransactionRepository_GetUnifiedHistory(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewTransactionRepository(db)

	t.Run("GetUnifiedHistory_emptyDB_returnsZeroAndEmptySlice", func(t *testing.T) {
		data, total, err := repo.GetUnifiedHistory(999, 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, data)
	})

	t.Run("GetUnifiedHistory_withRecords_returnsCombined", func(t *testing.T) {
		// create user + wallet as FK prerequisites
		user := &models.User{Email: "tx-repo@test.com", Password: "hash"}
		db.Create(user)
		db.Create(&models.Wallet{UserID: user.ID, Balance: 0})

		// create a funding record
		db.Create(&models.Funding{
			UserID: user.ID,
			Type:   "DEPOSIT",
			Amount: 500.0,
			Status: "COMPLETED",
		})

		// create a transaction record
		db.Create(&models.Transaction{
			UserID: user.ID,
			Type:   "INVEST",
			Amount: 300.0,
		})

		data, total, err := repo.GetUnifiedHistory(user.ID, 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, data, 2)
	})

	t.Run("GetUnifiedHistory_pagination_returnsCorrectPage", func(t *testing.T) {
		user := &models.User{Email: "tx-page@test.com", Password: "hash"}
		db.Create(user)
		db.Create(&models.Wallet{UserID: user.ID, Balance: 0})

		for i := 0; i < 5; i++ {
			db.Create(&models.Transaction{UserID: user.ID, Type: "INVEST", Amount: float64(i * 100)})
		}

		data, total, err := repo.GetUnifiedHistory(user.ID, 2, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, data, 2)
	})
}
