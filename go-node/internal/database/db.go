package database

import (
	"fmt"
	"log"

	"licenta/go-node/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := "host=db user=admin password=pass dbname=robo_advisory port=5432 sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Fatal error: could not connect to PostgreSQL database! \n", err)
	}

	fmt.Println("Successfully connected to PostgreSQL.")

	// AutoMigrate automatically creates or updates the db table
	DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.ActionToken{},
		&models.Wallet{},
		&models.Transaction{},
		&models.InvestmentRound{},
		&models.Portfolio{},
		&models.HistoricalMarketData{},
	)
	if err != nil {
		log.Fatal("Error during table migration: ", err)
	}
	fmt.Println("Database tables migrated successfully.")
}
