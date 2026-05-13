package models

// RebalanceBatchRequest is sent to decisional node to request a batch rebalance
type RebalanceBatchRequest struct {
	Threshold float64                `json:"threshold"` // e.g. 0.02
	CashFirst bool                   `json:"cash_first"`
	Users     []RebalanceUserRequest `json:"users"`
}

// RebalanceUserRequest is a single user request for rebalancing used in batching
type RebalanceUserRequest struct {
	RequestID         string             `json:"request_id"` // UUID to map back to the UserID
	CurrentAllocation map[string]float64 `json:"current_allocation"`
	TargetWeights     map[string]float64 `json:"target_weights"`
}

// RebalanceBatchResponse is the response of the decisional-node for a batch of rebalancing
type RebalanceBatchResponse struct {
	Results []RebalanceUserResponse `json:"results"`
}

// RebalanceUserResponse is a single user response for rebalancing used in batching
type RebalanceUserResponse struct {
	RequestID       string             `json:"request_id"`
	AdjustedTargets map[string]float64 `json:"adjusted_targets"`
	Skipped         []string           `json:"skipped"`
}
