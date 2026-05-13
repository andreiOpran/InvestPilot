package models

import "time"

// PortfolioHistoryPoint holds a users portfolio value and contributions at a given timestamp
type PortfolioHistoryPoint struct {
	Timestamp        time.Time `json:"timestamp"`
	PortfolioValue   float64   `json:"portfolio_value"`
	ReturnPercentage float64   `json:"return_percentage"`
	NetContributions float64   `json:"net_contributions"`
}

// AssetPricePoint holds a ticker together with its price at a given timestamp
type AssetPricePoint struct {
	Ticker    string
	Timestamp time.Time
	Price     float64
}

type PortfolioHistoryResponse struct {
	Range string                  `json:"range"`
	Data  []PortfolioHistoryPoint `json:"data"`
}

type PaginatedTransactionsResponse struct {
	Data       []UnifiedTransaction `json:"data"`
	TotalCount int64                `json:"total_count"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
}

type HoldingResponse struct {
	Ticker       string  `json:"ticker"`
	Shares       float64 `json:"shares"`
	CurrentPrice float64 `json:"current_price"`
	CurrentValue float64 `json:"current_value"`
	TargetWeight float64 `json:"target_weight"`
}

type PortfolioSummaryResponse struct {
	LiveTotalValue    float64           `json:"live_total_value"`
	NetContributions  float64           `json:"net_contributions"`
	AllTimeProfitLoss float64           `json:"all_time_profit_loss"`
	RoundID           uint              `json:"round_id"`
	Holdings          []HoldingResponse `json:"holdings"`
}
