package models

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
