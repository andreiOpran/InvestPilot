# Robo-Advisory Platform: Master Development Plan

## 1. Project Overview

This project is an Automated Wealth Management (Robo-Advisory) platform using a "Set & Forget" passive investment strategy. It operates in a simulated "Paper Trading" environment. The system accepts user funds, automatically calculates an optimal portfolio using the Hierarchical Risk Parity (HRP) algorithm, and executes simulated trades.

---

## 2. Tech Stack & Architecture

**Operational Node (Backend API):** Golang (Gin Framework, GORM). Owns all users, wallets, auth (including 2FA), payments, database schema, and frontend serving. The single source of truth for all business policy configuration.

**Decisional Node (Math Engine):** Python 3 (FastAPI, Uvicorn, Pandas, SciPy, yfinance). A private microservice and pure mathematical engine. It computes HRP weights and Monte Carlo simulations. It holds no business state — all outputs are written to the shared database for Go to consume. Auto-generates Swagger UI at `/docs`.

**Database:** PostgreSQL. Owned and schema-managed exclusively by Go (GORM AutoMigrate). Python is a writer/reader tenant — it never runs DDL. Stores users, balances, asset holdings, historical market data, pre-computed model portfolios, and forecast results.

**Message Broker:** RabbitMQ (AMQP). Orchestrates all asynchronous background tasks (`CMD_SYNC`, `CMD_GENERATE`, `CMD_REBALANCE_USER`, `CMD_FORECAST`), decoupling the Go operational node from the Python math engine.

**External Systems:** Stripe API (Sandbox) for simulating user deposits and cashouts (Paper Trading).

**Frontend:** HTML, Bootstrap, Vanilla JS (Fetch API), Plotly.js (for charts).

**Infrastructure:** Docker Compose (local dev), Digital Ocean VPC (production).

---

## 3. Architectural Principles & Decision Log

### 3.1 Separation of Concerns: Go vs. Python

The boundary between the two nodes is strict and intentional:

| Responsibility | Owner | Reasoning |
|---|---|---|
| Database schema (DDL) | Go | Go owns the DB; Python is a tenant |
| Business policy config (risk/horizon tables, thresholds) | Go | Policy can change without math changing |
| Asset universe & ticker lists | Go config file | Business decision, not a math constant |
| HRP algorithm & weight computation | Python | Pure math, no user context needed |
| Cash-first rebalancing rule | Python | Mathematically expressible on weight vectors |
| Rebalancing delta threshold filter | Python | Math operation, parameter passed from Go |
| Share count calculation (`shares = weight × value / price`) | Go | Requires live price data Go already holds |
| Buy/sell execution (writing Portfolio rows) | Go | Requires user context Python never sees |
| Data ingestion (yfinance → DB) | Python | Python owns the yfinance dependency |

### 3.2 Closing Prices vs. Real-Time Prices

A deliberate decision was made **not** to fetch real-time prices during rebalancing. This platform implements passive, long-term, monthly rebalancing. The difference between yesterday's closing price and today's delayed market price (yfinance is 15 minutes delayed by regulation, not truly real-time) is economically irrelevant on a 30-day rebalancing cycle.

Real-time pricing would add: a new external API dependency on the rebalance critical path, rate limit risk, and a new failure mode. For passive ETF investing it adds zero value to the user's outcome.

Instead, Go performs a **staleness check** before any rebalance executes: if the most recent price in `daily_market_data` for any ticker is older than 2 trading days, the rebalance aborts and an alert is logged. This catches sync failures without adding live-fetch complexity.

### 3.3 Local DailyMarketData vs. Live yfinance Fetching

All ETF price data is persisted in the local PostgreSQL `daily_market_data` table via the daily `/sync` pipeline. Reasons:

1. **Frontend Performance:** The Plotly.js dashboard renders portfolio evolution charts on every login. Serving pre-synced data from the local DB enables instant, reliable chart rendering at zero external cost.
2. **Microservice Decoupling:** Go resolves share counts independently from `daily_market_data` without ever calling Python for price data.
3. **Fault Tolerance:** If yfinance is unavailable when the monthly rebalance fires, locally synced prices ensure the cycle completes successfully.

### 3.4 Model Portfolios: Pre-Computed and Persisted

Python computes all 15 model portfolio buckets daily (every combination of Risk 1–5 × Horizon short/medium/long) and writes the results to the `model_portfolios` table. On rebalance day, Go reads the latest pre-computed bucket — it does not call Python at all for weights. This provides fault tolerance: if Python was unavailable today, Go uses yesterday's bucket.

### 3.5 Go Owns the Database Schema

Go initializes and migrates all tables via GORM AutoMigrate, including tables that Python writes to (`model_portfolios`, `forecast_results`). This is correct: Go is the DB owner, Python is a tenant. Python never runs DDL — only DML (INSERT/UPDATE). If a table doesn't exist when Python tries to write, it crashes with a clear error, making deployment order problems immediately visible rather than silently failing.

### 3.6 Anonymous Portfolio Protocol (Rebalancing Delta)

When Go asks Python to compute rebalancing adjustments, it sends only anonymized weight vectors — no user IDs, emails, balances, or PII of any kind. Go derives the bucket key from the user's risk/horizon profile before sending. Python receives:

```json
{
  "request_id": "abc-123",
  "current_allocation": { "SPY": 0.38, "USD": 0.12, "BND": 0.27, "GLD": 0.08, "VNQ": 0.05, "QQQ": 0.10 },
  "target_weights":     { "SPY": 0.42, "BND": 0.30, "GLD": 0.10, "QQQ": 0.18 },
  "threshold": 0.02,
  "cash_first": true
}
```

Python returns adjusted target weights and a list of skipped tickers. Go then handles the share math and DB writes.

---

## 4. Database Schema (GORM Models — Go owns all DDL)

```go
// Investor account
type User struct {
    ID                  uint      `gorm:"primaryKey"`
    Email               string    `gorm:"unique;not null"`
    Password            string    `gorm:"not null"` // bcrypt hashed
    IsEmailVerified     bool      `gorm:"default:false"`
    TwoFactorSecret     string    // AES-256-GCM encrypted TOTP secret
    IsTwoFactorEnable   bool      `gorm:"default:false"`
    InvestmentHorizon   int       // in years (mapped to short/medium/long in Go config)
    RiskTolerance       int       // 1 (min) to 5 (max)
    FailedLoginAttempts int       `gorm:"default:0"`
    LockoutUntil        time.Time `gorm:"index"`
    CreatedAt           time.Time
    UpdatedAt           time.Time

    Wallet       Wallet
    Sessions     []Session
    ActionTokens []ActionToken
}

// Tracks user login attempts to prevent brute-force attacks via progressive lockouts
type LoginAttempt struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	IsSuccess bool      `gorm:"not null"`
	IPAddress string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null;index"`
}

// Long-lived refresh tokens (multi-device support)
type Session struct {
    ID           uint      `gorm:"primaryKey"`
    UserID       uint      `gorm:"not null;index"`
    FamilyID     string    `gorm:"index"` // groups tokens from the same login family
    RefreshToken string    `gorm:"unique;not null"`
    IsUsed       bool      `gorm:"default:false"` // reuse detection flag
    ClientIP     string
    UserAgent    string
    ExpiresAt    time.Time `gorm:"not null"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// Short-lived single-use tokens (email verification, password reset)
type ActionToken struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"not null;index"`
    Token     string    `gorm:"unique;not null"`
    Type      string    `gorm:"not null"` // "verify_email" | "reset_password"
    ExpiresAt time.Time `gorm:"not null"`
    CreatedAt time.Time
}

// Uninvested money — staging area between bank and portfolio
type Wallet struct {
    ID        uint      `gorm:"primaryKey"`
    UserId    uint      `gorm:"unique;not null"`
    Balance   float64   `gorm:"not null;default:0.0"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Tracks money moving in/out for the contribution chart
type Transaction struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"not null;index"`
    Type      string    `gorm:"not null"` // "INVEST" | "SELL"
    Amount    float64   `gorm:"not null"` // always positive
    CreatedAt time.Time
}

// Groups all holdings from one optimization run
type InvestmentRound struct {
    ID         uint        `gorm:"primaryKey"`
    UserID     uint        `gorm:"not null;index"`
    TotalValue float64     `gorm:"not null"`
    IsActive   bool        `gorm:"not null;default:true"` // false after a newer round replaces it
    CreatedAt  time.Time
    Portfolios []Portfolio
}

// Single holding within an investment round
// Ticker: ETF symbol (e.g. SPY, QQQ, BND) or "USD" for uninvested cash
type Holding struct {
    ID              uint      `gorm:"primaryKey"`
    UserID          uint      `gorm:"not null;index"`
    RoundID         uint      `gorm:"not null;index"`
    Ticker          string    `gorm:"not null"`
    Weight          float64   `gorm:"not null"` // HRP weight e.g. 0.40, or 1.0 for USD
    Shares          float64   `gorm:"not null"` // number of shares, or dollar amount for USD
    PurchasePrice   float64   `gorm:"not null"` // price per share at purchase; 1.0 for USD
    AllocatedAmount float64   `gorm:"not null"` // total dollars in this holding
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Daily closing prices for each ETF — written by Python (sync), read by Go (rebalance) and Python (HRP)
type DailyMarketData struct {
    ID         uint      `gorm:"primaryKey"`
    Ticker     string    `gorm:"not null;uniqueIndex:idx_ticker_date"`
    Date       time.Time `gorm:"not null;uniqueIndex:idx_ticker_date"`
    ClosePrice float64   `gorm:"not null"` // adjusted closing price
    CreatedAt  time.Time
}

// Pre-computed HRP bucket weights — written daily by Python, read by Go on rebalance day
// Go owns this schema. Python never runs DDL, only INSERTs.
type ModelPortfolio struct {
    ID         uint      `gorm:"primaryKey"`
    BucketKey  string    `gorm:"not null;index"` // e.g. "risk_3_horizon_medium"
    Weights    string    `gorm:"not null"`        // JSON: {"SPY": 0.42, "BND": 0.30, ...}
    ComputedAt time.Time `gorm:"not null"`
    CreatedAt  time.Time
}

// Async Monte Carlo forecast results — written by Python, polled by Go
type ForecastResult struct {
    ID         uint      `gorm:"primaryKey"`
    TaskID     string    `gorm:"unique;not null;index"` // UUID issued by Go
    Status     string    `gorm:"not null;default:'pending'"` // "pending" | "complete" | "error"
    Payload    string    // JSON with percentile arrays, nil until complete
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// Tracks fiat money moving in and out of the platform via Stripe or paper trading
type Funding struct {
	ID              uint    `gorm:"primaryKey"`
	UserID          uint    `gorm:"not null;index"`
	Type            string  `gorm:"not null"` // "DEPOSIT", "WITHDRAWAL"
	Amount          float64 `gorm:"not null"`
	StripePaymentID string  `gorm:"index"`    // external reference ID
	Status          string  `gorm:"not null"` // "COMPLETED", "PENDING", "FAILED"
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

---

## 5. Go Configuration (Business Policy — Not Python's Concern)

All investment policy parameters live in a Go config struct loaded from a YAML file or environment variables. Python receives these as inputs in RabbitMQ messages — it never hardcodes them.

```go
type InvestmentConfig struct {
    // Asset universe passed to Python for sync and HRP
    EquityTickers []string // ["VTI", "VOO", "QQQ", "VEA", "VWO", "VNQ", "XLF", "XLV", "XLE", "XLK"]
    BondTickers   []string // ["BND", "TLT", "LQD", "HYG", "BNDX"]

    // Macro allocation table: risk level → base equity ratio
    BaseEquityAllocation map[int]float64 // {1: 0.20, 2: 0.40, 3: 0.60, 4: 0.80, 5: 0.90}

    // Horizon multipliers applied to equity ratio
    HorizonMultipliers map[string]float64 // {"short": 0.70, "medium": 1.00, "long": 1.10}

    // Horizon classification thresholds (years)
    HorizonShortMax  int // 3
    HorizonMediumMax int // 7
    // > HorizonMediumMax = "long"

    // Algorithm tuning
    MaxEquityCap     float64 // 0.95
    TopNEquities     int     // 6  (Sharpe-ranked selection before HRP)
    WeightThreshold  float64 // 0.02 (minimum allocation; crumbs redistributed proportionally)

    // Rebalancing parameters
    RebalanceDeltaThreshold float64 // 0.02 (ignore deltas smaller than this)
    CashFirstEnabled        bool    // true
    PriceStalenessDays      int     // 2 (abort rebalance if prices older than this)
}
```

---

## 6. Data & Workflow Architecture

### 6.1 Daily Background Pipeline (Go cron → RabbitMQ → Python)

```
[Go cron: daily at market close]
  │
  ├─ Publishes CMD_SYNC to RabbitMQ
  │     Python consumer:
  │       → Fetches 5y of closing prices via yfinance
  │       → Upserts into daily_market_data (ON CONFLICT DO UPDATE)
  │       → Acknowledges message
  │
  └─ Publishes CMD_GENERATE to RabbitMQ (after sync completes)
        Python consumer receives:
          {
            "equity_tickers": [...],   ← from Go config
            "bond_tickers": [...],     ← from Go config
            "macro_allocations": {...},
            "horizon_multipliers": {...},
            "max_equity_cap": 0.95,
            "top_n_equities": 6,
            "weight_threshold": 0.02
          }
        Python:
          → Reads daily_market_data from shared DB
          → Runs HRP separately on equity universe and bond universe
          → Applies macro allocation table to blend equity/bond HRP weights
          → Applies weight threshold cleanup (crumbs redistributed proportionally)
          → Writes one row per bucket_key to model_portfolios table
          → 15 rows total (risk 1–5 × short/medium/long)
```

### 6.2 Monthly Rebalance Pipeline (Go cron → Python → Go)

```
[Go cron: every 30 days]
  │
  ├─ STALENESS CHECK
  │     Go queries: SELECT MAX(date) FROM daily_market_data GROUP BY ticker
  │     If any ticker's latest price > 2 trading days old → ABORT + log alert
  │
  ├─ Go reads model_portfolios (latest row per bucket_key)
  │     No Python call needed — buckets already pre-computed and persisted
  │
  ├─ Go queries all users with IsActive InvestmentRound
  │
  └─ For each user:
        Go derives bucket_key: risk_{n}_horizon_{short|medium|long}
        Go computes current_allocation as weight map from Portfolio rows
          (includes "USD" for any uninvested cash)
        Go publishes CMD_REBALANCE_USER to RabbitMQ:
          {
            "request_id": "<uuid>",         ← no user ID, no PII
            "current_allocation": {...},
            "target_weights": {...},        ← Go already resolved from model_portfolios
            "threshold": 0.02,
            "cash_first": true
          }

        Python consumer:
          1. Cash-first rule: exhaust USD allocation before selling any ETF
          2. Threshold filter: skip tickers where |current - target| < threshold
          3. Returns: { "request_id": "...", "adjusted_targets": {...}, "skipped": [...] }

        Go receives adjusted_targets:
          → Reads latest ClosePrice per ticker from daily_market_data
          → Computes: shares = (adjusted_weight × total_value) / close_price
          → Writes new Portfolio rows in a DB transaction
          → Marks old InvestmentRound IsActive=false
          → Creates new InvestmentRound row
```

### 6.3 Asynchronous Forecast Pipeline (Task ID + Polling)

```
[User requests forecast on frontend]
  │
  Go POST /forecast:
    → Generates UUID (task_id)
    → Inserts ForecastResult row: {task_id, status: "pending"}
    → Publishes CMD_FORECAST to RabbitMQ:
        {
          "task_id": "<uuid>",
          "weights": {...},
          "initial_investment": 10000,
          "monthly_contribution": 500,
          "years": 20
        }
    → Returns HTTP 202 Accepted: { "task_id": "<uuid>" }

  Python consumer:
    → Reads daily_market_data to compute portfolio mean return & volatility
    → Runs Monte Carlo (10,000 scenarios, Geometric Brownian Motion)
    → Writes percentiles to ForecastResult: {status: "complete", payload: {...}}

  Frontend polling GET /forecast/status/:task_id (every 2s via setInterval):
    → Go reads ForecastResult by task_id
    → Returns {"status": "pending"} or final JSON payload
    → Frontend renders Plotly.js "Cone of Uncertainty" on completion
```

---

## 7. Development Roadmap

### Phase 1: Foundation

- [x] Set up Docker Compose with Go, Python, RabbitMQ, and PostgreSQL services
- [x] Configure RabbitMQ exchanges and queues (`cmd_queue`, `result_queue`)
- [x] Connect Go to PostgreSQL using GORM
- [x] Define database models and run AutoMigrate (including ModelPortfolio and ForecastResult)
- [x] Create dummy user seeding for initial testing
- [x] Create basic POST /deposit endpoint to simulate adding funds

### Phase 2: Identity Management, Authentication & Security

This phase implements a complete, enterprise-grade IAM flow using discrete `Session` and `ActionToken` tables for robust session management, multi-device support, and security lifecycle.

**Email Delivery Architecture (Dependency Injection):**
An `EmailSender` interface decouples email logic from business logic.
- Development: Native Go `net/smtp` with Gmail App Password
- Production: SendGrid API integration

- [x] Install `golang.org/x/crypto/bcrypt` and `github.com/golang-jwt/jwt/v5`
- [x] Install `github.com/pquerna/otp` for TOTP (Google Authenticator) 2FA
- [x] Install `github.com/robfig/cron/v3` for background task scheduling
- [x] Implement `EmailSender` interface (SMTP) with Goroutines to prevent blocking
- [x] **POST /register:** Hash password with bcrypt, create User (IsEmailVerified=false), create ActionToken (type: "verify_email"), send activation email
- [x] **GET /verify-email:** Validate token from ActionToken, set IsEmailVerified=true, delete token
- [x] **POST /login (Step 1):** Verify credentials, check IsEmailVerified, return "2fa_required" if 2FA enabled. Dummy bcrypt hash prevents user enumeration via timing attacks
- [x] **GET /2fa/setup:** Generate TOTP secret, encrypt with AES-256-GCM (zero-knowledge DB storage), return plaintext secret + Base64 QR code for device pairing
- [x] **POST /2fa/enable:** Validate initial 6-digit TOTP code, permanently enable 2FA
- [x] **POST /verify-2fa (Step 2):** Validate TOTP against decrypted secret, proceed to session token generation
- [x] **Token Strategy (Multi-Device):** Short-lived Access Token (JWT, 10min) + secure random Refresh Token stored in Session table (7 days)
- [x] **POST /refresh-token:** Refresh Token Rotation with Reuse Detection. Stolen token → invalidate entire FamilyID. Optimistic Concurrency Control (OCC) via UpdatedAt prevents race conditions from concurrent browser tabs
- [x] **POST /logout:** Delete Refresh Token row from Session table
- [x] **POST /forgot-password:** Create ActionToken (type: "reset_password", 15min), send recovery link. Constant-time delay + random noise prevents email enumeration via timing attacks
- [x] **POST /reset-password:** DB transaction: hash new password + invalidate all sessions + delete single-use recovery token
- [x] **Data Lifecycle CRON:** Nightly job at 03:00 AM purges expired Session and ActionToken records
- [x] Gin middleware to protect routes (require Bearer Access Token)
- [x] Refactor GET /user and POST /deposit to use userID from JWT context
- [ ] **Cloudflare Turnstile:** Anti-bot challenge on /login, /register, /forgot-password. Server-side verification before bcrypt to save CPU
- [x] **IP-Based Rate Limiting:** Global Gin middleware using Token Bucket algorithm
- [x] **Email-Based Account Lockout:** 5-attempt threshold → 15-minute lockout via LockoutUntil, immune to IP rotation
- [x] **Atomic Deposit Logic:** Refactor POST /deposit to use `gorm.Expr("balance + ?", amount)` for atomic DB-level increments (prevents Lost Update race conditions)
- [x] **Decisional Node Isolation:** Move all Python-calling routes inside the `protected` middleware group

### Phase 3: The Python Math Engine & Data Persistence

**ETF Universe (defined in Go config, passed to Python in messages):**
- Equities: VTI, VOO, QQQ, VTV, VUG, IWM, VEA, VWO, VNQ, VNQI, XLF, XLV, XLE, XLK
- Bonds: BND, TLT, LQD, HYG, BNDX

**Investment Strategy:** User clicks Invest → funds held as USD ticker → daily HRP runs produce pre-computed buckets → every 30 days Go rebalances all users using latest pre-computed buckets.

**Python's Contract:** Python is a pure function of its inputs. It reads `daily_market_data`, writes `model_portfolios` and `forecast_results`. It never touches user tables. All policy parameters arrive in the RabbitMQ message payload — nothing is hardcoded in Python source.

- [x] Setup yfinance in Python to fetch closing prices for all ETFs
- [x] Upsert prices into PostgreSQL using ON CONFLICT (ticker, date) DO UPDATE
- [x] Expose POST /sync in FastAPI as a standalone data ingestion endpoint
- [x] Implement HRP (Hierarchical Risk Parity) algorithm using scipy, pandas
- [x] Refactor /sync to accept ticker lists from the incoming RabbitMQ message payload (not hardcoded in Python)
- [x] Refactor /generate-models to accept all policy parameters from the incoming message payload:
  - macro_allocations table
  - horizon_multipliers table
  - max_equity_cap, top_n_equities, weight_threshold
- [x] After computing all 15 bucket weights, persist each to `model_portfolios` table (INSERT, not just HTTP return)
- [x] Implement `CMD_REBALANCE_USER` consumer in Python:
  - Receives: current_allocation, target_weights, threshold, cash_first flag
  - Applies cash-first rule: deplete USD allocation before selling any ETF
  - Applies threshold filter: skip tickers where |current - target| < threshold
  - Returns: adjusted_targets + skipped list
  - No DB reads during this operation — pure math on inputs
- [x] Implement RabbitMQ consumer in Python using `pika` to handle CMD_SYNC, CMD_GENERATE, CMD_REBALANCE_USER, CMD_FORECAST
- [x] Implement Monte Carlo Simulation (10,000 scenarios, Geometric Brownian Motion):
  - Reads daily_market_data to compute portfolio mean return and volatility
  - Writes percentile results to `forecast_results` table keyed by task_id

### Phase 4: Orchestration (Go + Python) & Stripe Integration

- [x] Implement Go config loader for InvestmentConfig (YAML or env vars) — this is the single source of truth for all policy parameters
- [x] Implement RabbitMQ Producer in Go to dispatch CMD_SYNC and CMD_GENERATE daily via cron
- [x] Integrate Stripe Sandbox API for POST /deposit (bank → wallet) and POST /cashout (wallet → bank)
- [x] **POST /invest:** Move wallet balance to portfolio as USD ticker, create InvestmentRound
- [x] **POST /rebalance (cron, every 30 days):**
  1. Staleness check: abort if any ticker's latest price in `daily_market_data` is older than `PriceStalenessDays` trading days
  2. Read latest pre-computed weights from `model_portfolios` for all 15 bucket keys
  3. Query all users with active InvestmentRound
  4. For each user:
     - Derive bucket_key from RiskTolerance + InvestmentHorizon (using Go config thresholds)
     - Compute current_allocation weight map from Portfolio rows (including USD)
     - Publish CMD_REBALANCE_USER to RabbitMQ (anonymous — no PII)
     - Receive adjusted_targets from Python reply queue
     - Read latest ClosePrice per ticker from daily_market_data
     - Compute `shares = (adjusted_weight × total_value) / close_price`
     - DB transaction: write new Portfolio rows + mark old InvestmentRound IsActive=false
- [x] **POST /forecast:** Accept user parameters, generate UUID (task_id), insert pending ForecastResult row, publish CMD_FORECAST to RabbitMQ, return HTTP 202 with task_id
- [x] **GET /forecast/status/:task_id:** Poll ForecastResult table, return pending status or final payload

### Phase 5: Frontend Dashboard

- [ ] Serve static HTML/JS from the Go router
- [ ] Build Login/Register UI (including 2FA QR code display and verification step)
- [ ] Build Dashboard UI: current balance, Stripe deposit form, portfolio allocation
- [ ] Fetch GET /user/portfolio and render Plotly.js Pie Chart of asset allocation
- [ ] **Frontend Security & Session Management:** Implement token storage strategy. Options: `localStorage` (simpler) vs `HttpOnly` Cookies (XSS-resistant). Decision must be documented
- [ ] **Frontend Fetch Interceptor (`fetchWithAuth`):** Global wrapper that intercepts 401 responses, silently calls POST /refresh-token, retries original request
- [ ] **Refresh Queue (Race Condition Mitigation):** Global `isRefreshing` semaphore + Promise queue. Only one refresh request fires when multiple calls expire simultaneously — others wait for the new token
- [ ] **XSS Awareness:** All dynamic data rendered via `textContent`, never `innerHTML`
- [ ] **Frontend Logout:** POST /logout → clear local state → redirect to login
- [ ] **Anti-Bot UI:** Cloudflare Turnstile widget on auth forms, `cf-turnstile-response` token in Fetch payloads
- [ ] **Monte Carlo UI:** POST /forecast → capture task_id → setInterval polling on GET /forecast/status/:task_id → Plotly.js "Cone of Uncertainty" showing 5th/50th/95th percentiles

### Phase 6: Cloud Deployment (Digital Ocean)

- [ ] Provision 2 Ubuntu Droplets (Droplet 1: PostgreSQL + Go; Droplet 2: Python)
- [ ] Configure internal VPC networking (Python node is never exposed to the public internet)
- [ ] Configure UFW firewall: block all except SSH and Go's web port
- [ ] Deploy and verify daily cron pipeline (sync → generate-models) in production
- [ ] Verify rebalance staleness check operates correctly across timezone boundaries
- [ ] Note on serverless option: scipy + pandas + numpy (~150MB) likely exceed DO Function limits and create cold start problems. Dedicated Droplet in private VPC preferred for both security and reliability

### Phase 7: Blockchain Audit Log (Optional / Bonus)

- [ ] Create AuditLog table in PostgreSQL
- [ ] Each rebalance event hashes: previous block hash + portfolio snapshot + Python script checksum (SHA-256)
- [ ] Build frontend "Block Explorer" to prove algorithm and history integrity to the user

### Phase 8: Enterprise Architecture Evolution (Optional / Future)

- [ ] **Go — Modular Monolith with Vertical Slices:** Transition from layered (`handlers/`, `services/`, `repositories/`) to business domain slices (`features/auth/`, `features/ledger/`, `features/portfolio/`) for maximum cohesion and independent team workflows
- [ ] **Notifications Microservice:** Extend RabbitMQ to publish domain events (e.g., `user.registered`, `rebalance.completed`). Standalone consumer handles all email delivery asynchronously with zero data loss on crash
- [ ] **Redis Caching (Cache-Aside Pattern):** Cache the 15 model portfolio weight sets in Redis RAM. Go reads from sub-millisecond cache on dashboard load instead of querying `model_portfolios` table. Cache invalidated and refreshed after each daily CMD_GENERATE run