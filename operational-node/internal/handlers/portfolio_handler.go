package handlers

import (
	"errors"
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type PortfolioHandler struct {
	portfolioService services.PortfolioService
}

func NewPortfolioHandler(portfolioService services.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
	}
}

func (h *PortfolioHandler) InvestHandler(c *gin.Context) {
	// extract userID from middleware
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDRaw.(uint)

	// bind input using InvestRequest'
	var req models.InvestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// route into service
	if err := h.portfolioService.Invest(userID, req.Amount); err != nil {
		if errors.Is(err, services.ErrInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process investment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Investment successfully added to portfolio under USD"})
}

func (h *PortfolioHandler) SellHandler(c *gin.Context) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDRaw.(uint)

	var req models.InvestRequest // reuses the same {amount} shape
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.portfolioService.Sell(userID, req.Amount); err != nil {
		switch {
		case errors.Is(err, services.ErrNoActivePortfolio):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrSellExceedsPortfolioValue):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process sell"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio liquidated and funds returned to wallet"})
}

func (h *PortfolioHandler) GetPortfolioSummaryHandler(c *gin.Context) {
	// extract userID from middleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	summary, err := h.portfolioService.GetPortfolioSummary(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch portfolio summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *PortfolioHandler) GetPortfolioHistoryHandler(c *gin.Context) {
	// extract userID from middleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// default to 1M
	timeRange := c.DefaultQuery("range", "1M")

	history, err := h.portfolioService.GetPortfolioHistory(userID.(uint), timeRange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch portfolio history"})
		return
	}

	c.JSON(http.StatusOK, history)
}
