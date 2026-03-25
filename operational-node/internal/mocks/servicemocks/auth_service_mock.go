package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

// MockAuthService implementation of services.AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) RegisterUser(req models.RegisterRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockAuthService) VerifyEmail(tokenString string) error {
	args := m.Called(tokenString)
	return args.Error(0)
}

func (m *MockAuthService) AuthenticateUser(email, password, clientIP, userAgent string) (*services.LoginResult, error) {
	args := m.Called(email, password, clientIP, userAgent)
	if args.Get(0) != nil {
		return args.Get(0).(*services.LoginResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Verify2FA(email, password, totpToken, clientIP, userAgent string) (string, string, error) {
	args := m.Called(email, password, totpToken, clientIP, userAgent)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) RefreshToken(refreshTokenStr, clientIP, userAgent string) (string, string, error) {
	args := m.Called(refreshTokenStr, clientIP, userAgent)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) LogoutUser(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

func (m *MockAuthService) ForgotPassword(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthService) ResetPassword(tokenStr, newPassword string) error {
	args := m.Called(tokenStr, newPassword)
	return args.Error(0)
}
