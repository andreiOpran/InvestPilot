package repomocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByID(userID uint) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByIDWithWallet(userID uint) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) Save(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindWalletByUserID(userID uint) (*models.Wallet, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*models.Wallet), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) DepositTx(userID uint, amount float64, fundingLog *models.Funding) error {
	args := m.Called(userID, amount, fundingLog)
	return args.Error(0)
}

func (m *MockUserRepository) CashoutTx(userID uint, amount float64, fundingLog *models.Funding) error {
	args := m.Called(userID, amount, fundingLog)
	return args.Error(0)
}
