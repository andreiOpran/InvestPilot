package services

import (
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
)

type DataPipelineService interface {
	RunDailyPipeline() error
	RunIntradayPipeline() error
}

type dataPipelineService struct{}

func NewDataPipelineService() DataPipelineService {
	return &dataPipelineService{}
}

// RunDataPipeline publishes CMD_SYNC_DAILY and CMD_GENERATE sequentially to RabbitMQ
func (s *dataPipelineService) RunDailyPipeline() error {
	if clients.Publisher == nil {
		log.Println("[SERVICE-ERROR] Could not run daily data pipeline: RabbitMQ Publisher is not initialized")
		return nil
	}

	invConfig := config.Env.Investment

	// dispatch CMD_SYNC_DAILY
	syncPayload := models.SyncPayload{
		EquityTickers: invConfig.EquityTickers,
		BondTickers:   invConfig.BondTickers,
	}

	log.Println("[INFO] Dispatching CMD_SYNC_DAILY to publisher...")
	err := clients.Publisher.PublishCommand("CMD_SYNC_DAILY", syncPayload)
	if err != nil {
		log.Printf("[SERVICE-ERROR] Failed to publish CMD_SYNC_DAILY: %v", err)
		return err
	}

	// dispatch CMD_GENERATE
	genPayload := models.GeneratePayload{
		EquityTickers:      invConfig.EquityTickers,
		BondTickers:        invConfig.BondTickers,
		MacroAllocations:   invConfig.BaseEquityAllocation,
		HorizonMultipliers: invConfig.HorizonMultipliers,
		MaxEquityCap:       invConfig.MaxEquityCap,
		TopNEquities:       invConfig.TopNEquities,
		WeightThreshold:    invConfig.WeightThreshold,
		Verbose:            invConfig.Verbose,
	}

	log.Println("[INFO] Dispatching CMD_GENERATE to publisher...")
	err = clients.Publisher.PublishCommand("CMD_GENERATE", genPayload)
	if err != nil {
		log.Printf("[SERVICE-ERROR] Failed to publish CMD_GENERATE: %v", err)
		return err
	}

	return nil
}

// RunIntradayPipeline publishes CMD_SYNC_DAILY and CMD_GENERATE sequentially to RabbitMQ
func (s *dataPipelineService) RunIntradayPipeline() error {
	if clients.Publisher == nil {
		log.Println("[SERVICE-ERROR] Could not run daily intraday data pipeline: RabbitMQ Publisher is not initialized")
		return nil
	}

	invConfig := config.Env.Investment

	// dispatch CMD_SYNC_INTRADAY
	syncPayload := models.SyncPayload{
		EquityTickers: invConfig.EquityTickers,
		BondTickers:   invConfig.BondTickers,
	}

	log.Println("[INFO] Dispatching CMD_SYNC_INTRADAY to publisher...")
	return clients.Publisher.PublishCommand("CMD_SYNC_INTRADAY", syncPayload)
}
