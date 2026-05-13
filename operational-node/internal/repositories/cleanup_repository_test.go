package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestCleanupRepository(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewCleanupRepository(db)

	now := time.Now()

	t.Run("DeleteExpiredActionTokens_success", func(t *testing.T) {
		// expired token
		db.Create(&models.ActionToken{Token: "token-exp", Type: "reset", ExpiresAt: now.Add(-1 * time.Hour)})
		// valid token
		db.Create(&models.ActionToken{Token: "token-valid", Type: "reset", ExpiresAt: now.Add(1 * time.Hour)})

		rows, err := repo.DeleteExpiredActionTokens(now)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows) // expecting exactly 1 token to be deleted
	})

	t.Run("DeleteExpiredSessionsBatch_success", func(t *testing.T) {
		db.Create(&models.Session{RefreshToken: "sess1", ExpiresAt: now.Add(-1 * time.Hour)})
		db.Create(&models.Session{RefreshToken: "sess2", ExpiresAt: now.Add(-2 * time.Hour)})
		db.Create(&models.Session{RefreshToken: "sess3", ExpiresAt: now.Add(1 * time.Hour)}) // should survive

		rows, err := repo.DeleteExpiredSessionsBatch(now, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), rows) // expecting exactly 2 sessions to be deleted
	})

	t.Run("DeleteOldLoginAttemptsBatch_success", func(t *testing.T) {
		old := now.Add(-48 * time.Hour)
		db.Create(&models.LoginAttempt{UserID: 1, IPAddress: "1.1.1.1", CreatedAt: old})
		db.Create(&models.LoginAttempt{UserID: 1, IPAddress: "2.2.2.2", CreatedAt: now})

		retentionDate := now.Add(-24 * time.Hour)
		rows, err := repo.DeleteOldLoginAttemptsBatch(retentionDate, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows)
	})
}
