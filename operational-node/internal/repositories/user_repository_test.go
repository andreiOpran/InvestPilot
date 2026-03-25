package repositories

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestUserRepository_FindByID(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("FindByID_userExists_returnsUser", func(t *testing.T) {
		user := models.User{Email: "test@test.com"}
		db.Create(&user)

		foundUser, err := repo.FindByID(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, "test@test.com", foundUser.Email)
	})

	t.Run("FindByID_userNotFound_returnsError", func(t *testing.T) {
		_, err := repo.FindByID(999)
		assert.Error(t, err)
	})
}

func TestUserRepository_FindByIDWithWallet(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("FindByIDWithWallet_success_returnsPreloadedWallet", func(t *testing.T) {
		user := models.User{Email: "wallet@test.com", Wallet: models.Wallet{Balance: 250.0}}
		db.Create(&user)

		foundUser, err := repo.FindByIDWithWallet(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, 250.0, foundUser.Wallet.Balance)
	})
}

func TestUserRepository_Save(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("Save_validUser_updatesRecord", func(t *testing.T) {
		user := models.User{Email: "save@test.com"}
		db.Create(&user)

		user.RiskTolerance = 5
		err := repo.Save(&user)
		assert.NoError(t, err)

		var updated models.User
		db.First(&updated, user.ID)
		assert.Equal(t, 5, updated.RiskTolerance)
	})
}

func TestUserRepository_AddWalletBalance(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("AddWalletBalance_success_updatesBalance", func(t *testing.T) {
		user := models.User{Email: "add@test.com", Wallet: models.Wallet{Balance: 100.0}}
		db.Create(&user)

		err := repo.AddWalletBalance(user.ID, 50.5)
		assert.NoError(t, err)

		wallet, _ := repo.FindWalletByUserID(user.ID)
		assert.Equal(t, 150.5, wallet.Balance)
	})
}
