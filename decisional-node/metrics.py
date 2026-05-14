from prometheus_client import Counter, Histogram, start_http_server

# Commands Received
# Labels:
#   command - CMD_SYNC_DAILY, CMD_SYNC_INTRADAY, CMD_GENERATE,
#              CMD_REBALANCE_USER, CMD_REBALANCE_BATCH, CMD_FORECAST
#   status  - "success", "error", "unknown_command"
COMMANDS_RECEIVED = Counter(
    "investpilot_decisional_commands_received_total",
    "Total RabbitMQ commands received and processed by the decisional node.",
    ["command", "status"],
)

# Command Processing Duration
# This is the pipeline duration equivalent - Go only publishes commands and
# cannot know when Python finishes. Python measures the actual execution time.
COMMAND_DURATION = Histogram(
    "investpilot_decisional_command_duration_seconds",
    "Processing duration per command type (pipeline duration measured here, not in Go).",
    ["command"],
    buckets=[0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0],
)

#  Pipeline Duration (full pipeline pass)
# Tracks wall-clock time for composite pipelines that span multiple commands.
# Labels:
#   pipeline - "daily" (CMD_SYNC_DAILY), "intraday" (CMD_SYNC_INTRADAY),
#              "rebalance_batch" (CMD_REBALANCE_BATCH, scales with user count)
PIPELINE_DURATION = Histogram(
    "investpilot_decisional_pipeline_duration_seconds",
    "Wall-clock duration of full pipeline passes executed by the decisional node.",
    ["pipeline"],
    buckets=[1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, 600.0],
)

#  Rebalance Business Metrics
REBALANCE_ASSETS_SKIPPED = Counter(
    "investpilot_decisional_rebalance_assets_skipped_total",
    "Total number of assets skipped during rebalance due to threshold.",
)

REBALANCE_BATCH_USERS = Histogram(
    "investpilot_decisional_rebalance_batch_users",
    "Number of users processed per batch rebalance run.",
    buckets=[1, 5, 10, 25, 50, 100, 250, 500],
)

#  Forecast Metrics
FORECAST_DURATION = Histogram(
    "investpilot_decisional_forecast_duration_seconds",
    "Monte Carlo forecast computation duration in seconds.",
    buckets=[0.5, 1.0, 2.0, 5.0, 10.0, 30.0],
)


def start_metrics_server(port: int = 9090):
    """Start the Prometheus HTTP metrics server on the given port."""
    start_http_server(port)