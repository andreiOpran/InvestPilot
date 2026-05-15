package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CommandsPublished tracks every RabbitMQ message the operational node sends
	// Labels:
	//   command - CMD_SYNC_DAILY, CMD_GENERATE, CMD_REBALANCE_USER, CMD_FORECAST
	//   status  - "success" or "error"
	CommandsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "commands_published_total",
			Help:      "Total number of RabbitMQ commands published by the operational node.",
		},
		[]string{"command", "status"},
	)

	// HttpRequestsTotal tracks all HTTP requests handled by the operational node
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled by the operational node.",
		},
		[]string{"method", "path", "status"},
	)

	// HttpRequestDuration tracks latency per route
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"method", "path"},
	)

	// RebalanceStaleDataAborts counts how many times the Go staleness check
	// aborted a rebalance before even publishing to RabbitMQ
	RebalanceStaleDataAborts = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "rebalance_stale_data_aborts_total",
			Help:      "Number of rebalance runs aborted because market data was stale.",
		},
	)
)
