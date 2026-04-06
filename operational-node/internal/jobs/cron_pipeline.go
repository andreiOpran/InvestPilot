package jobs

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
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
func StartDataPipelineJob() {
	c := cron.New()
	schedule := config.Env.DataPipelineCronSchedule

	// run every day at configured schedule (preferably after US market close)
	_, err := c.AddFunc(schedule, func() {
		log.Println("[CRON-JOB] Starting daily data pipeline (CMD_SYNC & CMD_GENERATE)")
		runDataPipeline()
	})

	if err != nil {
		log.Fatalf("CRON-ERROR] Failed to schedule data pipeline cron: %v", err)
	}

	c.Start()
	log.Println("[SYSTEM] Scheduled data pipeline cron job")
}

// runDataPipeline publishes CMD_SYNC and CMD_GENERATE sequentially to RabbitMQ
func runDataPipeline() {
	if clients.Publisher == nil {
		log.Println("[CRON-ERROR] Could not run daily data pipeline: RabbitMQ Publisher is not initialized")
		return
	}

	invConfig := config.Env.Investment

	// dispatch CMD_SYNC
	syncPayload := SyncPayload{
		EquityTickers: invConfig.EquityTickers,
		BondTickers:   invConfig.BondTickers,
	}

	log.Println("[CRON-JOB] Dispatching CMD_SYNC to publisher...")
	err := clients.Publisher.PublishCommand("CMD_SYNC", syncPayload)
	if err != nil {
		log.Printf("[CRON-ERROR] Failed to publish CMD_SYNC: %v", err)
		return
	}

	// Dispatch CMD_GENERATE
	genPayload := GeneratePayload{
		EquityTickers:      invConfig.EquityTickers,
		BondTickers:        invConfig.BondTickers,
		MacroAllocations:   invConfig.BaseEquityAllocation,
		HorizonMultipliers: invConfig.HorizonMultipliers,
		MaxEquityCap:       invConfig.MaxEquityCap,
		TopNEquities:       invConfig.TopNEquities,
		WeightThreshold:    invConfig.WeightThreshold,
		Verbose:            invConfig.Verbose,
	}

	log.Println("[CRON-JOB] Dispatching CMD_GENERATE to publisher...")
	err = clients.Publisher.PublishCommand("CMD_GENERATE", genPayload)
	if err != nil {
		log.Printf("[CRON-ERROR] Failed to publish CMD_GENERATE: %v", err)
		return
	}
}
