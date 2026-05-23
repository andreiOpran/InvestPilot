package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/metrics"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/google/uuid"
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
	// staleness check
	maxDate, err := s.rebalanceRepo.GetLatestMarketDataDate()
	if err != nil {
		return err
	}

	// trading paused on weekends, so we add 2 days when checking
	daysOld := int(time.Since(maxDate).Hours() / 24)
	maxDaysStaleness := config.Env.Investment.PriceStalenessDays
	if daysOld > maxDaysStaleness+2 {
		metrics.RebalanceStaleDataAborts.Inc()
		log.Printf("[REBALANCE ABORTED] %v", ErrRebalancePausedStaleMarketData)
		return ErrRebalancePausedStaleMarketData
	}

	// load and map latest model portfolios
	portfolioRows, err := s.rebalanceRepo.GetLatestModelPortfolios()
	if err != nil {
		return err
	}
	latestModels := make(map[string]map[string]float64)
	for _, m := range portfolioRows {
		var weights map[string]float64
		if err := json.Unmarshal([]byte(m.Weights), &weights); err != nil {
			return err
		}
		latestModels[m.BucketKey] = weights
	}

	// load and map latest prices, while injecting USD
	priceRows, err := s.rebalanceRepo.GetLatestPrices()
	if err != nil {
		return err
	}
	latestPrices := make(map[string]float64)
	for _, res := range priceRows {
		latestPrices[res.Ticker] = res.ClosePrice
	}
	latestPrices["USD"] = 1.0 // fiat money, by convention price is 1.0

	// get maximum InvestmentRound ID currently in the DB
	// to be used as a ceiling for deactivating old rounds
	maxID, err := s.rebalanceRepo.GetMaxRoundID()
	if err != nil {
		return err
	}

	// process users in batches
	var lastID uint = 0

	for {
		activeRounds, err := s.rebalanceRepo.GetInvestmentRoundsBatchByStatus(
			true, // isActive
			lastID,
			maxID,
			config.Env.RebalanceBatchSize,
		)
		if err != nil {
			return err
		}

		if len(activeRounds) == 0 {
			break // no more users to process
		}

		// prepare batch request
		var batchRequest models.RebalanceBatchRequest
		batchRequest.Threshold = config.Env.Investment.RebalanceDeltaThreshold
		batchRequest.CashFirst = config.Env.Investment.CashFirstEnabled

		// map to link decisional node anonymous request_id back to actual node data
		roundMap := make(map[string]models.InvestmentRound)

		// prepare payload for decisional node
		for _, round := range activeRounds {

			// derive bucket key logic, with casting from int to string for the horizon
			horizonStr := "long"
			if round.User.InvestmentHorizon <= config.Env.Investment.HorizonShortMax {
				horizonStr = "short"
			} else if round.User.InvestmentHorizon <= config.Env.Investment.HorizonMediumMax {
				horizonStr = "medium"
			}
			bucketKey := fmt.Sprintf("risk_%d_horizon_%s", round.User.RiskTolerance, horizonStr)

			targetWeights := latestModels[bucketKey]
			if targetWeights == nil {
				continue // no model exists for this profile
			}

			currentAllocation := make(map[string]float64)
			for _, h := range round.Holdings {
				currentAllocation[h.Ticker] = h.AllocatedAmount / round.TotalValue
			}

			reqID := uuid.New().String()
			roundMap[reqID] = round

			batchRequest.Users = append(batchRequest.Users, models.RebalanceUserRequest{
				RequestID:         reqID,
				CurrentAllocation: currentAllocation,
				TargetWeights:     targetWeights,
			})
		}

		// RPC Call to decisional node (skip if this batch had active rounds but none matched a model)
		if len(batchRequest.Users) > 0 {
			responseBytes, err := clients.Publisher.PublishRPC("CMD_REBALANCE_BATCH", batchRequest)
			if err != nil {
				return err
			}

			var batchResponse models.RebalanceBatchResponse
			if err := json.Unmarshal(responseBytes, &batchResponse); err != nil {
				return err
			}

			// compute new share allocations for this batch
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

			// atomic DB transaction swap for this chunk
			err = s.rebalanceRepo.ExecuteBatchRebalanceTransaction(newRounds, oldRoundIDs)
			if err != nil {
				return err
			}

			log.Printf("[REBALANCE] Successfully processed batch of %d users", len(newRounds))
		}

		lastID = activeRounds[len(activeRounds)-1].ID
	}

	return nil
}
