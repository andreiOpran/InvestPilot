package repositories

import (
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	config.Env.LoginAttemptScanningLimit = 30
	config.Env.CleanupBatchSize = 100
}

func setupTestDB() (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	database.DB = db

	database.DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.ActionToken{},
		&models.LoginAttempt{},
		&models.Wallet{},
		&models.Funding{},
		&models.Transaction{},
		&models.InvestmentRound{},
		&models.Holding{},
		&models.ForecastResult{},
		&models.DailyMarketData{},
		&models.IntradayMarketData{},
		&models.ModelPortfolio{},
	)

	return db, func() {
		database.DB = nil
	}
}
