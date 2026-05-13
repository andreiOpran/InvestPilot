package servicemocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type MockOnboardingService struct {
	mock.Mock
}

func (m *MockOnboardingService) GetQuestions() []models.OnboardingQuestion {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).([]models.OnboardingQuestion)
	}
	return nil
}

func (m *MockOnboardingService) SubmitOnboarding(userID uint, req models.OnboardingSubmitRequest) error {
	args := m.Called(userID, req)
	return args.Error(0)
}
