# InvestPilot

![Continuous Integration](https://github.com/andreiOpran/InvestPilot/actions/workflows/ci.yml/badge.svg)
![Build and Deploy](https://github.com/andreiOpran/InvestPilot/actions/workflows/deploy.yml/badge.svg)

A robo-advisory platform for passive "set & forget" investing. Uses Hierarchical Risk Parity for portfolio construction, Monte Carlo for forecasting, and automated monthly rebalancing, all in a paper trading environment.

**Live:** [investpilot.live](https://investpilot.live)

---

## Architecture

Three-service architecture with strict ownership boundaries:

```
Browser → Cloudflare → Traefik (k3s)
  ├── /api/*  → Operational Node (Go/Gin :8081)
  └── /*      → Nginx → React SPA
                    ↓ RabbitMQ (AMQP)
              Decisional Node (Python - math only)
                    ↓ PostgreSQL (Supabase)
```

| Service | Stack | Responsibility |
|---------|-------|---------------|
| **Operational Node** | Go + Gin + GORM | Auth, business logic, DB schema, Stripe payments, cron jobs |
| **Decisional Node** | Python + Pika | HRP weights, Monte Carlo, rebalance deltas, yfinance sync (pure math, no business logic) |
| **Frontend** | React 19 + TypeScript + Vite | SPA served by Nginx in production |
| **Database** | PostgreSQL (Supabase) | Go owns DDL; Python is a DML-only tenant |
| **Message broker** | RabbitMQ (CloudAMQP) | Go publishes commands; Python consumes (prefetch=1) |

---

## Key Design Decisions

- **Python never owns business logic**: it computes math and returns results; Go drives all decisions
- **No live price fetching**: all ETF prices come from `daily_market_data` (synced daily by Python via yfinance); Go checks staleness before rebalancing
- **Model portfolios pre-computed**: Python generates all 15 HRP buckets (risk 1-5 x short/medium/long horizon) daily and persists them; Go reads from DB on rebalance day, no Python call needed
- **Anonymous rebalancing**: Go strips all PII before publishing `CMD_REBALANCE_USER`; Python receives only weight vectors
- **JWT + family-based sessions**: multi-device support with refresh token rotation and reuse detection
- **Cloudflare Turnstile** gates auth flows server-side

---

## RabbitMQ Commands

| Command | Trigger | Python action |
|---------|---------|---------------|
| `CMD_SYNC_DAILY` | Daily cron | Fetch 5y closing prices via yfinance, upsert `daily_market_data` |
| `CMD_SYNC_INTRADAY` | Intraday cron | Fetch 15-min prices |
| `CMD_GENERATE` | After sync | Compute HRP for all 15 buckets, write `model_portfolios` |
| `CMD_REBALANCE_USER` | Monthly cron (per user) | Apply cash-first rule + threshold filter, return adjusted weights |
| `CMD_REBALANCE_BATCH` | Monthly cron | Batch variant of above |
| `CMD_FORECAST` | User request | Monte Carlo (10k scenarios, GBM), write `forecast_results` |

---

## Async Patterns

**Forecast:** Go creates pending `ForecastResult` row, publishes `CMD_FORECAST`, returns `task_id`. Frontend polls `/api/v1/forecast/status/{task_id}` every 2s until Python writes the result.

**Rebalancing:** Monthly cron runs a staleness check, reads pre-computed buckets, publishes per-user commands. Python returns deltas; Go executes share math and DB writes.

---

## Local Development

**Prerequisites:** Docker, Docker Compose, Node.js, Go 1.23+

```bash
# Start all backend services
make up        # PostgreSQL, RabbitMQ, Go backend, Python engine, Adminer

# Frontend (separate terminal)
cd frontend
npm install
npm run dev    # Vite dev server at :3000, proxies /api/* → :8081
```

Other useful commands:
```bash
make down
make rebuild   # Force rebuild containers
make logs

# Go tests
cd operational-node && go test -v ./internal/...

# Frontend build
cd frontend && npm run build
```

**Ports:**

| Service | Port |
|---------|------|
| Frontend (dev) | 3000 |
| Go API | 8081 |
| PostgreSQL | 5432 |
| RabbitMQ AMQP | 5672 |
| RabbitMQ Management | 15672 |
| Adminer | 8082 |

Env files: `.env-operational-node`, `.env-decisional-node`, `.env-frontend` (`.example` siblings in repo).

---

## Production Infrastructure

Self-hosted **k3s** cluster on 3 DigitalOcean VPS nodes (`investpilot` namespace):

| Node | Role | Size |
|------|------|------|
| k3s-master | Control plane + Traefik | 2vCPU / 4GB |
| k3s-worker-1 | Go + Nginx pods (2 replicas) | 2vCPU / 4GB |
| k3s-worker-2 | Python consumer (1 replica) | 2vCPU / 2GB |

- **TLS:** Cloudflare Full Strict + origin certificate
- **CI/CD:** GitHub Actions builds images, pushes to `ghcr.io`, rolling deploy via `kubectl set image`
- **Secrets:** Kubernetes `Secret` objects (`go-secrets`, `python-secrets`)
- **SSH:** Tailscale mesh (port 2222 not exposed publicly)

---

## Observability

Grafana Cloud + Grafana Agent (DaemonSet, ~60MB/node):

- **Infra metrics:** kube-state-metrics + node-exporter for pod restarts, node CPU/RAM/disk
- **Go custom metrics:** `investpilot_operational_*` - HTTP request rate/latency, commands published, rebalance stale aborts
- **Python custom metrics:** `investpilot_decisional_*` - commands received, pipeline duration, forecast duration, rebalance assets skipped
- **Alerts:** CrashLoop, stale pipelines, high 5xx rate, RabbitMQ publish failures, low node memory, routed to email via Grafana Cloud Alertmanager

---

## Frontend Stack

React 19 · TypeScript · Vite · TanStack Query · Zustand · Axios · React Hook Form + Zod · shadcn/ui · Recharts · Stripe.js · Cloudflare Turnstile · Sonner

Key patterns:
- `authStore` (Zustand) holds `accessToken`, user, and `status`
- `useSilentRestore` hydrates session on load via `POST /refresh-token` (httpOnly cookie)
- Axios interceptor handles 401 → silent refresh → retry with a single in-flight refresh queue
- `ProtectedRoute` blocks unauthenticated access and redirects unboarded users to `/onboarding`

---

## Portfolio Logic

1. User completes onboarding questionnaire; Go derives `riskTolerance` (1-5) and `investmentHorizon`, maps to one of 15 HRP bucket keys
2. User deposits funds into wallet; `POST /invest` moves cash to portfolio as `USD` ticker
3. Daily pipeline syncs ETF prices and regenerates all 15 model portfolio weight sets
4. Monthly cron rebalances all users: reads pre-computed weights, publishes per-user delta command, Python returns adjusted targets, Go executes share math and writes new `InvestmentRound`
5. Forecast: Monte Carlo over user's current weights, renders a "cone of uncertainty" chart (P10/P50/P90)

**ETF universe:** VTI, VOO, QQQ, VTV, VUG, IWM, VEA, VWO, VNQ, VNQI, XLF, XLV, XLE, XLK (equities) + BND, TLT, LQD, HYG, BNDX (bonds)
