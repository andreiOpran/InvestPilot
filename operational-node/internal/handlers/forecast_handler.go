package handlers

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type ForecastHandler struct {
	forecastService services.ForecastService
}

func NewForecastHandler(fs services.ForecastService) *ForecastHandler {
	return &ForecastHandler{forecastService: fs}
}

// POST /forecast
func (h *ForecastHandler) RequestForecastHandler(c *gin.Context) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDRaw.(uint)

	var req models.ForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	taskID, err := h.forecastService.RequestForecast(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// return 202 Accepted indicating task is processing asynchronously
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Forecast request accepted",
		"task_id": taskID,
	})
}
