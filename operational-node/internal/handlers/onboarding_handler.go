package handlers

import (
	"errors"
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type OnboardingHandler struct {
	onboardingService services.OnboardingService
}

func NewOnboardingHandler(onboardingService services.OnboardingService) *OnboardingHandler {
	return &OnboardingHandler{onboardingService: onboardingService}
}

// GetQuestionsHandler returns a dynamic list of questions
func (h *OnboardingHandler) GetQuestionsHandler(c *gin.Context) {
	questions := h.onboardingService.GetQuestions()

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
	})
}

// SubmitHandler processes the answers and updates the user profile
func (h *OnboardingHandler) SubmitHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.OnboardingSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	err := h.onboardingService.SubmitOnboarding(userID.(uint), req)
	if err != nil {
		if errors.Is(err, services.ErrMissingAnswer) || errors.Is(err, services.ErrInvalidOption) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// generic fallback for db errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Onboarding completed successfully. Profile updated.",
	})
}
