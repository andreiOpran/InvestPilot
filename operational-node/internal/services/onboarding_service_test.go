package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/repomocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

func TestGetQuestions(t *testing.T) {
	mockRepo := new(repomocks.MockUserRepository)
	svc := NewOnboardingService(mockRepo)

	questions := svc.GetQuestions()
	assert.Len(t, questions, 3)
	assert.Equal(t, "q1_age", questions[0].ID)
	assert.Equal(t, "q2_goal", questions[1].ID)
	assert.Equal(t, "q3_drop", questions[2].ID)
	for _, q := range questions {
		assert.NotEmpty(t, q.Options)
	}
}

func TestSubmitOnboarding(t *testing.T) {
	validAnswers := map[string]string{
		"q1_age":  "age_20",
		"q2_goal": "goal_growth",
		"q3_drop": "drop_buy",
	}

	t.Run("SubmitOnboarding_missingAnswer_returnsErrMissingAnswer", func(t *testing.T) {
		mockRepo := new(repomocks.MockUserRepository)
		svc := NewOnboardingService(mockRepo)

		req := models.OnboardingSubmitRequest{Answers: map[string]string{
			"q1_age":  "age_20",
			"q2_goal": "goal_growth",
			// q3_drop missing
		}}
		err := svc.SubmitOnboarding(1, req)
		assert.ErrorIs(t, err, ErrMissingAnswer)
	})

	t.Run("SubmitOnboarding_invalidOption_returnsErrInvalidOption", func(t *testing.T) {
		mockRepo := new(repomocks.MockUserRepository)
		svc := NewOnboardingService(mockRepo)

		req := models.OnboardingSubmitRequest{Answers: map[string]string{
			"q1_age":  "not_a_real_option",
			"q2_goal": "goal_growth",
			"q3_drop": "drop_buy",
		}}
		err := svc.SubmitOnboarding(1, req)
		assert.ErrorIs(t, err, ErrInvalidOption)
	})

	t.Run("SubmitOnboarding_findByIDError_returnsError", func(t *testing.T) {
		mockRepo := new(repomocks.MockUserRepository)
		svc := NewOnboardingService(mockRepo)

		mockRepo.On("FindByID", uint(1)).Return((*models.User)(nil), ErrInternal).Once()

		req := models.OnboardingSubmitRequest{Answers: validAnswers}
		err := svc.SubmitOnboarding(1, req)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("SubmitOnboarding_saveError_returnsError", func(t *testing.T) {
		mockRepo := new(repomocks.MockUserRepository)
		svc := NewOnboardingService(mockRepo)

		user := &models.User{RiskTolerance: 0, InvestmentHorizon: 0}
		mockRepo.On("FindByID", uint(1)).Return(user, nil).Once()
		mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(ErrInternal).Once()

		req := models.OnboardingSubmitRequest{Answers: validAnswers}
		err := svc.SubmitOnboarding(1, req)
		assert.Error(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("SubmitOnboarding_success_updatesRiskAndHorizon", func(t *testing.T) {
		mockRepo := new(repomocks.MockUserRepository)
		svc := NewOnboardingService(mockRepo)

		user := &models.User{}
		mockRepo.On("FindByID", uint(1)).Return(user, nil).Once()
		mockRepo.On("Save", mock.AnythingOfType("*models.User")).Return(nil).Once()

		req := models.OnboardingSubmitRequest{Answers: validAnswers}
		err := svc.SubmitOnboarding(1, req)
		assert.NoError(t, err)
		// user object should have been mutated before Save
		assert.Greater(t, user.RiskTolerance, 0)
		mockRepo.AssertExpectations(t)
	})
}
