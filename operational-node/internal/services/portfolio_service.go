package services

import (
	"sort"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type PortfolioService interface {
	Invest(userID uint, amount float64) error
	Sell(userID uint, amount float64) error
	GetPortfolioSummary(userID uint) (*models.PortfolioSummaryResponse, error)
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
		Type:   "INVEST",
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

func (s *portfolioService) Sell(userID uint, amount float64) error {
	activeRound, err := s.portfolioRepo.GetRoundWithHoldingsByStatus(userID, true)
	if err != nil {
		return err
	}
	if activeRound == nil {
		return ErrNoActivePortfolio
	}

	// resolve current prices for all non-USD holdings
	var tickers []string
	for _, h := range activeRound.Holdings {
		if h.Ticker != "USD" {
			tickers = append(tickers, h.Ticker)
		}
	}
	latestPrices, err := s.portfolioRepo.GetLatestPrices(tickers)
	if err != nil {
		return err
	}

	// calculate live total value
	liveTotalValue := 0.0
	for _, h := range activeRound.Holdings {
		price := 1.0
		if h.Ticker != "USD" {
			price = latestPrices[h.Ticker]
			if price == 0 {
				price = h.PurchasePrice
			}
		}
		liveTotalValue += h.Shares * price
	}

	if amount > liveTotalValue {
		return ErrSellExceedsPortfolioValue
	}

	remainingToSell := amount
	var newHoldings []models.Holding

	// pass 1: drain USD first (1 share = $1, no price lookup needed)
	for _, h := range activeRound.Holdings {
		if h.Ticker != "USD" {
			continue
		}
		consumed := h.Shares
		if remainingToSell < consumed {
			consumed = remainingToSell
		}
		remainingToSell -= consumed
		newUSDShares := h.Shares - consumed
		if newUSDShares >= 0.01 {
			newHoldings = append(newHoldings, models.Holding{
				UserID:          userID,
				Ticker:          "USD",
				Weight:          h.Weight,
				Shares:          newUSDShares,
				PurchasePrice:   h.PurchasePrice,
				AllocatedAmount: h.AllocatedAmount - consumed,
			})
		}
		break // only one USD holding per round
	}

	// pass 2: if USD was not enough, sell proportionally from ETF holdings
	if remainingToSell > 0.005 {
		totalETFValue := 0.0
		for _, h := range activeRound.Holdings {
			if h.Ticker == "USD" {
				continue
			}
			price := latestPrices[h.Ticker]
			if price == 0 {
				price = h.PurchasePrice
			}
			totalETFValue += h.Shares * price
		}
		fraction := remainingToSell / totalETFValue
		for _, h := range activeRound.Holdings {
			if h.Ticker == "USD" {
				continue // already handled
			}
			price := latestPrices[h.Ticker]
			if price == 0 {
				price = h.PurchasePrice
			}
			newShares := h.Shares * (1 - fraction)
			if newShares*price < 0.01 {
				continue // drop dust positions
			}
			newHoldings = append(newHoldings, models.Holding{
				UserID:          userID,
				Ticker:          h.Ticker,
				Weight:          h.Weight,
				Shares:          newShares,
				PurchasePrice:   h.PurchasePrice,
				AllocatedAmount: h.AllocatedAmount * (1 - fraction),
			})
		}
	} else {
		// USD covered the full sell — copy ETF holdings unchanged
		for _, h := range activeRound.Holdings {
			if h.Ticker == "USD" {
				continue
			}
			newHoldings = append(newHoldings, models.Holding{
				UserID:          userID,
				Ticker:          h.Ticker,
				Weight:          h.Weight,
				Shares:          h.Shares,
				PurchasePrice:   h.PurchasePrice,
				AllocatedAmount: h.AllocatedAmount,
			})
		}
	}

	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return err
	}
	wallet.Balance += amount

	activeRound.IsActive = false

	txRecord := &models.Transaction{
		UserID: userID,
		Type:   "SELL",
		Amount: amount,
	}

	// newRound is nil when the portfolio is fully liquidated
	var newRound *models.InvestmentRound
	remainingValue := liveTotalValue - amount
	if len(newHoldings) > 0 && remainingValue > 0.01 {
		newRound = &models.InvestmentRound{
			UserID:     userID,
			TotalValue: remainingValue,
			IsActive:   true,
			Holdings:   newHoldings,
		}
	}

	return s.portfolioRepo.ExecuteSellTransaction(wallet, txRecord, activeRound, newRound)
}

func (s *portfolioService) GetPortfolioSummary(userID uint) (*models.PortfolioSummaryResponse, error) {
	// get active round
	activeRound, err := s.portfolioRepo.GetRoundWithHoldingsByStatus(userID, true)
	if err != nil {
		return nil, err
	}

	// get transactions for net contribution (INVEST minus SELL)
	investTxs, err := s.portfolioRepo.GetInvestTransactions(userID)
	if err != nil {
		return nil, err
	}

	netContributions := 0.0
	for _, tx := range investTxs {
		if tx.Type == "INVEST" {
			netContributions += tx.Amount
		} else if tx.Type == "SELL" {
			netContributions -= tx.Amount
		}
	}

	// if no active round exists yet (new user)
	if activeRound == nil {
		return &models.PortfolioSummaryResponse{
			LiveTotalValue:    0.0,
			NetContributions:  netContributions,
			AllTimeProfitLoss: 0.0,
			RoundID:           0,
			Holdings:          []models.HoldingResponse{},
		}, nil
	}

	// extract unique tickers from the active round (skip USD)
	var tickers []string
	for _, h := range activeRound.Holdings {
		if h.Ticker != "USD" {
			tickers = append(tickers, h.Ticker)
		}
	}

	// fetch latest prices for the tickers
	latestPrices, err := s.portfolioRepo.GetLatestPrices(tickers)
	if err != nil {
		return nil, err
	}

	var holdingResponses []models.HoldingResponse
	liveTotalValue := 0.0

	// generate holding responses and calculate live baseline
	for _, h := range activeRound.Holdings {
		currentPrice := 1.0 // default for USD
		if h.Ticker != "USD" {
			currentPrice = latestPrices[h.Ticker]
			// fallback to purchase price if intraday data is missing for some reason
			if currentPrice == 0 {
				currentPrice = h.PurchasePrice
			}
		}

		currentValue := h.Shares * currentPrice
		liveTotalValue += currentValue

		holdingResponses = append(holdingResponses, models.HoldingResponse{
			Ticker:       h.Ticker,
			Shares:       h.Shares,
			CurrentPrice: currentPrice,
			CurrentValue: currentValue,
			TargetWeight: h.Weight,
		})
	}

	allTimeProfitLoss := liveTotalValue - netContributions

	return &models.PortfolioSummaryResponse{
		LiveTotalValue:    liveTotalValue,
		NetContributions:  netContributions,
		AllTimeProfitLoss: allTimeProfitLoss,
		RoundID:           activeRound.ID,
		Holdings:          holdingResponses,
	}, nil
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

	// fetch transactions data (used in contributions line on the chart, INVEST minus SELL)
	transactions, err := s.portfolioRepo.GetInvestTransactions(userID)
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

	// store values from the first day of the interval
	// to have as a comparison for return percentages
	var baselinePortfolioValue float64
	var baselineNetContributions float64

	for i, t := range allTimestamps {
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
		for _, tx := range transactions {
			if tx.CreatedAt.Before(t) || tx.CreatedAt.Equal(t) {
				if tx.Type == "INVEST" {
					netContributions += tx.Amount
				} else if tx.Type == "SELL" {
					netContributions -= tx.Amount
				}
			}
		}

		// calculate percentage for profit/loss
		if i == 0 {
			// capture initial values at start of the timeframe
			baselinePortfolioValue = portfolioValue
			baselineNetContributions = netContributions
		}

		// net contributions made strictly during this timeframe
		timeframeNetContributions := netContributions - baselineNetContributions

		// capital at risk is the portfolio size we started + any new cash we added
		capitalAtRisk := baselinePortfolioValue + timeframeNetContributions

		returnPercentage := 0.0
		// ensure its not the first point (which must be 0) and prevent div zero
		if i > 0 && capitalAtRisk > 0 {
			// profit generated exclusively in this timeframe
			timeframeProfit := portfolioValue - capitalAtRisk
			returnPercentage = (timeframeProfit / capitalAtRisk) * 100
		}

		dataPoints = append(dataPoints, models.PortfolioHistoryPoint{
			Timestamp:        t,
			PortfolioValue:   portfolioValue,
			ReturnPercentage: returnPercentage,
			NetContributions: netContributions,
		})
	}

	return models.PortfolioHistoryResponse{
		Range: timeRange,
		Data:  dataPoints,
	}, nil
}
