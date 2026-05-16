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
		// USD covered the full sell - copy ETF holdings unchanged
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
	since, interval, isIntraday := parseHistoryTimeRange(timeRange, now)

	// fetch rounds and extract unique tickers
	rounds, err := s.portfolioRepo.GetHistoricalRounds(userID, since)
	if err != nil {
		return models.PortfolioHistoryResponse{}, err
	}

	tickers, tickerMap := extractTickers(rounds)

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

	timestampSet := collectPricingTimestamps(pricing, interval)
	effectiveSince := since // updated if intraday fallback fires

	// for intraday ranges on weekends/holidays, the 24h window may fall entirely
	// outside market hours; fall back to the last available trading session
	if isIntraday && len(timestampSet) == 0 && len(tickers) > 0 {
		rounds, pricing, tickers, effectiveSince, timestampSet, err = s.applyIntradayFallback(userID, since, tickers, tickerMap, pricing, rounds, interval, now)
		if err != nil {
			return models.PortfolioHistoryResponse{}, err
		}
	}

	// if USD-only portfolio, generate anchored timestamps to real market data
	// so the chart aligns with actual trading hours instead of synthetic intervals
	if len(tickers) == 0 {
		timestampSet, err = s.buildUSDTimestampSet(since, isIntraday, interval, now)
		if err != nil {
			return models.PortfolioHistoryResponse{}, err
		}
	}

	allTimestamps := sortedTimestamps(timestampSet)

	// pre-seed with pre-window prices so seed round holdings are valued from timestamp zero
	lastKnownPrices, err := s.portfolioRepo.GetPricesBeforeWindow(tickers, effectiveSince, isIntraday)
	if err != nil {
		return models.PortfolioHistoryResponse{}, err
	}

	return models.PortfolioHistoryResponse{
		Range: timeRange,
		Data:  buildDataPoints(allTimestamps, tickers, pricing, rounds, transactions, lastKnownPrices),
	}, nil
}

func parseHistoryTimeRange(timeRange string, now time.Time) (since time.Time, interval time.Duration, isIntraday bool) {
	switch timeRange {
	case "1D":
		return now.AddDate(0, 0, -1), 15 * time.Minute, true
	case "1W":
		return now.AddDate(0, 0, -7), 1 * time.Hour, true
	case "1M":
		return now.AddDate(0, -1, 0), 24 * time.Hour, false
	case "6M":
		return now.AddDate(0, -6, 0), 24 * time.Hour, false
	case "1Y":
		return now.AddDate(-1, 0, 0), 24 * time.Hour, false
	case "YTD":
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()), 24 * time.Hour, false
	case "5Y":
		return now.AddDate(-5, 0, 0), 5 * 24 * time.Hour, false
	default:
		// default to 1M
		return now.AddDate(0, -1, 0), 24 * time.Hour, false
	}
}

func extractTickers(rounds []models.InvestmentRound) ([]string, map[string]bool) {
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
	tickers := make([]string, 0, len(tickerMap))
	for t := range tickerMap {
		tickers = append(tickers, t)
	}
	return tickers, tickerMap
}

// bucketTimestamp rounds ts up to the next interval boundary (e.g. 14:15 -> 15:00)
func bucketTimestamp(ts time.Time, interval time.Duration) time.Time {
	if interval > 0 && ts.Truncate(interval) != ts {
		// round up to next boundary (14:15 -> 15:00)
		return ts.Add(interval).Truncate(interval)
	}
	return ts
}

// collectPricingTimestamps extracts and deduplicates all unique timestamps across all tickers
// Assets may have prices recorded at slightly different timestamps, so we build a unified
// timeline by collecting unique bucketed timestamps (sorted asc by the caller)
func collectPricingTimestamps(pricing map[string][]models.AssetPricePoint, interval time.Duration) map[time.Time]struct{} {
	set := make(map[time.Time]struct{})
	for _, prices := range pricing {
		for _, p := range prices {
			// aggregate multiple DB timestamps into defined interval bucket
			set[bucketTimestamp(p.Timestamp, interval)] = struct{}{}
		}
	}
	return set
}

// applyIntradayFallback handles the case where the current 24h/7d window falls entirely
// outside market hours (weekend/holiday). It looks back up to 4 days to find the last
// trading session and rebuilds rounds, pricing, tickers, and the timestamp set for that session
func (s *portfolioService) applyIntradayFallback(
	userID uint,
	since time.Time,
	tickers []string,
	tickerMap map[string]bool,
	pricing map[string][]models.AssetPricePoint,
	rounds []models.InvestmentRound,
	interval time.Duration,
	now time.Time,
) ([]models.InvestmentRound, map[string][]models.AssetPricePoint, []string, time.Time, map[time.Time]struct{}, error) {
	var err error
	pricing, err = s.portfolioRepo.GetPricingData(tickers, since.AddDate(0, 0, -4), true)
	if err != nil {
		return nil, nil, nil, since, nil, err
	}

	var latestDay time.Time
	for _, prices := range pricing {
		for _, p := range prices {
			if day := p.Timestamp.Truncate(24 * time.Hour); day.After(latestDay) {
				latestDay = day
			}
		}
	}

	timestampSet := make(map[time.Time]struct{})
	if latestDay.IsZero() {
		return rounds, pricing, tickers, since, timestampSet, nil
	}

	for _, prices := range pricing {
		for _, p := range prices {
			if p.Timestamp.Truncate(24 * time.Hour).Equal(latestDay) {
				ts := bucketTimestamp(p.Timestamp, interval)
				if !ts.After(now) {
					timestampSet[ts] = struct{}{}
				}
			}
		}
	}

	// update effective window to the start of the last trading session
	effectiveSince := latestDay

	// re-fetch rounds so earlier rounds active during this session are included
	rounds, err = s.portfolioRepo.GetHistoricalRounds(userID, effectiveSince)
	if err != nil {
		return nil, nil, nil, effectiveSince, nil, err
	}

	// find tickers in the extended rounds that weren't in the original set
	var deltaTickers []string
	for _, r := range rounds {
		for _, h := range r.Holdings {
			if h.Ticker != "USD" && !tickerMap[h.Ticker] {
				tickerMap[h.Ticker] = true
				deltaTickers = append(deltaTickers, h.Ticker)
			}
		}
	}

	// only fetch pricing for the genuinely new tickers
	if len(deltaTickers) > 0 {
		deltaPricing, err := s.portfolioRepo.GetPricingData(deltaTickers, effectiveSince, true)
		if err != nil {
			return nil, nil, nil, effectiveSince, nil, err
		}
		for ticker, prices := range deltaPricing {
			pricing[ticker] = prices
		}
		tickers = append(tickers, deltaTickers...)
	}

	return rounds, pricing, tickers, effectiveSince, timestampSet, nil
}

// buildUSDTimestampSet returns market-anchored timestamps for a USD-only portfolio,
// falling back to the last trading session when the intraday window is empty.
func (s *portfolioService) buildUSDTimestampSet(
	since time.Time,
	isIntraday bool,
	interval time.Duration,
	now time.Time,
) (map[time.Time]struct{}, error) {
	marketTimestamps, err := s.portfolioRepo.GetMarketTimestamps(since, isIntraday)
	if err != nil {
		return nil, err
	}

	// if intraday window is empty (weekend/holiday), fall back to last trading session
	if isIntraday && len(marketTimestamps) == 0 {
		marketTimestamps, err = s.portfolioRepo.GetMarketTimestamps(since.AddDate(0, 0, -4), isIntraday)
		if err != nil {
			return nil, err
		}
		var latestDay time.Time
		for _, ts := range marketTimestamps {
			if day := ts.Truncate(24 * time.Hour); day.After(latestDay) {
				latestDay = day
			}
		}
		filtered := marketTimestamps[:0]
		for _, ts := range marketTimestamps {
			if ts.Truncate(24 * time.Hour).Equal(latestDay) {
				filtered = append(filtered, ts)
			}
		}
		marketTimestamps = filtered
	}

	set := make(map[time.Time]struct{})
	for _, ts := range marketTimestamps {
		bucketed := bucketTimestamp(ts, interval)
		if interval > 0 && bucketed.After(now) {
			continue
		}
		set[bucketed] = struct{}{}
	}
	return set, nil
}

func sortedTimestamps(set map[time.Time]struct{}) []time.Time {
	result := make([]time.Time, 0, len(set))
	for t := range set {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Before(result[j])
	})
	return result
}

// buildDataPoints walks the sorted timeline and computes portfolio value, net contributions,
// and return percentage at each timestamp. lastKnownPrices is mutated in-place as a price board.
func buildDataPoints(
	allTimestamps []time.Time,
	tickers []string,
	pricing map[string][]models.AssetPricePoint,
	rounds []models.InvestmentRound,
	transactions []models.Transaction,
	lastKnownPrices map[string]float64,
) []models.PortfolioHistoryPoint {
	// state trackers for algorithmic traversal
	priceIndices := make(map[string]int)
	// store values from the first day of the interval
	// to have as a comparison for return percentages
	var baselinePortfolioValue, baselineNetContributions float64
	dataPoints := make([]models.PortfolioHistoryPoint, 0, len(allTimestamps))

	for i, t := range allTimestamps {
		// update current price board for this specific timestamp
		// advance a pointer for each ticker, using last known
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
		for j := range rounds {
			if rounds[j].CreatedAt.Before(t) || rounds[j].CreatedAt.Equal(t) {
				activeRound = &rounds[j]
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
				} else if price, ok := lastKnownPrices[h.Ticker]; ok {
					portfolioValue += h.Shares * price
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

	return dataPoints
}
