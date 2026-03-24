package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func initDB() {
	dsn := "host=db user=admin password=pass dbname=robo_advisory port=5432 sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Fatal error: could not connect to PostgreSQL database! \n", err)
	}

	fmt.Println("Successfully connected to PostgreSQL.")

	// AutoMigrate automatically creates or updates the db table
	DB.AutoMigrate(
		&User{},
		&Session{},
		&ActionToken{},
		&Wallet{},
		&Transaction{},
		&InvestmentRound{},
		&Portfolio{},
		&HistoricalMarketData{},
	)
	if err != nil {
		log.Fatal("Error during table migration: ", err)
	}
	fmt.Println("Database tables migrated successfully.")
}

func main() {
	initDB()
	initEmailer()
	StartTokenCleanupJob()

	r := gin.Default()

	fmt.Println("Operational Node (Go) starting on port 8080...")

	RegisterRoutes(r)

	r.Run(":8080")
}
