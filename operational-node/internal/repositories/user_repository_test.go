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

func TestUserRepository_DepositTx(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("DepositTx_success_updatesBalanceAndCreatesFundingLog", func(t *testing.T) {
		user := models.User{Email: "deposit@test.com", Wallet: models.Wallet{Balance: 100.0}}
		db.Create(&user)

		funding := &models.Funding{
			UserID:          user.ID,
			Type:            "DEPOSIT",
			Amount:          50.0,
			StripePaymentID: "pi_test_1",
			Status:          "COMPLETED",
		}
		err := repo.DepositTx(user.ID, 50.0, funding)
		assert.NoError(t, err)

		var wallet models.Wallet
		db.Where("user_id = ?", user.ID).First(&wallet)
		assert.InDelta(t, 150.0, wallet.Balance, 0.01)

		var count int64
		db.Model(&models.Funding{}).Where("user_id = ? AND stripe_payment_id = ?", user.ID, "pi_test_1").Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestUserRepository_CashoutTx(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("CashoutTx_success_updatesBalance", func(t *testing.T) {
		user := models.User{Email: "cashout@test.com", Wallet: models.Wallet{Balance: 200.0}}
		db.Create(&user)

		funding := &models.Funding{
			UserID:          user.ID,
			Type:            "WITHDRAWAL",
			Amount:          75.0,
			StripePaymentID: "sim_out_1",
			Status:          "COMPLETED",
		}
		err := repo.CashoutTx(user.ID, 75.0, funding)
		assert.NoError(t, err)

		var wallet models.Wallet
		db.Where("user_id = ?", user.ID).First(&wallet)
		assert.InDelta(t, 125.0, wallet.Balance, 0.01)
	})

	t.Run("CashoutTx_insufficientFunds_returnsError", func(t *testing.T) {
		user := models.User{Email: "broke@test.com", Wallet: models.Wallet{Balance: 10.0}}
		db.Create(&user)

		funding := &models.Funding{
			UserID: user.ID,
			Type:   "WITHDRAWAL",
			Amount: 500.0,
			Status: "COMPLETED",
		}
		err := repo.CashoutTx(user.ID, 500.0, funding)
		assert.ErrorIs(t, err, ErrUserCashoutInsufficientFunds)
	})
}

func TestUserRepository_FindWalletByUserID(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewUserRepository(db)

	t.Run("FindWalletByUserID_success_returnsWallet", func(t *testing.T) {
		user := models.User{Email: "fw@test.com", Wallet: models.Wallet{Balance: 99.0}}
		db.Create(&user)

		wallet, err := repo.FindWalletByUserID(user.ID)
		assert.NoError(t, err)
		assert.InDelta(t, 99.0, wallet.Balance, 0.01)
	})
}
