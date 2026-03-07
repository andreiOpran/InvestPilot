Robo-Advisory Platform: Master Development Plan

1. Project Overview

This project is an Automated Wealth Management (Robo-Advisory) platform using a "Set & Forget" passive investment strategy. It operates in a simulated "Paper Trading" environment. The system accepts user funds, automatically calculates an optimal portfolio using the Markowitz Efficient Frontier, and executes simulated trades.

2. Tech Stack & Architecture

**Operational Node (Backend API):** Golang (Gin Framework, GORM). Handles users, wallets, auth (including 2FA), payments, and frontend serving.

**Decisional Node (Math Engine):** Python 3 (FastAPI, Uvicorn, Pandas, SciPy, yfinance). A private microservice that computes the Markowitz algorithm. Auto-generates Swagger UI at /docs.

**Database:** PostgreSQL. Stores users, balances, asset holdings, and historical market data.

**External Systems:** Stripe API (Sandbox) for simulating user deposits (Paper Trading).

**Frontend:** HTML, Bootstrap, Vanilla JS (Fetch API), Plotly.js (for charts).

**Infrastructure:** Docker Compose (local dev), Digital Ocean VPC (production).

3. Database Schema (GORM Models)

```go
// investor account
type User struct {
    ID                uint      `gorm:"primaryKey"`
    Email             string    `gorm:"unique;not null"`
    Password          string    `gorm:"not null"` // bcrypt hashed
    TwoFactorSecret   string    // secret for Google Authenticator/TOTP (Phase 5)
    IsTwoFactorEnable bool      `gorm:"default:false"`
    InvestmentHorizon int       // in years
    RiskTolerance     int       // 1 (min) to 5 (max)
    CreatedAt         time.Time
    UpdatedAt         time.Time
    Wallet            Wallet
}

// uninvested money, staging area between bank and portfolio
type Wallet struct {
    ID        uint      `gorm:"primaryKey"`
    UserId    uint      `gorm:"unique;not null"`
    Balance   float64   `gorm:"not null;default:0.0"`
    CreatedAt time.Time
    UpdatedAt time.Time
}

// tracks money moving in/out of the portfolio for the contribution chart
type Transaction struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"not null;index"`
    Type      string    `gorm:"not null"` // "invest" or "sell"
    Amount    float64   `gorm:"not null"` // always positive
    CreatedAt time.Time
}

// groups all holdings belonging to one optimization run
type InvestmentRound struct {
    ID         uint        `gorm:"primaryKey"`
    UserID     uint        `gorm:"not null;index"`
    TotalValue float64     `gorm:"not null"`
    IsActive   bool        `gorm:"not null;default:true"` // false after a newer round replaces it
    CreatedAt  time.Time
    Portfolios []Portfolio
}

// single holding within an investment round
// Ticker can be ETF (SPY, QQQ, BND, GLD, VNQ) or "USD" for uninvested cash
type Portfolio struct {
    ID              uint      `gorm:"primaryKey"`
    UserID          uint      `gorm:"not null;index"`
    RoundID         uint      `gorm:"not null;index"`
    Ticker          string    `gorm:"not null"`
    Weight          float64   `gorm:"not null"` // markowitz weight e.g. 0.40, or 1.0 for USD
    Shares          float64   `gorm:"not null"` // number of shares, or dollar amount for USD
    PurchasePrice   float64   `gorm:"not null"` // price per share at purchase, 1.0 for USD
    AllocatedAmount float64   `gorm:"not null"` // total dollars in this holding
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// daily closing prices for each ETF, used as Markowitz input
type HistoricalMarketData struct {
    ID         uint      `gorm:"primaryKey"`
    Ticker     string    `gorm:"not null;uniqueIndex:idx_ticker_date"`
    Date       time.Time `gorm:"not null;uniqueIndex:idx_ticker_date"`
    ClosePrice float64   `gorm:"not null"` // adjusted closing price
    CreatedAt  time.Time
}
```

4. Development Roadmap

**Phase 1: Foundation**

- [x] Set up Docker Compose with Go, Python, and PostgreSQL in a private VPC.
- [x] Connect Go to PostgreSQL using GORM.
- [x] Define database models and run AutoMigrate.
- [x] Create dummy user seeding for initial testing.
- [x] Create basic POST /deposit endpoint to simulate adding funds.

**Phase 2: Authentication & Security with 2FA**

- [x] Install golang.org/x/crypto/bcrypt and github.com/golang-jwt/jwt/v5. (github.com/pquerna/otp still needed for 2FA)
- [x] Implement POST /register: Hash password with bcrypt, create User, and attach empty Wallet. (2FA secret generation pending)
- [x] Implement POST /login: Verify bcrypt hash, generate and return JWT token. (temp token + 2FA verification step pending)
- [ ] Implement POST /verify-2fa: Validate the 6-digit TOTP code against the user's secret. Generate and return the final JWT token.
- [x] Create a Gin Middleware to protect routes (require Bearer Token).
- [x] Refactor GET /user and POST /deposit to use userID from JWT context instead of hardcoded dummy email.

**Phase 3: The Python Math Engine & Data Persistence**

ETF universe: SPY (US equities), QQQ (US tech), BND (bonds), GLD (gold), VNQ (real estate)

Investment strategy: user clicks Invest -> funds held as USD ticker until scheduled rebalance -> every 30 days optimizer runs for all users -> old round marked IsActive=false -> new Portfolio rows created with real ETF weights.

**Architectural Justification: Local HistoricalMarketData vs. Live yfinance Fetching**

A deliberate design decision was made to persist all ETF price data in the local PostgreSQL `HistoricalMarketData` table via the `/sync` ingestion pipeline, rather than fetching prices live from Yahoo Finance on demand. Three enterprise-grade concerns motivate this:

1. **Frontend Performance & Rate Limiting (Charts):** The Plotly.js dashboard must render a user's portfolio evolution over time on every login. Fetching months of price history from Yahoo Finance on each page load would immediately exhaust yfinance's rate limits under any real user load. Serving pre-synced data from the local DB enables instant, reliable chart rendering at zero external cost.

2. **Microservice Decoupling (Go vs. Python):** The Python node is a pure mathematical engine — it outputs portfolio weights (dimensionless percentages). The Go node is responsible for trade execution and must independently resolve the *actual current ETF prices* to compute exact share counts (`shares = (weight * total_value) / latest_close_price`). By reading prices from PostgreSQL directly, Go never needs to call Python for price data, keeping the two nodes cleanly decoupled and independently deployable.

3. **Fault Tolerance & Resilience:** If the external yfinance API is unavailable exactly when the monthly `/rebalance` cron job fires, a live-fetch dependency would crash the entire rebalance cycle. By relying on locally synced closing prices, Go can always execute the rebalance successfully using the most recent data available in the DB, ensuring the platform remains operational regardless of third-party outages.

- [x] Setup yfinance in Python to fetch 2 years of daily closing prices for all 5 ETFs.
- [x] Store/Update fetched prices in PostgreSQL using ON CONFLICT - 502 trading days x 5 tickers = 2510 rows.
- [x] Expose POST /sync in FastAPI as a data ingestion pipeline: triggers yfinance fetching and upserts all prices into PostgreSQL, independently of optimization.
- [ ] Write the Markowitz Optimization algorithm using scipy.optimize, reading historical data from the DB.
- [ ] **Architectural decision — Model Portfolios:** Instead of optimizing per-user, pre-compute a fixed set of "Model Portfolios" covering every combination of Risk Level (1–5) and Investment Horizon (Short: 1–3 yrs, Medium: 4–7 yrs, Long: 8+ yrs) — 15 buckets total. This is computationally O(K) where K=15, rather than O(N) per user.
- [ ] Expose `POST /generate-models` in FastAPI (replaces the per-user `/optimize`). It reads historical prices from the DB, runs Markowitz for each of the 15 risk/horizon buckets, and returns a JSON dictionary mapping each bucket key to its optimal ETF weights (e.g., `{"risk_4_horizon_long": {"SPY": 0.80, "BND": 0.20}, "risk_2_horizon_short": {"BND": 0.60, "GLD": 0.25, "SPY": 0.15}, ...}`).

**Phase 4: Orchestration (Go + Python) & Stripe Integration**

- [ ] Integrate Stripe Sandbox API in Go for POST /deposit (bank -> wallet) and POST /cashout (wallet -> bank).
- [ ] Implement POST /invest in Go: moves wallet balance to portfolio as USD ticker, creates InvestmentRound.
- [ ] Implement POST /rebalance in Go (cron, every 30 days) using the Model Portfolios approach:
  1. Go makes a **single** call to Python's `POST /generate-models`, receiving the full dictionary of 15 pre-computed Model Portfolios. This reduces the Python node's computational load from O(N users) to O(K=15 models).
  2. Go queries PostgreSQL for all users with an active InvestmentRound.
  3. For each user, Go derives the bucket key from their `RiskTolerance` (1–5) and `InvestmentHorizon` (mapped to short/medium/long), then looks up the matching weights in the received dictionary — no further Python calls needed.
  4. Go calculates exact share counts using: `shares = (weight * total_value) / latest_close_price` where `latest_close_price` is read from the `HistoricalMarketData` table.
  5. New `Portfolio` rows are saved and the old `InvestmentRound` is marked `IsActive=false`.

**Phase 5: Frontend Dashboard**

- [ ] Serve static HTML/JS from the Go router.
- [ ] Build Login/Register UI (including 2FA QR code display and verification step).
- [ ] Build Dashboard UI: Show current balance and Stripe deposit form.
- [ ] Fetch GET /user/portfolio and use Plotly.js to render a Pie Chart of the user's asset allocation.

**Phase 6: Cloud Deployment (Digital Ocean)**

- [ ] Provision 2 Ubuntu Droplets.
- [ ] Configure internal VPC networking.
- [ ] Configure UFW firewall (block everything except SSH and Go's web ports).
- [ ] Deploy PostgreSQL and Go on Droplet 1, Python on Droplet 2.
- [ ] (Optional) Replace Python Droplet with a DigitalOcean Function (serverless). Python only runs during /sync and /rebalance — idle 99% of the time, so a serverless function eliminates the cost of a permanent server. Go would invoke the function URL instead of the internal VPC address. Note: keeping Python as a dedicated Droplet inside the VPC is the preferred security choice — the decisional node is never exposed to the public internet and is only reachable by Go over the private network. The serverless option trades that security boundary for cost efficiency. Additional concern: scipy, pandas, and numpy are large packages (~150MB combined) which may exceed serverless function storage limits and significantly increase cold start times, making a dedicated Droplet more practical.

**Phase 7: Blockchain Audit Log** (Optional/Bonus)

- [ ] Create an AuditLog table in PostgreSQL.
- [ ] Every transaction hashes previous block + current data + Python script checksum (SHA-256).
- [ ] Build a frontend "Block Explorer" to prove algorithm and history integrity to the user.
