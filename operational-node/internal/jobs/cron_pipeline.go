package jobs

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/robfig/cron/v3"
)

// StartDataPipelineJob schedules the CMD_SYNC and CMD_GENERATE messages
func StartDataPipelineJob(pipelineService services.DataPipelineService) {
	c := cron.New()
	schedule := config.Env.DataPipelineCronSchedule

	// run every day at configured schedule (preferably after US market close)
	_, err := c.AddFunc(schedule, func() {
		log.Println("[CRON-JOB] Starting daily data pipeline (CMD_SYNC & CMD_GENERATE)")

		if err := pipelineService.RunDailyPipeline(); err != nil {
			log.Printf("[CRON-ERROR] Daily data pipeline failed: %v", err)
		} else {
			log.Println("[CRON-JOB] Daily data pipeline dispatched successfully")
		}

	})

	if err != nil {
		log.Fatalf("CRON-ERROR] Failed to schedule data pipeline cron: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] Scheduled data pipeline cron job")
}
