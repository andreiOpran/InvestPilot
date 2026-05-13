package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/andreiOpran/licenta/operational-node/internal/mocks/servicemocks"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
)

func setupOnboardingRouter(mockSvc *servicemocks.MockOnboardingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewOnboardingHandler(mockSvc)

	r.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	r.GET("/onboarding/questions", handler.GetQuestionsHandler)
	r.POST("/onboarding/submit", handler.SubmitHandler)
	return r
}

func TestGetQuestionsHandler(t *testing.T) {
	mockSvc := new(servicemocks.MockOnboardingService)
	r := setupOnboardingRouter(mockSvc)

	questions := []models.OnboardingQuestion{
		{ID: "q1", Text: "Question 1", Options: []models.OnboardingOption{{ID: "opt1", Text: "Option 1"}}},
	}
	mockSvc.On("GetQuestions").Return(questions).Once()

	req, _ := http.NewRequest(http.MethodGet, "/onboarding/questions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Question 1")
	mockSvc.AssertExpectations(t)
}

func TestSubmitHandler(t *testing.T) {
	t.Run("SubmitHandler_badJSON_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockOnboardingService)
		r := setupOnboardingRouter(mockSvc)
		req, _ := http.NewRequest(http.MethodPost, "/onboarding/submit", bytes.NewBuffer([]byte("{bad}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("SubmitHandler_missingAnswer_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockOnboardingService)
		r := setupOnboardingRouter(mockSvc)
		payload := models.OnboardingSubmitRequest{Answers: map[string]string{"q1": "opt1"}}
		body, _ := json.Marshal(payload)
		mockSvc.On("SubmitOnboarding", uint(1), payload).Return(services.ErrMissingAnswer).Once()

		req, _ := http.NewRequest(http.MethodPost, "/onboarding/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("SubmitHandler_invalidOption_returns400", func(t *testing.T) {
		mockSvc := new(servicemocks.MockOnboardingService)
		r := setupOnboardingRouter(mockSvc)
		payload := models.OnboardingSubmitRequest{Answers: map[string]string{"q1": "invalid"}}
		body, _ := json.Marshal(payload)
		mockSvc.On("SubmitOnboarding", uint(1), payload).Return(services.ErrInvalidOption).Once()

		req, _ := http.NewRequest(http.MethodPost, "/onboarding/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("SubmitHandler_internalError_returns500", func(t *testing.T) {
		mockSvc := new(servicemocks.MockOnboardingService)
		r := setupOnboardingRouter(mockSvc)
		payload := models.OnboardingSubmitRequest{Answers: map[string]string{"q1": "opt1"}}
		body, _ := json.Marshal(payload)
		mockSvc.On("SubmitOnboarding", uint(1), payload).Return(services.ErrInternal).Once()

		req, _ := http.NewRequest(http.MethodPost, "/onboarding/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("SubmitHandler_success_returns200", func(t *testing.T) {
		mockSvc := new(servicemocks.MockOnboardingService)
		r := setupOnboardingRouter(mockSvc)
		payload := models.OnboardingSubmitRequest{Answers: map[string]string{"q1": "opt1", "q2": "opt2"}}
		body, _ := json.Marshal(payload)
		mockSvc.On("SubmitOnboarding", uint(1), payload).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/onboarding/submit", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "completed")
		mockSvc.AssertExpectations(t)
	})
}
