package repositories

import "errors"

var (
	ErrMarketDataStale = errors.New("market data is stale: prices are older than allowed max days")
)
