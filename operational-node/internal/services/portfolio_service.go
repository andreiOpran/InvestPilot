package services

import (
	"sort"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type PortfolioService interface {
	Invest(userID uint, amount float64) error
	GetPortfolioHistory(userID uint, timeRange string) (models.PortfolioHistoryResponse, error)
}

type portfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
}

func NewPortfolioService(portfolioRepo repositories.PortfolioRepository, userRepo repositories.UserRepository) PortfolioService {
	return &portfolioService{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
	}
}

func (s *portfolioService) Invest(userID uint, amount float64) error {
	// domain check: fetch wallet and validate balance
	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return err
	}
	if wallet.Balance < amount {
		return ErrInsufficientBalance
	}

	// domain action: deduct money locally
	wallet.Balance -= amount

	// domain action: create transaction
	txRecord := &models.Transaction{
		UserID: userID,
		Type:   "invest",
		Amount: amount,
	}

	oldRound, err := s.portfolioRepo.GetRoundWithHoldingsByStatus(userID, true)
	if err != nil {
		return err
	}

	var newTotalValue float64
	var newHoldings []models.Holding
	usdFound := false

	// domain logic: adjust holdings
	if oldRound != nil {
		oldRound.IsActive = false // retire
		newTotalValue = oldRound.TotalValue + amount

		for _, h := range oldRound.Holdings {
			// copy old asset tracking
			newH := models.Holding{
				UserID:          userID,
				Ticker:          h.Ticker,
				Weight:          h.Weight,
				Shares:          h.Shares,
				PurchasePrice:   h.PurchasePrice,
				AllocatedAmount: h.AllocatedAmount,
			}

			// add funds to USD bucket if it exists
			if h.Ticker == "USD" {
				newH.Shares += amount
				newH.AllocatedAmount += amount
				usdFound = true
			}
			newHoldings = append(newHoldings, newH)
		}
	} else {
		newTotalValue = amount
	}

	// if no USD position existed previously, initialize it here
	if !usdFound {
		newHoldings = append(newHoldings, models.Holding{
			UserID:          userID,
			Ticker:          "USD",
			Weight:          amount / newTotalValue, // roughly 1.0
			Shares:          amount,
			PurchasePrice:   1.0,
			AllocatedAmount: amount,
		})
	}

	// build final new InvestmentRound object
	newRound := &models.InvestmentRound{
		UserID:     userID,
		TotalValue: newTotalValue,
		IsActive:   true,
		Holdings:   newHoldings, // Repo handles writing all these objects natively via GORM cascade
	}

	// give prepared domain models to repo to execute as one transaction
	return s.portfolioRepo.ExecuteInvestTransaction(wallet, txRecord, oldRound, newRound)
}

func (s *portfolioService) GetPortfolioHistory(userID uint, timeRange string) (models.PortfolioHistoryResponse, error) {
	now := time.Now()
	var since time.Time
	var interval time.Duration // if interval is left 0, it is default to 1 day
	isIntraday := false

	switch timeRange {
	case "1D":
		since = now.AddDate(0, 0, -1)
		isIntraday = true
		interval = 15 * time.Minute
	case "1W":
		since = now.AddDate(0, 0, -7)
		isIntraday = true
		interval = 1 * time.Hour
	case "1M":
		since = now.AddDate(0, -1, 0)
	case "6M":
		since = now.AddDate(0, -6, 0)
	case "1Y":
		since = now.AddDate(-1, 0, 0)
	case "YTD":
		since = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	case "5Y":
		since = now.AddDate(-5, 0, 0)
	default:
		// default to 1M
		since = now.AddDate(0, -1, 0)
	}

	// fetch rounds and extract unique tickers
	rounds, err := s.portfolioRepo.GetHistoricalRounds(userID, since)
	if err != nil {
		return models.PortfolioHistoryResponse{}, err
	}

	// extract unique tickers across historical rounds,
	// so we only fetch pricing for what was actually held
	tickerMap := make(map[string]bool)
	for _, r := range rounds {
		for _, h := range r.Holdings {
			if h.Ticker != "USD" {
				tickerMap[h.Ticker] = true
			}
		}
	}

	var tickers []string
	for t := range tickerMap {
		tickers = append(tickers, t)
	}

	// fetch pricing data
	pricing, err := s.portfolioRepo.GetPricingData(tickers, since, isIntraday)
	if err != nil {
		return models.PortfolioHistoryResponse{}, err
	}

	// fetch funding data (used in contributions line on the chart)
	fundings, err := s.portfolioRepo.GetHistoricalFundings(userID)
	if err != nil {
		return models.PortfolioHistoryResponse{}, err
	}

	// extract and sort all unique timestamps across all tickers
	// assets might have prices recorded at slightly different timestamp,
	// so we extract a timeline by getting unique timestamps (sorted asc)
	timestampSet := make(map[time.Time]struct{})
	for _, prices := range pricing {
		for _, p := range prices {
			ts := p.Timestamp

			// aggregate multiple DB timestamps into defined interval bucket
			if interval > 0 {
				// round up to next boundary (14:15 -> 15:00)
				if ts.Truncate(interval) != ts {
					ts = ts.Add(interval).Truncate(interval)
				}
			}

			timestampSet[ts] = struct{}{}
		}
	}
	var allTimestamps []time.Time
	for t := range timestampSet {
		allTimestamps = append(allTimestamps, t)
	}
	sort.Slice(allTimestamps, func(i, j int) bool {
		return allTimestamps[i].Before(allTimestamps[j])
	})

	// build time series
	var dataPoints []models.PortfolioHistoryPoint
	// state trackers for algorithmic traversal
	lastKnownPrices := make(map[string]float64)
	priceIndices := make(map[string]int)

	for _, t := range allTimestamps {
		// update current price board for this specific timestamp
		// advance a pointer for each ticker , using last known
		// price to handle mismatched pricing timestamps
		for _, ticker := range tickers {
			prices := pricing[ticker]
			idx := priceIndices[ticker]
			for idx < len(prices) && !prices[idx].Timestamp.After(t) {
				lastKnownPrices[ticker] = prices[idx].Price
				idx++
			}
			// save pointer for next iteration
			priceIndices[ticker] = idx
		}

		// determine active portfolio composition by finding the
		// InvestmentRound that was active exactly at timestamp 't'
		var activeRound *models.InvestmentRound
		for i := range rounds {
			if rounds[i].CreatedAt.Before(t) || rounds[i].CreatedAt.Equal(t) {
				activeRound = &rounds[i]
			} else {
				// because rounds are fetched sorted by CreatedAt, we can shortcircuit
				break
			}
		}

		// calculate total portfolio value at timestamp 't'
		portfolioValue := 0.0
		if activeRound != nil {
			for _, h := range activeRound.Holdings {
				if h.Ticker == "USD" {
					// for USD, shares represent cash value
					portfolioValue += h.Shares
				} else {
					if price, ok := lastKnownPrices[h.Ticker]; ok {
						portfolioValue += h.Shares * price
					}
				}
			}
		}

		// calculate total net contributions up to 't'
		netContributions := 0.0
		for _, f := range fundings {
			if f.CreatedAt.Before(t) || f.CreatedAt.Equal(t) {
				if f.Type == "DEPOSIT" {
					netContributions += f.Amount
				} else if f.Type == "WITHDRAWAL" {
					netContributions -= f.Amount
				}
			}
		}

		dataPoints = append(dataPoints, models.PortfolioHistoryPoint{
			Timestamp:        t,
			PortfolioValue:   portfolioValue,
			NetContributions: netContributions,
		})
	}

	return models.PortfolioHistoryResponse{
		Range: timeRange,
		Data:  dataPoints,
	}, nil
}
