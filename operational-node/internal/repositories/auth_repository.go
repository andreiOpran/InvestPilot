package repositories

import (
	"time"

	"gorm.io/gorm"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// AuthRepository defines the database operations for authentication
type AuthRepository interface {
	FindUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
	CreateActionToken(token *models.ActionToken) error
	FindActionToken(tokenStr, tokenType string) (*models.ActionToken, error)
	DeleteActionToken(token *models.ActionToken) error
	VerifyEmailTx(userID uint, tokenID uint) error
	FindSessionByToken(refreshToken string) (*models.Session, error)
	DeleteSessionsByFamily(familyID string) error
	DeleteSession(session *models.Session) error
	MarkSessionAsUsed(sessionID uint, originalUpdatedAt time.Time) (int64, error)
	CreateSession(session *models.Session) error
	DeleteSessionByToken(refreshToken string) error
	ResetPasswordTx(userID uint, tokenID uint, newPassword string) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) FindUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *authRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *authRepository) CreateActionToken(token *models.ActionToken) error {
	return r.db.Create(token).Error
}

func (r *authRepository) FindActionToken(tokenStr, tokenType string) (*models.ActionToken, error) {
	var token models.ActionToken
	err := r.db.Where("token = ? AND type = ?", tokenStr, tokenType).First(&token).Error
	return &token, err
}

func (r *authRepository) DeleteActionToken(token *models.ActionToken) error {
	return r.db.Delete(token).Error
}

// VerifyEmailTx wraps the user update and token deletion in a single atomic transaction
func (r *authRepository) VerifyEmailTx(userID uint, tokenID uint) error {
	tx := r.db.Begin()

	if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("is_email_verified", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.ActionToken{}, tokenID).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *authRepository) FindSessionByToken(refreshToken string) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("refresh_token = ?", refreshToken).First(&session).Error
	return &session, err
}

func (r *authRepository) DeleteSessionsByFamily(familyID string) error {
	return r.db.Where("family_id = ?", familyID).Delete(&models.Session{}).Error
}

func (r *authRepository) DeleteSession(session *models.Session) error {
	return r.db.Delete(session).Error
}

func (r *authRepository) MarkSessionAsUsed(sessionID uint, originalUpdatedAt time.Time) (int64, error) {
	res := r.db.Model(&models.Session{}).
		Where("id = ? AND updated_at = ?", sessionID, originalUpdatedAt).
		Update("is_used", true)
	return res.RowsAffected, res.Error
}

func (r *authRepository) CreateSession(session *models.Session) error {
	return r.db.Create(session).Error
}

func (r *authRepository) DeleteSessionByToken(refreshToken string) error {
	return r.db.Where("refresh_token = ?", refreshToken).Delete(&models.Session{}).Error
}

// ResetPasswordTx handles the critical transaction of updating a password and destroying sessions
func (r *authRepository) ResetPasswordTx(userID uint, tokenID uint, newPassword string) error {
	tx := r.db.Begin()

	if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("password", newPassword).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Delete(&models.ActionToken{}, tokenID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("user_id = ?", userID).Delete(&models.Session{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
