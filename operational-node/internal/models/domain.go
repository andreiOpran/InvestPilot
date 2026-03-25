package models

import "time"

// investor account
type User struct {
	ID                uint   `gorm:"primaryKey"`
	Email             string `gorm:"unique;not null"`
	Password          string `gorm:"not null"` // bcrypt hashed
	IsEmailVerified   bool   `gorm:"default:false"`
	TwoFactorSecret   string // secret for Google Authenticator/TOTP
	IsTwoFactorEnable bool   `gorm:"default:false"`
	InvestmentHorizon int    // in years
	RiskTolerance     int    // risk from 1 (min) to 5 (max)
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Wallet            Wallet        // one-to-one relation with trading balance
	Sessions          []Session     // one-to-many relationship
	ActionTokens      []ActionToken // one-to-many relationship
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
	ExpiresAt    time.Time `gorm:"not null;index"` // indexed for fast cleanup
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// manages temporary, short-lived tokens (email verification, password reset)
type ActionToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	Token     string    `gorm:"unique;not null"`
	Type      string    `gorm:"not null"`       // "verify_email", "reset_password"
	ExpiresAt time.Time `gorm:"not null;index"` // indexed for fast cleanup
	CreatedAt time.Time
}

// user's paper trading balance, uninvested money available to deposit or withdraw
type Wallet struct {
	ID        uint    `gorm:"primaryKey"`
	UserID    uint    `gorm:"unique;not null"` // foreign key to user
	Balance   float64 `gorm:"not null;default:0.0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// tracks money moving in and out of the portfolio for the contribution chart
type Transaction struct {
	ID        uint    `gorm:"primaryKey"`
	UserID    uint    `gorm:"not null;index"`
	Type      string  `gorm:"not null"` // "invest" or "sell"
	Amount    float64 `gorm:"not null"`
	CreatedAt time.Time
}

// groups all portfolio trades belonging to one optimization run
type InvestmentRound struct {
	ID         uint    `gorm:"primaryKey"`
	UserID     uint    `gorm:"not null;index"`        // foreign key to user
	TotalValue float64 `gorm:"not null"`              // total amount this round
	IsActive   bool    `gorm:"not null;default:true"` // false after a newer round replaces it
	CreatedAt  time.Time
	Holdings   []Holding // one-to-many relationship with holdings
}

// a single holding within an investment round
// can be ETF ("LYMS", "XDWI") or cash ("USD")
type Holding struct {
	ID                uint    `gorm:"primaryKey"`
	UserID            uint    `gorm:"not null;index"` // foreign key to user
	InvestmentRoundID uint    `gorm:"not null;index"` // foreign key to InvestmentRound
	Ticker            string  `gorm:"not null"`       // "LYMS", "XDWI" or "USD"
	Weight            float64 `gorm:"not null"`       // markowitz weight (0.40 or 1.0 for USD)
	Shares            float64 `gorm:"not null"`       // number of shares or dollar amount for USD
	PurchasePrice     float64 `gorm:"not null"`       // price per share at purchase, 1.0 for USD
	AllocatedAmount   float64 `gorm:"not null"`       // total dollars in this holding
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type HistoricalMarketData struct {
	ID         uint      `gorm:"primaryKey"`
	Ticker     string    `gorm:"not null;uniqueIndex:idx_ticker_date"` // "LYMS", "XDWI"
	Date       time.Time `gorm:"not null;uniqueIndex:idx_ticker_date"` // trading day
	ClosePrice float64   `gorm:"not null"`                             // adjusted price at closing
	CreatedAt  time.Time
}
