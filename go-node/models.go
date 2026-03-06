package main

// investor account
type User struct {
	ID                uint        `gorm:"primaryKey"`
	Email             string      `gorm:"unique;not null"`
	Password          string      `gorm:"not null"`  // brcypt hashed
	InvestmentHorizon int         `gorm:"default:5"` // years
	RiskTolerance     int         `gorm:"default:3"` // risk from 1 (min) to 5 (max)
	Wallet            Wallet      // one-to-one relation with financial balance
	Portfolios        []Portfolio // one-to-many reation with assets
}

// user's paper trading balance
type Wallet struct {
	ID      uint    `gorm:"primaryKey"`
	UserId  uint    // foreign key to user
	Balance float64 `gorm:"default:0.0"` // sum available to invest
}

// portofolio
type Portfolio struct {
	ID     uint    `gorm:"primaryKey"`
	UserId uint    // foreign key to user
	Ticker string  `gorm:"not null"` // "LYMS", "XDWI"
	Shares float64 `gorm:"not null"` // number of shares or percentage holding
}
