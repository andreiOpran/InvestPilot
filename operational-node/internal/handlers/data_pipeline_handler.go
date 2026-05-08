/*

THIS IS USED FOR K8S DEPLOYMENT. IN LOCAL DEV, THE DATA PIPELINES ARE TRIGGERED BY CRON JOB (operational-node/internal/jobs/cron_pipeline.go)

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

// RunPipeline triggers both CMD_SYNC_DAILY and CMD_GENERATE sequentially
func (h *DataPipelineHandler) RunDataPipeline(c *gin.Context) {
	if err := h.service.RunDailyPipeline(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dispatch data pipeline commands"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Daily pipeline commands (CMD_SYNC_DAILY & CMD_GENERATE) dispatched successfully"})
}

// RunIntradayPipeline triggers CMD_SYNC_INTRADAY
func (h *DataPipelineHandler) RunIntradayPipeline(c *gin.Context) {
	if err := h.service.RunIntradayPipeline(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to dispatch intraday pipeline"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Intraday pipeline (CMD_SYNC_INTRADAY) dispatched successfully"})
}
