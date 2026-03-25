package database

import (
	"fmt"
	"log"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := config.Env.DatabaseURL

	if dsn == "" {
		log.Fatal("DATABASE_URL is not set in configuration")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Fatal error: could not connect to PostgreSQL database: %v", err)
	}

	fmt.Println("Successfully connected to PostgreSQL.")

	// AutoMigrate automatically creates or updates the db tables
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.ActionToken{},
		&models.Wallet{},
		&models.Transaction{},
		&models.InvestmentRound{},
		&models.Portfolio{},
		&models.HistoricalMarketData{},
	); err != nil {
		log.Fatalf("Error during table migration: %v", err)
	}

	fmt.Println("Database tables migrated successfully.")

	// configure connection pool
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
}
