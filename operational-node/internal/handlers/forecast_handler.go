package handlers

import (
	"encoding/json"
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

// GET /forecast/status/:task_id

func (h *ForecastHandler) GetForecastStatusHandler(c *gin.Context) {
	// auth check to ensure only logged in uesers query task IDs
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
		return
	}

	result, err := h.forecastService.GetForecastByTaskID(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Forecast task not found"})
		return
	}

	// if pending, just return status
	if result.Status == "pending" {
		c.JSON(http.StatusOK, gin.H{
			"task_id": result.TaskID,
			"status":  result.Status,
		})
		return
	}

	// if error, indicate clearly
	if result.Status == "error" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"task_id": result.TaskID,
			"status":  result.Status,
			"error":   "An error occurred during forecast computation in the math engine",
		})
		return
	}

	// if complete, parse json payload string into map so gin formats it as nested json
	var payload map[string]interface{}
	if result.Payload != "" {
		if err := json.Unmarshal([]byte(result.Payload), &payload); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse result payload form database"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"task_id": result.TaskID,
		"status":  result.Status,
		"payload": payload,
	})
}
