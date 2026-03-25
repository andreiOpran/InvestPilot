package servicemocks

import (
	"github.com/stretchr/testify/mock"
)

// MockSecurityService implementation of services.SecurityService
type MockSecurityService struct {
	mock.Mock
}

func (m *MockSecurityService) Setup2FA(userID uint) (string, string, string, error) {
	args := m.Called(userID)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockSecurityService) Enable2FA(userID uint, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}
