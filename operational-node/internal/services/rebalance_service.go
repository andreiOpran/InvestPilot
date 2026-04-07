package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type RebalanceService interface {
	RunMonthlyRebalance() error
}

type rebalanceService struct {
	rebalanceRepo repositories.RebalanceRepository
	userRepo      repositories.UserRepository
}

func NewRebalanceService(rr repositories.RebalanceRepository, ur repositories.UserRepository) RebalanceService {
	return &rebalanceService{rebalanceRepo: rr, userRepo: ur}
}

func (s *rebalanceService) RunMonthlyRebalance() error {
	stalenessDays := config.Env.Investment.PriceStalenessDays

	// staleness check
	if err := s.rebalanceRepo.CheckPriceStaleness(stalenessDays); err != nil {
		log.Printf("[REBALANCE ABORTED] %v", err)
		if errors.Is(err, repositories.ErrMarketDataStale) {
			return ErrRebalancePausedStaleMarketData
		}
		return err
	}

	// load latest mopdel portfolios and active rouinds
	latestModels, err := s.rebalanceRepo.GetLatestModelPortfolios()
	if err != nil {
		return err
	}

	activeRounds, err := s.rebalanceRepo.GetActiveInvestmentRounds()
	if err != nil {
		return err
	}

	if len(activeRounds) == 0 {
		return nil // nothing to do
	}

	// prepare batch request
	var batchRequest models.RebalanceBatchRequest
	batchRequest.Threshold = config.Env.Investment.RebalanceDeltaThreshold
	batchRequest.CashFirst = config.Env.Investment.CashFirstEnabled

	// map to link decisional node anonymous request_id back to actual node data
	roundMap := make(map[string]models.InvestmentRound)

	for _, round := range activeRounds {
		user, _ := s.userRepo.FindByID(round.UserID)

		// derive bucket key logic, with casting from int to string for the horizon
		horizonStr := "long"
		if user.InvestmentHorizon <= config.Env.Investment.HorizonShortMax {
			horizonStr = "short"
		} else if user.InvestmentHorizon <= config.Env.Investment.HorizonMediumMax {
			horizonStr = "medium"
		}
		bucketKey := fmt.Sprintf("risk_%d_horizon_%s", user.RiskTolerance, horizonStr)

		targetWeights := latestModels[bucketKey]
		if targetWeights == nil {
			continue // no model exists for this profile
		}

		currentAllocation := make(map[string]float64)
		for _, h := range round.Holdings {
			currentAllocation[h.Ticker] = h.AllocatedAmount / round.TotalValue
		}

		reqID := fmt.Sprintf("%d", round.UserID)
		roundMap[reqID] = round

		batchRequest.Users = append(batchRequest.Users, models.RebalanceUserRequest{
			RequestID:         reqID,
			CurrentAllocation: currentAllocation,
			TargetWeights:     targetWeights,
		})
	}

	// RPC Call to decisional node
	responseBytes, err := clients.Publisher.PublishRPC("CMD_REBALANCE_BATCH", batchRequest)
	if err != nil {
		return err
	}

	var batchResponse models.RebalanceBatchResponse
	if err := json.Unmarshal(responseBytes, &batchResponse); err != nil {
		return err
	}

	// read latest prices for Share math
	latestPrices, err := s.rebalanceRepo.GetLatestPrices()
	if err != nil {
		return err
	}

	// compute new share allocations
	var newRounds []models.InvestmentRound
	var oldRoundIDs []uint

	for _, result := range batchResponse.Results {
		oldRound := roundMap[result.RequestID]
		oldRoundIDs = append(oldRoundIDs, oldRound.ID)

		var newHoldings []models.Holding
		for ticker, targetWeight := range result.AdjustedTargets {
			allocatedAmount := targetWeight * oldRound.TotalValue
			price := latestPrices[ticker]
			shares := allocatedAmount / price

			newHoldings = append(newHoldings, models.Holding{
				UserID:          oldRound.UserID,
				Ticker:          ticker,
				Weight:          targetWeight,
				Shares:          shares,
				PurchasePrice:   price,
				AllocatedAmount: allocatedAmount,
			})
		}

		newRounds = append(newRounds, models.InvestmentRound{
			UserID:     oldRound.UserID,
			TotalValue: oldRound.TotalValue,
			IsActive:   true,
			Holdings:   newHoldings,
		})
	}

	// atomic DB transaction swap
	return s.rebalanceRepo.ExecuteBatchRebalanceTransaction(newRounds, oldRoundIDs)
}
