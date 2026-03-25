package repomocks

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

// MockAuthRepository implementation for testing
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) FindUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockAuthRepository) CreateActionToken(token *models.ActionToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockAuthRepository) FindActionToken(tokenStr, tokenType string) (*models.ActionToken, error) {
	args := m.Called(tokenStr, tokenType)
	if args.Get(0) != nil {
		return args.Get(0).(*models.ActionToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthRepository) DeleteActionToken(token *models.ActionToken) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockAuthRepository) VerifyEmailTx(userID uint, tokenID uint) error {
	args := m.Called(userID, tokenID)
	return args.Error(0)
}

func (m *MockAuthRepository) FindSessionByToken(refreshToken string) (*models.Session, error) {
	args := m.Called(refreshToken)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthRepository) DeleteSessionsByFamily(familyID string) error {
	args := m.Called(familyID)
	return args.Error(0)
}

func (m *MockAuthRepository) DeleteSession(session *models.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockAuthRepository) MarkSessionAsUsed(sessionID uint, originalUpdatedAt time.Time) (int64, error) {
	args := m.Called(sessionID, originalUpdatedAt)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepository) CreateSession(session *models.Session) error {
	args := m.Called(session)
	return args.Error(0)
}

func (m *MockAuthRepository) DeleteSessionByToken(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

func (m *MockAuthRepository) ResetPasswordTx(userID uint, tokenID uint, newPassword string) error {
	args := m.Called(userID, tokenID, newPassword)
	return args.Error(0)
}
