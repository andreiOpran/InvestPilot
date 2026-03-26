Robo-Advisory Platform: Master Development Plan

1. Project Overview

This project is an Automated Wealth Management (Robo-Advisory) platform using a "Set & Forget" passive investment strategy. It operates in a simulated "Paper Trading" environment. The system accepts user funds, automatically calculates an optimal portfolio using the Markowitz Efficient Frontier, and executes simulated trades.

2. Tech Stack & Architecture

**Operational Node (Backend API):** Golang (Gin Framework, GORM). Handles users, wallets, auth (including 2FA), payments, and frontend serving.

**Decisional Node (Math Engine):** Python 3 (FastAPI, Uvicorn, Pandas, SciPy, yfinance). A private microservice that computes the Markowitz algorithm. Auto-generates Swagger UI at /docs.

**Database:** PostgreSQL. Stores users, balances, asset holdings, and historical market data.

**Message Broker:** RabbitMQ (AMQP). Orchestrates asynchronous tasks (e.g., Sync and Rebalance), decoupling the Go operational node from the Python math engine.

**External Systems:** Stripe API (Sandbox) for simulating user deposits (Paper Trading).

**Frontend:** HTML, Bootstrap, Vanilla JS (Fetch API), Plotly.js (for charts).

**Infrastructure:** Docker Compose (local dev), Digital Ocean VPC (production).

3. Database Schema (GORM Models)

```go
// investor account
type User struct {
    ID                  uint      `gorm:"primaryKey"`
    Email               string    `gorm:"unique;not null"`
    Password            string    `gorm:"not null"` // bcrypt hashed
    IsEmailVerified     bool      `gorm:"default:false"`
    TwoFactorSecret     string    // secret for Google Authenticator/TOTP
    IsTwoFactorEnable   bool      `gorm:"default:false"`
    InvestmentHorizon   int       // in years
    RiskTolerance       int       // 1 (min) to 5 (max)
    FailedLoginAttempts int       `gorm:"default:0"`
    LockoutUntil        time.Time `gorm:"index"`
    CreatedAt           time.Time
    UpdatedAt           time.Time
    
    // relationships
    Wallet            Wallet
    Sessions          []Session
    ActionTokens      []ActionToken
}

// manages long-lived refresh tokens (allows multi-device logins)
type Session struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;index"`
	FamilyID     string    `gorm:"index"` // makes the relaionship between same user sessions
	RefreshToken string    `gorm:"unique;not null"`
	IsUsed       bool      `gorm:"default:false"` // reuse detection
	ClientIP     string    // optional: logged in IP
	UserAgent    string    // optional: device (Chrome/Mac)
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// manages temporary, short-lived tokens (email verification, password reset)
type ActionToken struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uint      `gorm:"not null;index"`
    Token     string    `gorm:"unique;not null"`
    Type      string    `gorm:"not null"` // "verify_email", "reset_password"
    ExpiresAt time.Time `gorm:"not null"`
    CreatedAt time.Time
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

- [ ] Set up Docker Compose with Go, Python, RabbitMQ, and PostgreSQL in a private VPC. (RabbitMQ is missing now)
- [ ] Configure RabbitMQ exchanges and queues for task distribution (e.g., `task_queue`).
- [x] Connect Go to PostgreSQL using GORM.
- [x] Define database models and run AutoMigrate.
- [x] Create dummy user seeding for initial testing.
- [x] Create basic POST /deposit endpoint to simulate adding funds.

**Phase 2: Identity Management, Authentication & Security**

This phase implements a complete, enterprise-grade Identity and Access Management (IAM) flow, utilizing discrete database tables (`Session`, `ActionToken`) for robust session management, multi-device support, and security lifecycle.

**Email Delivery Architecture (Dependency Injection):**
An `EmailSender` interface will be implemented to decouple the email logic from the business logic. 
- *Development:* Native Go `net/smtp` using a Gmail App Password for rapid local testing.
- *Production (Future-proofing):* SendGrid API integration.

- [x] Install `golang.org/x/crypto/bcrypt` and `github.com/golang-jwt/jwt/v5`. 
- [x] Install `github.com/pquerna/otp` for TOTP (Google Authenticator) 2FA generation and validation.
- [x] Install `github.com/robfig/cron/v3` for automated background maintenance tasks.
- [x] Implement `EmailSender` interface (SMTP strategy) with Goroutines to prevent network latency from blocking API responses.
- [x] **POST /register:** Hash password with bcrypt, create `User` (with `IsEmailVerified=false`). Create a record in `ActionToken` (Type: "verify_email") and send the activation link via email. 
- [x] **GET /verify-email:** Accept token from URL query, find it in `ActionToken`, ensure it's not expired. Set `User.IsEmailVerified=true`, and delete the token row.
- [x] **POST /login (Step 1):** Verify email exists and password matches. Check `IsEmailVerified`. If user has 2FA enabled, return a temporary `status: "2fa_required"` response instead of tokens. (Includes dummy bcrypt hashing to prevent User Enumeration via Timing Attacks).
- [x] **GET /2fa/setup:** Generate TOTP secret. Encrypt the secret using **AES-256-GCM (Encryption at Rest)** for zero-knowledge database storage, while safely returning the plaintext secret and Base64 QR code to the frontend for initial device pairing.
- [x] **POST /2fa/enable:** Validate the initial 6-digit TOTP code against the decrypted `TwoFactorSecret` to permanently enable 2FA on the user's account.
- [x] **POST /verify-2fa (Step 2):** Validate the 6-digit TOTP code against the user's decrypted `TwoFactorSecret`. If successful, proceed to generate session tokens.
- [x] **Token Strategy (Multi-Device):** Generate a Short-Lived Access Token (JWT, expires in 10 mins). Generate a secure random string for the Refresh Token, store it in the `Session` table (expires in 7 days), and return both to the client.
- [x] **POST /refresh-token (Rotation & Reuse Detection):** Implement Refresh Token Rotation. Group tokens by `FamilyID`. If a used (stolen) refresh token is presented, trigger a security alert and invalidate the entire token family to protect the account.
- [x] **POST /refresh-token (Race Condition Mitigation):** Implemented **Optimistic Concurrency Control (OCC)** using the `UpdatedAt` timestamp to elegantly prevent Race Conditions and database anomalies during concurrent token renewal requests from multiple browser tabs.
- [x] **POST /logout:** Accept the current Refresh Token and delete its corresponding row from the `Session` table.
- [x] **POST /forgot-password:** Create an `ActionToken` (Type: "reset_password", expires in 15 mins) and send the recovery link via email. Implemented constant-time delay simulation (with random noise) to prevent Email Enumeration via Timing Attacks.
- [x] **POST /reset-password:** Validate the token from the `ActionToken` table. Wrapped in a strict database transaction to hash the new password, invalidate all existing sessions, and **securely delete the single-use recovery token**.
- [x] **Data Lifecycle Management (CRON):** Implement a nightly background job (running at 03:00 AM) to purge expired `Session` and `ActionToken` records, preventing database bloat and maintaining query performance.
- [ ] **Cloudflare Turnstile Integration:** Implement an anti-bot challenge on `/login`, `/register`, and `/forgot-password`. Verification happens server-side before any `bcrypt` processing to save CPU resources.
- [ ] **IP-Based Rate Limiting:** Implement a global Gin middleware using a "Token Bucket" algorithm to restrict requests per IP, preventing simple Denial of Service (DoS) and brute-force noise.
- [ ] **Email-Based Account Lockout:** Implement a 5-attempt threshold logic. If exceeded, the specific account is locked for 15 minutes (`LockoutUntil`), regardless of the attacker's IP rotation.
- [ ] **Atomic Deposit Logic:** Refactor `POST /deposit` to use database-level atomic increments (`gorm.Expr("balance + ?", amount)`) to prevent "Lost Update" race conditions during concurrent requests.
- [ ] **Decisional Node Isolation:** Move `/simulate-investment` (and any route calling Python) inside the `protected` group to ensure only authenticated users can trigger CPU-intensive Markowitz calculations.
- [x] Create a Gin Middleware to protect routes (require Bearer Access Token).
- [x] Refactor `GET /user` and `POST /deposit` to use `userID` from JWT context.

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
- [ ] Implement a RabbitMQ Consumer in Python using the `pika` library to listen for incoming background jobs.
- [ ] Refactor `sync` and `generate-models` logic to be triggerable via RabbitMQ messages (e.g., `CMD_SYNC`, `CMD_GENERATE`) instead of just HTTP endpoints.

**Phase 4: Orchestration (Go + Python) & Stripe Integration**

- [ ] Implement a RabbitMQ Producer in Go (using `robfig/cron` for scheduling) to dispatch tasks asynchronously to the Decisional Node.
- [ ] Integrate Stripe Sandbox API in Go for POST /deposit (bank -> wallet) and POST /cashout (wallet -> bank).
- [ ] Implement POST /invest in Go: moves wallet balance to portfolio as USD ticker, creates InvestmentRound.
- [ ] Implement POST /rebalance in Go (cron, every 30 days) using the Model Portfolios approach:
  1. Go's cron job publishes a `GENERATE_MODELS` message to RabbitMQ. Python consumes it, runs the HRP optimization for the 15 buckets, and returns the weights via a reply queue (RPC pattern) or saves them directly to the DB.
  2. Go queries PostgreSQL for all users with an active InvestmentRound.
  3. For each user, Go derives the bucket key from their `RiskTolerance` (1–5) and `InvestmentHorizon` (mapped to short/medium/long), then looks up the matching weights in the received dictionary — no further Python calls needed.
  4. Go calculates exact share counts using: `shares = (weight * total_value) / latest_close_price` where `latest_close_price` is read from the `HistoricalMarketData` table.
  5. New `Portfolio` rows are saved and the old `InvestmentRound` is marked `IsActive=false`.

**Phase 5: Frontend Dashboard**

- [ ] Serve static HTML/JS from the Go router.
- [ ] Build Login/Register UI (including 2FA QR code display and verification step).
- [ ] Build Dashboard UI: Show current balance and Stripe deposit form.
- [ ] Fetch GET /user/portfolio and use Plotly.js to render a Pie Chart of the user's asset allocation.
- [ ] **Frontend Security & Session Management:** Design and implement the token storage strategy. Decide between using `localStorage` (Standard SPA approach) or `HttpOnly` Cookies (Advanced XSS mitigation) for securely holding the `access_token` and `refresh_token`.
- [ ] **Frontend Axios/Fetch Interceptor:** Create a global wrapper for all API calls (`fetchWithAuth`). This function must automatically intercept `401 Unauthorized` responses, pause the original request, and silently call `POST /refresh-token` (handling either raw token strings or cookie-based credentials).
- [ ] **Frontend Race Condition Mitigation (The "Refresh Queue"):** Implement a global "semaphore" (`isRefreshing` boolean) and a Promise queue inside the interceptor. If multiple API calls fail simultaneously due to an expired token, only *one* refresh request is sent to the backend, while the others wait in the queue for the new token, ensuring smooth UX and preventing backend 409 Conflicts.
- [ ] **Frontend XSS Awareness:** Ensure all dynamic user data is rendered safely using `textContent` (or framework equivalents) rather than `innerHTML` to prevent Cross-Site Scripting attacks from stealing tokens (especially critical if opting for the `localStorage` strategy).
- [ ] **Frontend Logout Flow:** Bind a logout button to `POST /logout` (sending the refresh token or invalidating the session cookie), followed by clearing the local state and redirecting to the login screen.
- [ ] **Anti-Bot UI:** Integrate the Cloudflare Turnstile widget into authentication forms and handle the `cf-turnstile-response` token in the Fetch API payloads.

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

**Phase 8: Enterprise Architecture Evolution (Optional / Future Enhancements)**

To scale the platform to handle enterprise-level traffic and improve maintainability, the architecture can evolve from a standard layered microservice approach to a highly optimized, event-driven ecosystem:

- [ ] **Core (Go) — Modular Monolith with Vertical Slices:** Transition the Go codebase from a technical Layered Architecture (`handlers/`, `services/`, `repositories/`) to Business Domains/Vertical Slices (e.g., `features/auth/`, `features/ledger/`, `features/portfolio/`). This maximizes cohesion and allows independent teams to work on isolated features without merge conflicts.
- [ ] **Brain (Python) — gRPC Communication:** Replace the current HTTP/JSON REST API between the Go node and the Python node with **gRPC (Protobuf)**. This enforces strictly typed contracts and enables ~10x faster binary data transfer, which is critical when transmitting large matrices of ETF weights and historical prices.
- [ ] **Nervous System Evolution:** Extend the existing RabbitMQ infrastructure to handle other system events beyond math calculations (e.g., publish a `user.registered` event). A standalone `Notifications` consumer will listen to this queue and handle email delivery asynchronously, ensuring zero data loss during crashes.
- [ ] **Persistence — Redis Caching Strategy:** Add **Redis** alongside PostgreSQL. Since the 15 Model Portfolios generated by Python only change upon monthly rebalancing, Go will cache these weights in Redis RAM. On every user login or dashboard refresh, Go reads the weights from the sub-millisecond cache instead of performing expensive network calls to the Python math engine (Cache-Aside pattern).