# Robo-Advisory Platform: Master Development Plan

## 1. Project Overview

This project is an Automated Wealth Management (Robo-Advisory) platform using a "Set & Forget" passive investment strategy. It operates in a simulated "Paper Trading" environment. The system accepts user funds, automatically calculates an optimal portfolio using the Markowitz Efficient Frontier, and executes simulated trades.

## 2. Tech Stack & Architecture

- **Operational Node (Backend API):** Golang (Gin Framework, GORM). Handles users, wallets, auth, payments, and frontend serving.
- **Decisional Node (Math Engine):** Python 3 (Flask, Pandas, SciPy, yfinance). A private microservice that computes the Markowitz algorithm.
- **Database:** PostgreSQL. Stores users, balances, asset holdings, and historical market data.
- **External Systems:** Stripe API (Sandbox) for simulating user deposits (Paper Trading).
- **Frontend:** HTML, Bootstrap, Vanilla JS (Fetch API), Plotly.js (for charts).
- **Infrastructure:** Docker Compose (local dev), Digital Ocean VPC (production).

## 3. Database Schema (GORM Models)

Context for Copilot: All Go code must adhere to these relational models

```
type User struct {
    ID                uint        `gorm:"primaryKey"`
    Email             string      `gorm:"unique;not null"`
    Password          string      `gorm:"not null"` // Must be hashed with bcrypt
    InvestmentHorizon int         `gorm:"default:5"` 
    RiskTolerance     int         `gorm:"default:3"` // Range 1 to 5
    Wallet            Wallet      // 1-to-1 relationship
    Portfolios        []Portfolio // 1-to-many relationship
}

type Wallet struct {
    ID      uint    `gorm:"primaryKey"`
    UserID  uint    // Foreign key to User
    Balance float64 `gorm:"default:0.0"` 
}

type Portfolio struct {
    ID     uint    `gorm:"primaryKey"`
    UserID uint    // Foreign key to User
    Ticker string  `gorm:"not null"` // e.g., "VTI", "BND"
    Shares float64 `gorm:"not null"` // Percentage or exact shares
}

// TODO: Add model for HistoricalMarketData to store yfinance prices
```

# Robo-Advisory Platform: Master Development Plan

## 1. Project Overview

This project is an Automated Wealth Management (Robo-Advisory) platform using a "Set & Forget" passive investment strategy. It operates in a simulated "Paper Trading" environment. The system accepts user funds, automatically calculates an optimal portfolio using the Markowitz Efficient Frontier, and executes simulated trades.

## 2. Tech Stack & Architecture

- **Operational Node (Backend API):** Golang (Gin Framework, GORM). Handles users, wallets, auth, payments, and frontend serving.
- **Decisional Node (Math Engine):** Python 3 (Flask, Pandas, SciPy, yfinance). A private microservice that computes the Markowitz algorithm.
- **Database:** PostgreSQL. Stores users, balances, asset holdings, and historical market data.
- **External Systems:** Stripe API (Sandbox) for simulating user deposits (Paper Trading).
- **Frontend:** HTML, Bootstrap, Vanilla JS (Fetch API), Plotly.js (for charts).
- **Infrastructure:** Docker Compose (local dev), Digital Ocean VPC (production).

## 3. Database Schema (GORM Models)

_Context for Copilot: All Go code must adhere to these relational models._

```go
type User struct {
    ID                uint        `gorm:"primaryKey"`
    Email             string      `gorm:"unique;not null"`
    Password          string      `gorm:"not null"` // Must be hashed with bcrypt
    InvestmentHorizon int         `gorm:"default:5"` 
    RiskTolerance     int         `gorm:"default:3"` // Range 1 to 5
    Wallet            Wallet      // 1-to-1 relationship
    Portfolios        []Portfolio // 1-to-many relationship
}

type Wallet struct {
    ID      uint    `gorm:"primaryKey"`
    UserID  uint    // Foreign key to User
    Balance float64 `gorm:"default:0.0"` 
}

type Portfolio struct {
    ID     uint    `gorm:"primaryKey"`
    UserID uint    // Foreign key to User
    Ticker string  `gorm:"not null"` // e.g., "VTI", "BND"
    Shares float64 `gorm:"not null"` // Percentage or exact shares
}

// TODO: Add model for HistoricalMarketData to store yfinance prices
```

## 4. Development Roadmap

### Phase 1: Foundation

- [ ] Set up Docker Compose with Go, Python, and PostgreSQL in a private VPC.
- [ ] Connect Go to PostgreSQL using GORM.
- [ ] Define database models and run AutoMigrate.
- [ ] Create dummy user seeding for initial testing.
- [ ] Create basic POST /deposit endpoint to simulate adding funds.

### Phase 2: Authentication & Security

- [ ] Install `golang.org/x/crypto/bcrypt` and `github.com/golang-jwt/jwt/v5`.
- [ ] Implement POST /register: Hash password, create User, and attach empty Wallet.
- [ ] Implement POST /login: Verify bcrypt hash, generate and return JWT token.
- [ ] Create a Gin Middleware to protect routes (require Bearer Token).

### Phase 3: The Python Math Engine & Data Persistence

- [ ] Setup yfinance in Python to fetch historical data for base ETFs (e.g., VTI, VXUS, BND).
- [ ] Store/Update fetched historical data in PostgreSQL to avoid repeated external API calls.
- [ ] Write the Markowitz Optimization algorithm using scipy.optimize, reading historical data from the DB.
- [ ] Expose POST /optimize in Flask that accepts `{ "funds": 5000, "risk_tolerance": 4 }`.
- [ ] Return ideal portfolio weights (e.g., `{"VTI": 0.60, "BND": 0.40}`).

### Phase 4: Orchestration (Go + Python) & Stripe Integration

- [ ] Integrate Stripe Sandbox API in Go to securely handle the POST /deposit logic.
- [ ] Update Go's flow: Once Stripe confirms payment, Go triggers Python's /optimize endpoint.
- [ ] Receive weights from Python, calculate exact shares based on wallet balance.
- [ ] Deduct balance from Wallet and save the new assets into the Portfolios table.

### Phase 5: Frontend Dashboard

- [ ] Serve static HTML/JS from the Go router.
- [ ] Build Login/Register UI.
- [ ] Build Dashboard UI: Show current balance and Stripe deposit form.
- [ ] Fetch GET /user/portfolio and use Plotly.js to render a Pie Chart of the user's asset allocation.

### Phase 6: Cloud Deployment (Digital Ocean)

- [ ] Provision 2 Ubuntu Droplets.
- [ ] Configure internal VPC networking.
- [ ] Configure UFW firewall (block everything except SSH and Go's web ports).
- [ ] Deploy PostgreSQL and Go on Droplet 1, Python on Droplet 2.

### Phase 7: Blockchain Audit Log (Optional/Bonus)

- [ ] Create an AuditLog table in PostgreSQL.
- [ ] Every transaction hashes previous block + current data + Python script checksum (SHA-256).
- [ ] Build a frontend "Block Explorer" to prove algorithm and history integrity to the user.
