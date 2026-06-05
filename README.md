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
| **Decisional Node** | Python + Pika | HRP weights, Monte Carlo, rebalance deltas, yfinance sync (pure math, no business logic; no HTTP API, only Prometheus on `:9090`) |
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

## Algorithms

**Hierarchical Risk Parity (HRP)**: chosen over classical mean-variance optimization because mean-variance requires inverting the covariance matrix, which becomes ill-conditioned when assets are highly correlated and produces unstable, concentrated weights. HRP replaces matrix inversion with a hierarchical tree structure. Three steps:

1. **Tree clustering**: convert correlation to distance `d = sqrt(0.5 * (1 - ρ))`, build a single-linkage dendrogram
2. **Quasi-diagonalization**: reorder assets so correlated ones are adjacent (covariance values concentrate around the diagonal)
3. **Recursive bisection**: split the ordered list in halves; assign each half a weight inversely proportional to its inverse-variance portfolio variance; recurse

HRP runs separately on equities and bonds; results are blended via a macro allocation table keyed on the user's risk profile. Implementation: [decisional-node/services/hrp_service.py](decisional-node/services/hrp_service.py).

**Monte Carlo forecast**: Geometric Brownian Motion with drift and volatility estimated from historical daily returns. 10,000 trajectories simulated; output is P10/P50/P90 percentile bands forming a "cone of uncertainty." Implementation: [decisional-node/services/monte_carlo_service.py](decisional-node/services/monte_carlo_service.py).

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

## Security

- **JWT auth** ([RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519)): short-lived access token + family-grouped refresh tokens with rotation; a reused refresh token (signal of theft) invalidates the entire family
- **TOTP 2FA** ([RFC 6238](https://datatracker.ietf.org/doc/html/rfc6238)): secrets encrypted at rest via AES-256-GCM; a DB compromise does not expose 2FA seeds
- **Cloudflare Turnstile**: anti-bot on `/register`, `/login`, `/forgot-password`, verified server-side before bcrypt to avoid CPU spend on bot traffic
- **Rate limiting**: global Gin middleware (token bucket, per-IP)
- **Account lockout**: 5 failed logins on the same email → 15 min lockout (immune to IP rotation)
- **Transactional email**: [Resend](https://resend.com/docs) with custom HTML templates (verification, password reset)
- **Internal endpoints**: `/internal/*` (CronJob targets) require an `X-Internal-Secret` header; never exposed via the public ingress
- **NetworkPolicy**: decisional pod is outbound-only (RabbitMQ + Supabase); inbound denied except TCP 9090 from the `monitoring` namespace for Prometheus scraping

---

## Local Development

**Prerequisites:** Docker, Docker Compose, Node.js, Go 1.23+

**Quick start:**
```bash
make up                # Start all backend services (PostgreSQL, RabbitMQ, Go, Python)
make frontend-dev      # Start frontend Vite dev server in a separate terminal
```

**All available commands:** see [Makefile](Makefile) or run `make help` for the full list (testing, linting, debugging, Kubernetes ops).

**Key commands:**
- `make test-all`: run Go + Python test suites
- `make lint`: lint all services
- `make logs`: tail all service logs
- `make k8s-*`: interact with production k3s cluster

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

## Testing

- **Go**: repository tests run on an in-memory SQLite via `setupTestDB()` (no external DB required); service tests use mocks generated in `internal/mocks/`
- **Python**: pytest suite covers HRP, Monte Carlo, and rebalance delta logic ([decisional-node/tests/](decisional-node/tests/))
- **Frontend**: type-checked at build time via `tsc` (Vite build)
- **CI gate**: the full suite runs on every push/PR via GitHub Actions; deploys are blocked unless CI is green on `main`

```bash
make test-operational     # Go unit tests (handlers + services + repositories)
make test-decisional      # Python pytest
make test-all             # Both
make coverage-operational # Go coverage HTML report
```

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
- **Scheduling:** Kubernetes `CronJob` objects POST to `/internal/*` endpoints (guarded by `X-Internal-Secret`); local dev uses `robfig/cron/v3` in-process
- **Network isolation:** decisional pod restricted by NetworkPolicy to outbound-only traffic; inbound limited to Prometheus scraping from the `monitoring` namespace

### Scheduled jobs (Kubernetes CronJobs)

| CronJob | Schedule | Target endpoint |
|---------|----------|-----------------|
| `cron-pipeline-daily` | 22:00 UTC daily | `POST /internal/pipeline/daily` (daily price sync + HRP regen) |
| `cron-pipeline-intraday` | every 15 min | `POST /internal/pipeline/intraday` |
| `cron-rebalance` | 02:00 UTC, 1st of month | `POST /internal/rebalance` (per-user rebalance) |
| `cron-cleanup` | 03:00 UTC daily | `POST /internal/cleanup` (purge expired sessions/tokens) |

---

## CI/CD

Two GitHub Actions workflows:

| Workflow | Trigger | Jobs |
|----------|---------|------|
| **Continuous Integration** (`ci.yml`) | Every push / PR | `Operational Node - Test, Vet & Lint`, `Decisional Node - Lint, Syntax & Audit`, `Frontend - Lint, Build & Audit` |
| **Build and Deploy** (`deploy.yml`) | CI success on `main` | `Build and Push Images` (3 images → `ghcr.io`, tagged by commit SHA), `Deploy to Kubernetes` (`kubectl set image` + `rollout status`) |

Images are tagged by commit SHA, not `latest`; every deploy is traceable and rollback is `kubectl set image` with a prior SHA.

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

---

## License

MIT License: see [LICENSE](LICENSE) file for details.
