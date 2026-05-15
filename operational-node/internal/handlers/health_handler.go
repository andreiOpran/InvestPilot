package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/andreiOpran/licenta/operational-node/internal/database"
)

func HealthHandler(c *gin.Context) {
	sqlDB, err := database.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
