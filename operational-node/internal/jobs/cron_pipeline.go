package jobs

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/robfig/cron/v3"
)

// StartDataPipelineJob schedules the CMD_SYNC_DAILY and CMD_GENERATE as a pair,
// and then schedules CMD_SYNC_INTRADAY, each with their own configured schedule
func StartDataPipelineJob(pipelineService services.DataPipelineService) {
	c := cron.New()

	// DAILY PIPELINE (CMD_SYNC_DAILY & CMD_GENERATE)
	// run every day at configured schedule (preferably after US market close)
	dailySchedule := config.Env.DailyDataPipelineCronSchedule
	_, err := c.AddFunc(dailySchedule, func() {
		log.Println("[CRON-JOB] Starting daily data pipeline (CMD_SYNC_DAILY & CMD_GENERATE)")

		if err := pipelineService.RunDailyPipeline(); err != nil {
			log.Printf("[CRON-ERROR] Daily data pipeline failed: %v", err)
		} else {
			log.Println("[CRON-JOB] Daily data pipeline dispatched successfully")
		}

	})
	if err != nil {
		log.Fatalf("CRON-ERROR] Failed to schedule daily data pipeline cron: %v", err)
	}

	// INTRADAY PIPELINE (CMD_SYNC_INTRADAY)
	// run every day at configured schedule (short interval for granularity)
	intradaySchedule := config.Env.IntradayDataPipelineCronSchedule
	_, err = c.AddFunc(intradaySchedule, func() {
		log.Println("[CRON-JOB] Starting intraday data pipeline (CMD_SYNC_INTRADAY)")

		if err := pipelineService.RunIntradayPipeline(); err != nil {
			log.Printf("[CRON-ERROR] Intraday data pipeline failed: %v", err)
		} else {
			log.Println("[CRON-JOB] Intraday data pipeline dispatched successfully")
		}
	})
	if err != nil {
		log.Fatalf("[CRON-ERROR] Failed to schedule intraday data pipeline: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] Scheduled data pipeline cron job")
}
