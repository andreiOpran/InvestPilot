# Observability Plan — InvestPilot Platform
## Grafana Cloud + Grafana Agent on K3s

**Stack:** Grafana Cloud (hosted Prometheus + Grafana + Alertmanager) · Grafana Agent (in-cluster scraper, ~50MB) · Email alerting via Grafana Cloud contact points
**Namespace:** `monitoring` (agent only) · `investpilot` (app metrics scraped from here)
**Custom metrics:** Go operational node (commands sent, HTTP latency) · Python decisional node (commands received, pipeline duration, stale-data failures)

> **Why Grafana Cloud instead of in-cluster Prometheus:**
> The cluster nodes (1GB workers, 2GB master already running etcd + Traefik) do not have
> enough RAM for a full Prometheus + Grafana + Alertmanager stack (~700MB+).
> Grafana Cloud hosts all storage and UI externally. Only the lightweight Grafana Agent
> runs in-cluster (~50MB per node) to scrape and forward metrics.

---

## Architecture Overview

```
                    ┌─────────────── In-cluster ────────────────────┐
                    │                                               │
  investpilot pods  │   Grafana Agent (DaemonSet, ~50MB/node)       │
  (Go :8081/metrics)│     scrapes pods + nodes every 30s            │
  (Python :9090/met)│          │                                    │
                    │          │  remote_write (HTTPS)              │
  node metrics ────►│          │                                    │
  kube-state-metrics│          ▼                                    │
                    └──────────┼────────────────────────────────────┘
                               │
                    ┌──────────▼────────────── Grafana Cloud ───────┐
                    │                                               │
                    │   Hosted Prometheus (stores 14 days)          │
                    │          │                                    │
                    │          ├──► Grafana UI (dashboards)         │
                    │          │       grafana.com/your-stack       │
                    │          │                                    │
                    │          └──► Alertmanager → Email            │
                    │                                               │
                    └───────────────────────────────────────────────┘
```

---

## Stage 11.1 — Create Grafana Cloud Account

1. Go to **grafana.com** → click **Create free account**
2. Create your stack (pick the region closest to your DO region — e.g. EU)
3. Once inside, navigate to **Home → Connections → Add new connection → Hosted Prometheus metrics**
4. Note down these three values — you need them for the agent config:

| Value | Where to find it |
|-------|-----------------|
| **Remote Write URL** | Connections → Prometheus → Details → Remote Write Endpoint |
| **Username** | Same page — numeric ID, e.g. `123456` |
| **API Key** | Connections → API keys → Add API key (role: MetricsPublisher) |

- [x] Create Grafana Cloud account
- [x] Note down Remote Write URL, Username, and API Key

---

## Stage 11.2 — Install Grafana Agent in the Cluster

The Grafana Agent is the only component that runs in your cluster. It scrapes your pods and nodes every 60 seconds and ships the data to Grafana Cloud over HTTPS. It runs as a DaemonSet — one pod per node.

> **Why not scrape the kubelet directly for node CPU/RAM?**
> The kubelet `/metrics` endpoint returns not just node metrics but also **cAdvisor data for every container on the node** — thousands of metric families. Buffering that in memory before the remote_write flush causes the agent to OOMKill consistently, even at 250Mi. Instead, we install `prometheus-node-exporter` as a separate tiny DaemonSet (~20MB RAM) that exposes only node-level CPU/RAM/disk/network with a predictable, small payload (~300 metrics total vs thousands from kubelet).

> **Why disk-backed WAL?**
> By default the agent stores its WAL (Write Ahead Log — the buffer between scraping and sending) at `/tmp/agent`, which is in-memory (tmpfs) and counts against the container memory limit. Each crash during the initial install grew the WAL, and each restart tried to replay a larger WAL, spiking memory further. Moving the WAL to a disk-backed emptyDir volume takes it out of the memory budget entirely.

> **Why `max_shards = 2`?**
> By default the agent's remote_write queue can spin up 50 parallel sender goroutines, each holding its own in-memory buffer. For a small cluster sending a few hundred metrics every 60 seconds, 1–2 shards is more than enough. Capping it eliminates ~48 unnecessary memory allocations.

### Step 1 — Create the credentials secret

Never put credentials in the Helm values file. Store them as a Kubernetes secret:

```bash
kubectl create namespace monitoring

kubectl create secret generic grafana-agent-credentials \
  --namespace monitoring \
  --from-literal=remote_write_url="https://prometheus-prod-xx-xxx.grafana.net/api/prom/push" \
  --from-literal=username="123456" \
  --from-literal=api_key="glc_eyJ..."
```

### Step 2 — Install kube-state-metrics (standalone, lightweight)

kube-state-metrics translates Kubernetes object state (pod restarts, deployment replicas, CronJob success times) into Prometheus metrics. Without it you cannot alert on pod crashes or stale CronJobs. Runs as a single pod (~50MB).

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-state-metrics prometheus-community/kube-state-metrics \
  --namespace monitoring \
  --set resources.requests.memory=40Mi \
  --set resources.limits.memory=80Mi \
  --set resources.requests.cpu=10m \
  --set resources.limits.cpu=100m
```

### Step 3 — Install node-exporter (node CPU/RAM/disk)

node-exporter is a tiny binary (~20MB RAM) that exposes only node-level OS metrics — CPU, RAM, disk, network — with no container-level data. Runs as a DaemonSet (one pod per node).

```bash
helm install node-exporter prometheus-community/prometheus-node-exporter \
  --namespace monitoring \
  --set resources.requests.memory=20Mi \
  --set resources.limits.memory=50Mi \
  --set resources.requests.cpu=10m \
  --set resources.limits.cpu=100m
```

Verify both are running:

```bash
kubectl get pods -n monitoring
# kube-state-metrics-xxxxx     Running   (x1)
# node-exporter-xxxxx          Running   (x3, one per node)

kubectl get svc -n monitoring
# kube-state-metrics                              8080/TCP
# node-exporter-prometheus-node-exporter          9100/TCP
# Note the exact service name — you need it in the agent scrape config below
```

### Step 4 — Create the agent values file

Create `monitoring/agent-values.yaml`:

```yaml
agent:
  mode: "flow"

  # Move WAL off tmpfs (in-memory) onto disk-backed emptyDir.
  # This takes WAL data out of the container memory limit entirely.
  storagePath: /var/lib/grafana-agent

  resources:
    requests:
      memory: "60Mi"
      cpu: "50m"
    limits:
      memory: "150Mi"
      cpu: "200m"

  configReloader:
    resources:
      requests:
        memory: "20Mi"
        cpu: "10m"
      limits:
        memory: "40Mi"
        cpu: "50m"

  extraEnv:
    - name: GRAFANA_REMOTE_WRITE_URL
      valueFrom:
        secretKeyRef:
          name: grafana-agent-credentials
          key: remote_write_url
    - name: GRAFANA_USERNAME
      valueFrom:
        secretKeyRef:
          name: grafana-agent-credentials
          key: username
    - name: GRAFANA_API_KEY
      valueFrom:
        secretKeyRef:
          name: grafana-agent-credentials
          key: api_key

  configMap:
    create: true
    content: |

      // == InvestPilot pod scraping ==========================================
      // Discovers pods in the investpilot namespace that have
      // prometheus.io/scrape: "true" annotation and scrapes them.

      discovery.kubernetes "investpilot_pods" {
        role = "pod"
        namespaces {
          names = ["investpilot"]
        }
      }

      discovery.relabel "investpilot_pods" {
        targets = discovery.kubernetes.investpilot_pods.targets

        // Only scrape pods that opted in
        rule {
          source_labels = ["__meta_kubernetes_pod_annotation_prometheus_io_scrape"]
          action        = "keep"
          regex         = "true"
        }
        // Use the custom /metrics path if set
        rule {
          source_labels = ["__meta_kubernetes_pod_annotation_prometheus_io_path"]
          action        = "replace"
          target_label  = "__metrics_path__"
          regex         = "(.+)"
        }
        // Use the custom port if set
        rule {
          source_labels = ["__address__", "__meta_kubernetes_pod_annotation_prometheus_io_port"]
          action        = "replace"
          regex         = "([^:]+)(?:\\d+)?;(\\d+)"
          replacement   = "$1:$2"
          target_label  = "__address__"
        }
        rule {
          source_labels = ["__meta_kubernetes_pod_label_app"]
          target_label  = "app"
        }
        rule {
          source_labels = ["__meta_kubernetes_namespace"]
          target_label  = "namespace"
        }
        rule {
          source_labels = ["__meta_kubernetes_pod_name"]
          target_label  = "pod"
        }
      }

      prometheus.scrape "investpilot_pods" {
        targets         = discovery.relabel.investpilot_pods.output
        forward_to      = [prometheus.remote_write.grafana_cloud.receiver]
        scrape_interval = "60s"
      }

      // == kube-state-metrics ===============================================
      // Pod restarts, deployment replicas, CronJob last success time.

      prometheus.scrape "kube_state_metrics" {
        targets = [{
          __address__ = "kube-state-metrics.monitoring.svc.cluster.local:8080",
        }]
        forward_to      = [prometheus.remote_write.grafana_cloud.receiver]
        scrape_interval = "60s"
      }

      // == node-exporter (node CPU/RAM/disk/network) ========================
      // ~300 metrics per node. Safe to scrape — no cAdvisor container data.
      // Service name format: <release-name>-prometheus-node-exporter

      prometheus.scrape "node_exporter" {
        targets = [{
          __address__ = "node-exporter-prometheus-node-exporter.monitoring.svc.cluster.local:9100",
        }]
        forward_to      = [prometheus.remote_write.grafana_cloud.receiver]
        scrape_interval = "60s"
      }

      // == Remote write to Grafana Cloud =====================================

      prometheus.remote_write "grafana_cloud" {
        endpoint {
          url = env("GRAFANA_REMOTE_WRITE_URL")
          basic_auth {
            username = env("GRAFANA_USERNAME")
            password = env("GRAFANA_API_KEY")
          }
          queue_config {
            min_shards           = 1
            max_shards           = 2    // default 50 — caps parallel sender goroutines
            max_samples_per_send = 500  // batch 500 samples per HTTP call (default 100)
            batch_send_deadline  = "10s"
          }
        }
        wal {
          truncate_frequency = "30m"   // flush WAL every 30min (default 2h)
          max_keepalive_time = "1h"    // drop WAL data older than 1h (default 8h)
        }
        external_labels = {
          cluster = "investpilot-k3s",
        }
      }

# Explicit disk-backed volume for the WAL — emptyDir without medium: Memory
# is written to the node's disk, not counted against container memory limit.
extraVolumes:
  - name: agent-wal
    emptyDir: {}

extraVolumeMounts:
  - name: agent-wal
    mountPath: /var/lib/grafana-agent

# The agent needs cluster-wide read access to discover pods and nodes
rbac:
  create: true

serviceAccount:
  create: true
```

### Step 5 — Install Grafana Agent

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

helm install grafana-agent grafana/grafana-agent \
  --namespace monitoring \
  -f monitoring/agent-values.yaml
```

### Step 6 — Verify everything is running and shipping

```bash
kubectl get pods -n monitoring
# Expected (all Running, restarts stable):
# grafana-agent-xxxxx                        2/2 Running   (x3, one per node)
# kube-state-metrics-xxxxx                   1/1 Running   (x1)
# node-exporter-prometheus-node-exporter-xxx 1/1 Running   (x3, one per node)

# Check agent logs — confirm WAL replay completed and data is being sent
kubectl logs -n monitoring -l app.kubernetes.io/name=grafana-agent --tail=20
# Healthy output looks like:
# "Done replaying WAL"
# "Remote storage resharding from=1 to=2"  ← normal auto-tuning
# No OOMKilled, no error lines

# Check node memory usage after the full stack is running
kubectl top node
# Master and workers should be well below 80%
```

Then in Grafana Cloud → **Explore** → switch to **code mode** and verify each source is flowing:

```promql
# kube-state-metrics working — pod-level cluster state
kube_pod_info

# node-exporter working — node RAM available
node_memory_MemAvailable_bytes

# node-exporter working — node CPU
node_cpu_seconds_total

# agent itself reporting
up{cluster="investpilot-k3s"}
```

> **Tip:** If `node_memory_MemAvailable_bytes` returns no data, check the node-exporter service name matches exactly what's in the agent config. Run `kubectl get svc -n monitoring` and verify the name before the `.monitoring.svc.cluster.local:9100` part.

- [x] All pods Running with stable restart count
- [x] `kube_pod_info` returns data in Grafana Cloud
- [x] `node_memory_MemAvailable_bytes` returns 3 results (one per node)
- [x] `node_cpu_seconds_total` returns data

---

## Stage 11.3 — Custom Metrics: Go Operational Node

### Step 1 — Add the dependency

```bash
cd operational-node
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
go get github.com/prometheus/client_golang/prometheus/promhttp
```

### Step 2 — Create the metrics registry

Create `operational-node/internal/metrics/metrics.go`:

```go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CommandsPublished tracks every RabbitMQ message the operational node sends.
	// Labels:
	//   command — CMD_SYNC_DAILY, CMD_GENERATE, CMD_REBALANCE_USER, CMD_FORECAST
	//   status  — "success" or "error"
	CommandsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "commands_published_total",
			Help:      "Total number of RabbitMQ commands published by the operational node.",
		},
		[]string{"command", "status"},
	)

	// HttpRequestsTotal tracks all HTTP requests handled by the operational node.
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled by the operational node.",
		},
		[]string{"method", "path", "status"},
	)

	// HttpRequestDuration tracks latency per route.
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
	// aborted a rebalance before even publishing to RabbitMQ.
	RebalanceStaleDataAborts = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "investpilot",
			Subsystem: "operational",
			Name:      "rebalance_stale_data_aborts_total",
			Help:      "Number of rebalance runs aborted because market data was stale.",
		},
	)
)
```

### Step 3 — Expose the /metrics endpoint

In `internal/router/routes.go`, register the handler and add the metrics middleware before route groups:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

r.Use(middleware.MetricsMiddleware())
r.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

Create `internal/middleware/metrics.go`:

```go
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/andreiOpran/licenta/operational-node/internal/metrics"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())
		metrics.HttpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
```

> **Security note:** `/metrics` is on port `8081` but Traefik only forwards `/api/*` externally — the endpoint is only reachable inside the cluster by the Grafana Agent.

### Step 4 — Instrument your RabbitMQ publisher

```go
import "github.com/yourusername/licenta/operational-node/internal/metrics"

func (r *RabbitMQClient) PublishCommand(command string, payload interface{}) error {
	err := ch.Publish(/* ... */)

	status := "success"
	if err != nil {
		status = "error"
	}
	metrics.CommandsPublished.WithLabelValues(command, status).Inc()

	return err
}
```

### Step 5 — Instrument the rebalance staleness check

```go
func (s *PipelineService) RunRebalance() error {
	stale, err := s.repo.CheckMarketDataStaleness()
	if err != nil || stale {
		metrics.RebalanceStaleDataAborts.Inc()  // ← add this
		log.Error("Rebalance aborted: stale market data")
		return fmt.Errorf("stale market data, aborting rebalance")
	}
	// ... rest of rebalance logic ...
}
```

### Step 6 — Add pod annotations to the Deployment

Edit `k8s/operational-node-deployment.yaml`:

```yaml
spec:
  template:
    metadata:
      labels:
        app: operational-node
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8081"
        prometheus.io/path: "/metrics"
```

---

## Stage 11.4 — Custom Metrics: Python Decisional Node

### Step 1 — Add the dependency

Add to `decisional-node/requirements.txt`:

```
prometheus-client==0.20.0
```

### Step 2 — Create the metrics module

Create `decisional-node/metrics.py`:

```python
from prometheus_client import Counter, Histogram, start_http_server

# ── Commands Received ────────────────────────────────────────────────────────
# Labels:
#   command — CMD_SYNC_DAILY, CMD_SYNC_INTRADAY, CMD_GENERATE,
#              CMD_REBALANCE_USER, CMD_REBALANCE_BATCH, CMD_FORECAST
#   status  — "success", "error", "unknown_command"
COMMANDS_RECEIVED = Counter(
    "investpilot_decisional_commands_received_total",
    "Total RabbitMQ commands received and processed by the decisional node.",
    ["command", "status"],
)

# ── Stale Data Failures ──────────────────────────────────────────────────────
# Labels:
#   command — which command triggered the stale check
#   reason  — "price_stale", "no_data", "ticker_missing", "db_error"
STALE_DATA_FAILURES = Counter(
    "investpilot_decisional_stale_data_failures_total",
    "Number of commands aborted because market data was stale or missing.",
    ["command", "reason"],
)

# ── Command Processing Duration ──────────────────────────────────────────────
# This is the pipeline duration equivalent — Go only publishes commands and
# cannot know when Python finishes. Python measures the actual execution time.
COMMAND_DURATION = Histogram(
    "investpilot_decisional_command_duration_seconds",
    "Processing duration per command type (pipeline duration measured here, not in Go).",
    ["command"],
    buckets=[0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0],
)

# ── Pipeline Duration (full pipeline pass) ───────────────────────────────────
# Tracks wall-clock time for composite pipelines that span multiple commands
# (e.g. CMD_SYNC_DAILY + CMD_GENERATE run sequentially as the daily pipeline).
# Labels:
#   pipeline — "daily" (sync+generate), "intraday" (sync only)
PIPELINE_DURATION = Histogram(
    "investpilot_decisional_pipeline_duration_seconds",
    "Wall-clock duration of full pipeline passes executed by the decisional node.",
    ["pipeline"],
    buckets=[1.0, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0, 600.0],
)

# ── Rebalance Business Metrics ───────────────────────────────────────────────
REBALANCE_ASSETS_SKIPPED = Counter(
    "investpilot_decisional_rebalance_assets_skipped_total",
    "Total number of assets skipped during rebalance due to threshold.",
)

REBALANCE_BATCH_USERS = Histogram(
    "investpilot_decisional_rebalance_batch_users",
    "Number of users processed per batch rebalance run.",
    buckets=[1, 5, 10, 25, 50, 100, 250, 500],
)

# ── Forecast Metrics ─────────────────────────────────────────────────────────
FORECAST_DURATION = Histogram(
    "investpilot_decisional_forecast_duration_seconds",
    "Monte Carlo forecast computation duration in seconds.",
    buckets=[0.5, 1.0, 2.0, 5.0, 10.0, 30.0],
)


def start_metrics_server(port: int = 9090):
    """Start the Prometheus HTTP metrics server on the given port."""
    start_http_server(port)
```

### Step 3 — Start the metrics server in app.py

Edit `decisional-node/app.py`:

```python
from metrics import (
    start_metrics_server,
    COMMANDS_RECEIVED,
    STALE_DATA_FAILURES,
    COMMAND_DURATION,
    PIPELINE_DURATION,
    REBALANCE_ASSETS_SKIPPED,
    REBALANCE_BATCH_USERS,
    FORECAST_DURATION,
)
import time

def main():
    repo = DataRepository(settings.DATABASE_URL)

    # ── Start Prometheus metrics server on :9090 ──────────────────────────
    start_metrics_server(port=9090)

    # ... existing RabbitMQ connection retry loop ...

    def callback(ch, method, properties, body):
        try:
            message = json.loads(body)
            command = message.get("command")
            payload = message.get("payload")
            response = None
            start_time = time.time()

            if command == "CMD_SYNC_DAILY":
                response = process_sync_daily(payload, repo)
            elif command == "CMD_SYNC_INTRADAY":
                response = process_sync_intraday(payload, repo)
            elif command == "CMD_GENERATE":
                response = process_generate_models(payload, repo)
            elif command == "CMD_REBALANCE_USER":
                response = process_rebalance_user(payload, repo)
            elif command == "CMD_REBALANCE_BATCH":
                response = process_rebalance_batch(payload, repo)
            elif command == "CMD_FORECAST":
                response = process_forecast(payload, repo)
            else:
                logging.warning(f"Unknown command: {command}")
                COMMANDS_RECEIVED.labels(command=command or "UNKNOWN", status="unknown_command").inc()
                ch.basic_ack(delivery_tag=method.delivery_tag)
                return

            COMMAND_DURATION.labels(command=command).observe(time.time() - start_time)

            if response and "error" in response:
                COMMANDS_RECEIVED.labels(command=command, status="error").inc()
            else:
                COMMANDS_RECEIVED.labels(command=command, status="success").inc()

            if properties.reply_to and response is not None:
                ch.basic_publish(
                    exchange='',
                    routing_key=properties.reply_to,
                    properties=pika.BasicProperties(
                        correlation_id=properties.correlation_id,
                        content_type="application/json"
                    ),
                    body=json.dumps(response)
                )

        except Exception as e:
            logging.error(f"Error processing message: {e}")
            COMMANDS_RECEIVED.labels(command=command or "UNKNOWN", status="error").inc()
        finally:
            ch.basic_ack(delivery_tag=method.delivery_tag)
```

### Step 4 — Instrument command_handlers.py for stale data

Edit `decisional-node/handlers/command_handlers.py`:

```python
import time
import logging
from datetime import datetime, timedelta
from metrics import STALE_DATA_FAILURES, REBALANCE_ASSETS_SKIPPED, REBALANCE_BATCH_USERS, FORECAST_DURATION, PIPELINE_DURATION


def _check_data_staleness(repo, tickers: list, max_stale_days: int = 2) -> tuple[bool, str]:
    """Returns (is_stale, reason). Checks every ticker has a fresh price row."""
    cutoff = datetime.utcnow() - timedelta(days=max_stale_days)
    try:
        if repo.get_tickers_older_than(cutoff, tickers):
            return True, "price_stale"
        if repo.get_missing_tickers(tickers):
            return True, "ticker_missing"
        return False, ""
    except Exception as e:
        logging.error(f"Staleness check DB error: {e}")
        return True, "db_error"


def process_generate_models(payload: dict, repo):
    all_tickers = payload.get("equity_tickers", []) + payload.get("bond_tickers", [])

    # ── Staleness check ───────────────────────────────────────────────────
    is_stale, reason = _check_data_staleness(repo, all_tickers)
    if is_stale:
        STALE_DATA_FAILURES.labels(command="CMD_GENERATE", reason=reason).inc()
        logging.error(f"CMD_GENERATE aborted: stale data ({reason})")
        return {"error": f"stale_data:{reason}"}
    # ─────────────────────────────────────────────────────────────────────

    # ... rest of existing logic ...


def process_rebalance_user(payload: dict, repo):
    req_id = payload.get("request_id")
    current_alloc = payload.get("current_allocation", {})
    target_weights = payload.get("target_weights", {})
    threshold = payload.get("threshold", 0.02)
    cash_first = payload.get("cash_first", True)

    if not req_id or not target_weights:
        return {"error": "request_id and target_weights required"}

    adjusted, skipped = compute_rebalance(current_alloc, target_weights, threshold, cash_first)

    # ── Track skipped assets ──────────────────────────────────────────────
    if skipped:
        REBALANCE_ASSETS_SKIPPED.inc(len(skipped))
    # ─────────────────────────────────────────────────────────────────────

    return {"request_id": req_id, "adjusted_targets": adjusted, "skipped": skipped}


def process_rebalance_batch(payload: dict, repo):
    threshold = payload.get("threshold", 0.02)
    cash_first = payload.get("cash_first", True)
    users = payload.get("users", [])

    # ── Track batch size ──────────────────────────────────────────────────
    REBALANCE_BATCH_USERS.observe(len(users))
    # ─────────────────────────────────────────────────────────────────────

    results = []
    for u_req in users:
        adjusted, skipped = compute_rebalance(
            u_req.get("current_allocation", {}),
            u_req.get("target_weights", {}),
            threshold, cash_first
        )
        if skipped:
            REBALANCE_ASSETS_SKIPPED.inc(len(skipped))
        results.append({"request_id": u_req.get("request_id"), "adjusted_targets": adjusted, "skipped": skipped})

    return {"results": results}


def process_sync_daily(payload: dict, repo):
    start = time.time()

    # ... existing sync logic ...

    # ── Track daily pipeline duration ─────────────────────────────────────
    PIPELINE_DURATION.labels(pipeline="daily").observe(time.time() - start)
    # ─────────────────────────────────────────────────────────────────────


def process_sync_intraday(payload: dict, repo):
    start = time.time()

    # ... existing intraday sync logic ...

    # ── Track intraday pipeline duration ──────────────────────────────────
    PIPELINE_DURATION.labels(pipeline="intraday").observe(time.time() - start)
    # ─────────────────────────────────────────────────────────────────────


def process_forecast(payload: dict, repo):
    start = time.time()

    # ... existing forecast logic ...

    # ── Track forecast duration ───────────────────────────────────────────
    FORECAST_DURATION.observe(time.time() - start)
    # ─────────────────────────────────────────────────────────────────────
```

### Step 5 — Expose port 9090 in the Deployment

Edit `k8s/decisional-deployment.yaml`:

```yaml
spec:
  template:
    metadata:
      labels:
        app: decisional-node
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      nodeSelector:
        kubernetes.io/hostname: k3s-worker-2
      containers:
        - name: decisional-node
          image: ghcr.io/andreiopran/investpilot-decisional-node:latest
          envFrom:
            - secretRef:
                name: python-secrets
          ports:
            - name: metrics
              containerPort: 9090
              protocol: TCP
```

> The decisional node still has no Service for HTTP traffic. The metrics port is only reachable inside the cluster by the Grafana Agent scraping the pod IP directly — this is correct and safe.

---

## Stage 11.5 — Alerting in Grafana Cloud

Grafana Cloud hosts the Alertmanager. You configure alert rules and email routing entirely in the UI — no YAML files to apply to the cluster.

### Step 1 — Configure email contact point

**Alerting → Contact points → Add contact point**

- Name: `email-critical`
- Type: **Email**
- Addresses: `your-personal@email.com`
- Enable: **Send resolved notifications**

Create a second one named `email-warnings` for the same address.

### Step 2 — Configure notification policy

**Alerting → Notification policies → Edit root policy**

```
Default policy:
  Contact point: email-warnings
  Group by: [alertname, namespace]
  Group wait: 30s | Group interval: 5m | Repeat interval: 4h

  ├── severity=critical                       → email-critical  (repeat: 1h)
  ├── alertname=DecisionalStaleDataFailure    → email-critical  (repeat: 30m)
  └── alertname=OperationalRebalanceStaleAbort→ email-critical  (repeat: 30m)
```

### Step 3 — Create alert rules

**Alerting → Alert rules → New alert rule**

Set the **data source** to your Grafana Cloud Prometheus, **folder** to `InvestPilot`, **evaluation group** to `investpilot-alerts`, and **evaluation interval** to `1m` on the group.

---

#### Pod CrashLooping
```promql
rate(kube_pod_container_status_restarts_total{namespace="investpilot"}[5m]) > 0
```
Pending: **2m** · Severity: `critical`
Summary: `Pod {{ $labels.pod }} is crash-looping`

---

#### Pod Not Ready
```promql
kube_pod_status_ready{namespace="investpilot", condition="true"} == 0
```
Pending: **5m** · Severity: `warning`

---

#### Deployment Replicas Mismatch
```promql
kube_deployment_spec_replicas{namespace="investpilot"}
  != kube_deployment_status_available_replicas{namespace="investpilot"}
```
Pending: **5m** · Severity: `warning`

---

#### CronJob Failed
```promql
kube_job_status_failed{namespace="investpilot"} > 0
```
Pending: **1m** · Severity: `critical`

---

#### Daily Pipeline Stale (>25h without success)
```promql
time() - kube_cronjob_status_last_successful_time{
  namespace="investpilot", cronjob="cron-pipeline-daily"
} > 90000
```
Pending: **0m** · Severity: `critical`
Summary: `Daily pipeline has not succeeded in >25 hours`

---

#### Intraday Pipeline Stale (>45min without success)
```promql
time() - kube_cronjob_status_last_successful_time{
  namespace="investpilot", cronjob="cron-pipeline-intraday"
} > 2700
```
Pending: **0m** · Severity: `warning`

---

#### Monthly Rebalance Stale (>35 days)
```promql
time() - kube_cronjob_status_last_successful_time{
  namespace="investpilot", cronjob="cron-rebalance"
} > 3024000
```
Pending: **0m** · Severity: `warning`

---

#### Decisional Node — Stale Data Failure
```promql
increase(investpilot_decisional_stale_data_failures_total[5m]) > 0
```
Pending: **0m** · Severity: `critical`
Summary: `Decisional node aborted {{ $labels.command }} due to stale market data ({{ $labels.reason }})`
Description: `Model portfolio generation or rebalance did not run. Check the daily sync pipeline.`

---

#### Operational Node — Rebalance Stale Abort
```promql
increase(investpilot_operational_rebalance_stale_data_aborts_total[5m]) > 0
```
Pending: **0m** · Severity: `critical`
Summary: `Go node aborted rebalance — stale market data detected before publishing`

---

#### High Command Error Rate
```promql
rate(investpilot_decisional_commands_received_total{status="error"}[5m])
  /
rate(investpilot_decisional_commands_received_total[5m])
> 0.1
```
Pending: **5m** · Severity: `warning`
Summary: `Command error rate >10% on {{ $labels.command }}`

---

#### RabbitMQ Publish Failures
```promql
increase(investpilot_operational_commands_published_total{status="error"}[5m]) > 0
```
Pending: **0m** · Severity: `critical`
Summary: `Operational node failing to publish {{ $labels.command }} to RabbitMQ`

---

#### Go API — High 5xx Error Rate
```promql
rate(investpilot_operational_http_requests_total{status=~"5.."}[5m])
  /
rate(investpilot_operational_http_requests_total[5m])
> 0.05
```
Pending: **2m** · Severity: `critical`
Summary: `Go API 5xx error rate >5% on {{ $labels.method }} {{ $labels.path }}`

---

#### Node High Memory (<15% free)
```promql
node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes < 0.15
```
Pending: **5m** · Severity: `warning`
Summary: `Node {{ $labels.instance }} has <15% free memory`

---

#### Node Critical Memory (<5% free)
```promql
node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes < 0.05
```
Pending: **2m** · Severity: `critical`
Summary: `Node {{ $labels.instance }} critically low on memory — OOM risk`

---

#### Node High CPU (>85% for 10m)
```promql
100 - (avg by(instance)(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 85
```
Pending: **10m** · Severity: `warning`

---

## Stage 11.6 — Grafana Cloud Dashboards

### Pre-built dashboards to import

**Dashboards → Import** → paste the ID → select your Grafana Cloud Prometheus as the data source.

| ID | Dashboard | What it shows |
|----|-----------|---------------|
| **15760** | Kubernetes Cluster Overview | Node count, pod status, resource usage cluster-wide |
| **6417** | Pod Resource Consumption | CPU/RAM per pod — set namespace filter to `investpilot` |
| **1860** | Node Exporter Full | Per-VPS CPU, RAM, disk, network |
| **13659** | Traefik 2.x | HTTP request rates, error rates, P99 latency |
| **14584** | Kubernetes CronJobs | Last success time, failed jobs, job duration |

### Custom InvestPilot dashboard

Create a new dashboard named **"InvestPilot — Application"** with these panels:

#### Panel 1 — Commands Published by Operational Node
```promql
sum by (command, status) (
  rate(investpilot_operational_commands_published_total[5m])
)
```
Visualization: **Bar chart** · stack by `status` · Title: "Commands Published (ops/sec)"

#### Panel 2 — Commands Received by Decisional Node
```promql
sum by (command, status) (
  rate(investpilot_decisional_commands_received_total[5m])
)
```
Visualization: **Bar chart** · same style

#### Panel 3 — Command Error Rate %
```promql
100 * (
  sum(rate(investpilot_decisional_commands_received_total{status="error"}[5m]))
  /
  sum(rate(investpilot_decisional_commands_received_total[5m]))
)
```
Visualization: **Gauge** · thresholds: 0=green, 5=yellow, 10=red

#### Panel 4 — Stale Data Aborts (last 24h)
```promql
sum(increase(investpilot_decisional_stale_data_failures_total[24h]))
```
Visualization: **Stat** · red when > 0, green when 0

#### Panel 5 — Go Rebalance Stale Aborts (last 24h)
```promql
sum(increase(investpilot_operational_rebalance_stale_data_aborts_total[24h]))
```
Visualization: **Stat** · same style

#### Panel 6 — Command Duration P99
```promql
histogram_quantile(0.99,
  sum by (command, le) (
    rate(investpilot_decisional_command_duration_seconds_bucket[5m])
  )
)
```
Visualization: **Time series** · one line per `command`

#### Panel 7 — Forecast Duration P50 / P95
```promql
# Query A — P50
histogram_quantile(0.50, rate(investpilot_decisional_forecast_duration_seconds_bucket[5m]))

# Query B — P95
histogram_quantile(0.95, rate(investpilot_decisional_forecast_duration_seconds_bucket[5m]))
```

#### Panel 8 — Assets Skipped in Rebalance
```promql
rate(investpilot_decisional_rebalance_assets_skipped_total[5m])
```

#### Panel 9 — Go API Request Rate & Error Rate
```promql
# Query A — request rate by path
sum by (method, path, status) (
  rate(investpilot_operational_http_requests_total[5m])
)

# Query B — P95 latency per path
histogram_quantile(0.95,
  sum by (path, le) (
    rate(investpilot_operational_http_request_duration_seconds_bucket[5m])
  )
)
```
Visualization: **Time series** · Query A as bar chart overlay, Query B as lines

#### Panel 10 — Pipeline Duration P50 / P95 (Decisional Node)
```promql
# Query A — P50
histogram_quantile(0.50,
  sum by (pipeline, le) (
    rate(investpilot_decisional_pipeline_duration_seconds_bucket[5m])
  )
)

# Query B — P95
histogram_quantile(0.95,
  sum by (pipeline, le) (
    rate(investpilot_decisional_pipeline_duration_seconds_bucket[5m])
  )
)
```
Visualization: **Time series** · one line per `pipeline` label · unit = seconds

#### Panel 11 — Pod Restarts
```promql
sum by (pod) (
  kube_pod_container_status_restarts_total{namespace="investpilot"}
)
```
Visualization: **Table** · sort descending


#### Panel 12 — CronJob Last Success
```promql
kube_cronjob_status_last_successful_time{namespace="investpilot"}
```
Visualization: **Table** · field override: unit = `dateTimeFromNow`

---

## Stage 11.7 — Apply Everything & Verify

### Final directory structure

```
monitoring/
└── agent-values.yaml     # Grafana Agent Helm values (scrape config + remote_write)
                          # No other files — Grafana, Prometheus, Alertmanager are hosted

k8s/
├── operational-node-deployment.yaml   # updated: prometheus.io/scrape annotations
└── decisional-deployment.yaml         # updated: prometheus.io/scrape annotations + port 9090
```

### Apply in order

```bash
# 1. Create namespace + credentials secret
kubectl create namespace monitoring

kubectl create secret generic grafana-agent-credentials \
  --namespace monitoring \
  --from-literal=remote_write_url="YOUR_REMOTE_WRITE_URL" \
  --from-literal=username="YOUR_USERNAME" \
  --from-literal=api_key="YOUR_API_KEY"

# 2. Install kube-state-metrics
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-state-metrics prometheus-community/kube-state-metrics \
  --namespace monitoring \
  --set resources.requests.memory=40Mi \
  --set resources.limits.memory=80Mi \
  --set resources.requests.cpu=10m \
  --set resources.limits.cpu=100m

# 3. Install node-exporter (node CPU/RAM/disk)
helm install node-exporter prometheus-community/prometheus-node-exporter \
  --namespace monitoring \
  --set resources.requests.memory=20Mi \
  --set resources.limits.memory=50Mi \
  --set resources.requests.cpu=10m \
  --set resources.limits.cpu=100m

# 4. Install Grafana Agent
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install grafana-agent grafana/grafana-agent \
  --namespace monitoring \
  -f monitoring/agent-values.yaml

# 5. Apply updated app manifests (annotations + metrics port on decisional node)
kubectl apply -f k8s/ -n investpilot
```

### Verification checklist

```bash
# ── Stack health ─────────────────────────────────────────────────────────
kubectl get pods -n monitoring
# grafana-agent-xxxxx                        2/2 Running  (x3, one per node)
# kube-state-metrics-xxxxx                   1/1 Running  (x1)
# node-exporter-prometheus-node-exporter-xxx 1/1 Running  (x3, one per node)

kubectl get svc -n monitoring
# kube-state-metrics                             8080/TCP
# node-exporter-prometheus-node-exporter         9100/TCP

kubectl top node
# All nodes well below 80% RAM

# ── Confirm agent is shipping ────────────────────────────────────────────
kubectl logs -n monitoring -l app.kubernetes.io/name=grafana-agent --tail=20
# Look for: "Done replaying WAL" and "Remote storage resharding"
# No OOMKilled, no error lines

# ── Metrics visible in Grafana Cloud ────────────────────────────────────
# Grafana Cloud → Explore → code mode — type each query manually (no paste)
# kube_pod_info                          → kube-state-metrics flowing
# node_memory_MemAvailable_bytes         → node-exporter flowing (3 results)
# node_cpu_seconds_total                 → node CPU flowing
# up{cluster="investpilot-k3s"}          → agent itself reporting

# ── Test alert email ─────────────────────────────────────────────────────
# Grafana Cloud → Alerting → Contact points → email-critical → Send test
# Check inbox within 30 seconds
```

---

## Summary — What You Get

| Layer | Component | Runs where | RAM cost in-cluster |
|-------|-----------|-----------|----------|
| **Scraping** | Grafana Agent | DaemonSet, all 3 nodes | ~60MB × 3 |
| **Cluster state** | kube-state-metrics | Single pod | ~50MB |
| **Node CPU/RAM/disk** | node-exporter | DaemonSet, all 3 nodes | ~20MB × 3 |
| **Storage (14 days)** | Grafana Cloud Prometheus | Hosted externally | 0MB |
| **Dashboards** | Grafana Cloud UI | Hosted externally | 0MB |
| **Alerting + email** | Grafana Cloud Alertmanager | Hosted externally | 0MB |
| **Go custom metrics** | In your app binary | Operational node pod | 0MB extra |
| **Python custom metrics** | In your app process | Decisional node pod | ~5MB extra |
| **Total in-cluster overhead** | | | **~290MB** vs ~700MB with full stack |

> **Key architectural decisions made during implementation:**
> - **Kubelet scrape removed** — returns thousands of cAdvisor metrics that OOMKilled the agent consistently even at 250Mi. Replaced with node-exporter (~300 metrics, predictable payload).
> - **WAL moved to disk** — `storagePath: /var/lib/grafana-agent` + explicit emptyDir volume takes WAL out of the container memory budget. Prevents crash-loop WAL accumulation.
> - **`max_shards = 2`** — caps parallel remote_write sender goroutines. Default of 50 wastes memory on a small cluster sending a few hundred metrics every 60s.
> - **60s scrape interval** — halves metric volume vs the default 30s. Sufficient resolution for thesis-level observability.