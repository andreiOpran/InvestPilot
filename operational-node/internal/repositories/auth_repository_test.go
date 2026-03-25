package repositories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestAuthRepository_UserOperations(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewAuthRepository(db)

	t.Run("CreateUser_success", func(t *testing.T) {
		user := &models.User{Email: "create@test.com", Password: "hashed"}
		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
	})

	t.Run("FindUserByEmail_exists_returnsUser", func(t *testing.T) {
		foundUser, err := repo.FindUserByEmail("create@test.com")
		assert.NoError(t, err)
		assert.Equal(t, "create@test.com", foundUser.Email)
	})
}

func TestAuthRepository_ActionTokenOperations(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewAuthRepository(db)

	t.Run("ActionTokens_CreateFindDelete_success", func(t *testing.T) {
		token := &models.ActionToken{
			UserID:    1,
			Token:     "secret-token",
			Type:      "verify_email",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		err := repo.CreateActionToken(token)
		assert.NoError(t, err)

		foundToken, err := repo.FindActionToken("secret-token", "verify_email")
		assert.NoError(t, err)
		assert.Equal(t, "secret-token", foundToken.Token)

		err = repo.DeleteActionToken(foundToken)
		assert.NoError(t, err)

		_, err = repo.FindActionToken("secret-token", "verify_email")
		assert.Error(t, err) // should be deleted
	})
}

func TestAuthRepository_Transactions(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewAuthRepository(db)

	t.Run("VerifyEmailTx_success_updatesUserAndDeletesToken", func(t *testing.T) {
		user := models.User{Email: "tx@test.com", IsEmailVerified: false}
		db.Create(&user)
		token := models.ActionToken{UserID: user.ID, Token: "tx-token", Type: "verify_email"}
		db.Create(&token)

		err := repo.VerifyEmailTx(user.ID, token.ID)
		assert.NoError(t, err)

		var updatedUser models.User
		db.First(&updatedUser, user.ID)
		assert.True(t, updatedUser.IsEmailVerified)
	})

	t.Run("ResetPasswordTx_success_updatesPasswordAndDeletesSessions", func(t *testing.T) {
		user := models.User{Email: "reset@test.com", Password: "old"}
		db.Create(&user)
		token := models.ActionToken{UserID: user.ID, Token: "reset-token", Type: "reset"}
		db.Create(&token)
		session := models.Session{UserID: user.ID, RefreshToken: "sess"}
		db.Create(&session)

		err := repo.ResetPasswordTx(user.ID, token.ID, "new-strong-pass")
		assert.NoError(t, err)

		var updatedUser models.User
		db.First(&updatedUser, user.ID)
		assert.Equal(t, "new-strong-pass", updatedUser.Password)

		var sessCount int64
		db.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&sessCount)
		assert.Equal(t, int64(0), sessCount) // sessions should be wiped
	})
}

func TestAuthRepository_SessionOperations(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()
	repo := NewAuthRepository(db)

	t.Run("Sessions_Lifecycle_success", func(t *testing.T) {
		session := &models.Session{
			UserID:       1,
			FamilyID:     "family-1",
			RefreshToken: "refresh-1",
		}

		err := repo.CreateSession(session)
		assert.NoError(t, err)

		foundSess, err := repo.FindSessionByToken("refresh-1")
		assert.NoError(t, err)
		assert.Equal(t, "family-1", foundSess.FamilyID)

		rows, err := repo.MarkSessionAsUsed(foundSess.ID, foundSess.UpdatedAt)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), rows)

		err = repo.DeleteSessionByToken("refresh-1")
		assert.NoError(t, err)
	})

	t.Run("DeleteSessionsByFamily_success", func(t *testing.T) {
		db.Create(&models.Session{FamilyID: "fam-kill", RefreshToken: "tok1"})
		db.Create(&models.Session{FamilyID: "fam-kill", RefreshToken: "tok2"})

		err := repo.DeleteSessionsByFamily("fam-kill")
		assert.NoError(t, err)

		var count int64
		db.Model(&models.Session{}).Where("family_id = ?", "fam-kill").Count(&count)
		assert.Equal(t, int64(0), count)
	})
}
