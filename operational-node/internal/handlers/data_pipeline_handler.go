/*

THIS IS USED ONLY FOR DEBUGGING, DATA PIPELINE IS TRIGGERED BY CRON JOB (operational-node/internal/jobs/cron_pipeline.go)

*/

package handlers

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/gin-gonic/gin"
)

type DataPipelineHandler struct {
	service services.DataPipelineService
}

func NewDataPipelineHandler(service services.DataPipelineService) *DataPipelineHandler {
	return &DataPipelineHandler{service: service}
}

// RunPipeline triggers both CMD_SYNC and CMD_GENERATE sequentially
func (h *DataPipelineHandler) RunDataPipeline(c *gin.Context) {
	if err := h.service.RunDailyPipeline(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dispatch data pipeline commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Daily pipeline commands (Sync & Generate) dispatched successfully"})
}
