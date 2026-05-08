package handlers

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/jobs"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type CronTriggerHandler struct {
	rebalanceService services.RebalanceService
}

func NewCronTriggerHandler(rebalanceService services.RebalanceService) *CronTriggerHandler {
	return &CronTriggerHandler{rebalanceService: rebalanceService}
}

func (h *CronTriggerHandler) RunRebalance(c *gin.Context) {
	if err := h.rebalanceService.RunMonthlyRebalance(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Monthly rebalance completed"})
}

func (h *CronTriggerHandler) RunCleanup(c *gin.Context) {
	jobs.ExecuteTokenCleanup()
	c.JSON(http.StatusOK, gin.H{"message": "Token cleanup completed"})
}
