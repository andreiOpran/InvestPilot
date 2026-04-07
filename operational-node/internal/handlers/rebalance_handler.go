package handlers

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type RebalanceHandler struct {
	service services.RebalanceService
}

func NewRebalanceHandler(service services.RebalanceService) *RebalanceHandler {
	return &RebalanceHandler{service: service}
}

// POST /rebalance triggers the monthly pipeline
func (h *RebalanceHandler) Rebalance(c *gin.Context) {
	if err := h.service.RunMonthlyRebalance(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Rebalance execution complete"})
}
