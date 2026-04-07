package jobs

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/services"
	"github.com/robfig/cron/v3"
)

// SyncPayload represents the JSON payload to send for CMD_SYNC
type SyncPayload struct {
	EquityTickers []string `json:"equity_tickers"`
	BondTickers   []string `json:"bond_tickers"`
}

// GeneratePayload represents the JSON payload to send for CMD_GENERATE
type GeneratePayload struct {
	EquityTickers      []string           `json:"equity_tickers"`
	BondTickers        []string           `json:"bond_tickers"`
	MacroAllocations   map[int]float64    `json:"macro_allocations"`
	HorizonMultipliers map[string]float64 `json:"horizon_multipliers"`
	MaxEquityCap       float64            `json:"max_equity_cap"`
	TopNEquities       int                `json:"top_n_equities"`
	WeightThreshold    float64            `json:"weight_threshold"`
	Verbose            bool               `json:"verbose"`
}

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
