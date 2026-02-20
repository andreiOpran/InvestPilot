package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- 1. MODELS (TABLE DEFINITIONS) ---

type User struct {
	ID                uint   `gorm:"primaryKey"`
	Email             string `gorm:"unique;not null"`
	Password          string `gorm:"not null"`
	InvestmentHorizon int    `gorm:"default:5"`
	RiskTolerance     int    `gorm:"default:3"`
	Wallet            Wallet
	Portfolios        []Portfolio
}

type Wallet struct {
	ID      uint `gorm:"primaryKey"`
	UserID  uint
	Balance float64 `gorm:"default:0.0"`
}

type Portfolio struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint
	Ticker string  `gorm:"not null"`
	Shares float64 `gorm:"not null"`
}

var DB *gorm.DB

// --- 2. DATABASE CONNECTION ---

func initDB() {
	dsn := "host=db user=admin password=pass dbname=robo_advisory port=5432 sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Fatal error: Could not connect to PostgreSQL! \n", err)
	}

	fmt.Println("✅ Successfully connected to PostgreSQL!")

	// Create tables if they don't exist
	DB.AutoMigrate(&User{}, &Wallet{}, &Portfolio{})
	fmt.Println("✅ Database tables ready!")
}

// --- 3. SEED DUMMY USER ---

func seedDummyUser() {
	var user User

	// Check if our test user already exists in the database
	result := DB.Where("email = ?", "test@roboadvisor.com").First(&user)

	// If the user does not exist, we create one
	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		fmt.Println("⚠️ Dummy user not found. Creating one now...")

		dummyUser := User{
			Email:             "test@roboadvisor.com",
			Password:          "password123", // Fake password for testing
			InvestmentHorizon: 10,            // Example: 10 years
			RiskTolerance:     4,             // Example: Moderately High Risk
			Wallet: Wallet{
				Balance: 0.0, // Wallet starts empty
			},
		}

		// Save the user and their wallet to the database
		DB.Create(&dummyUser)
		fmt.Println("✅ Dummy user created successfully with an empty wallet!")
	} else {
		fmt.Println("✅ Dummy user already exists in the database. Ready for testing!")
	}
}

// --- 4. MAIN SERVER ---

func main() {
	initDB()

	// Run the seeding function every time the server starts
	seedDummyUser()

	r := gin.Default()

	// Simple endpoint to check server status
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "Server is running perfectly",
			"database": "Connected",
		})
	})

	// NEW ENDPOINT: Fetch our dummy user and their wallet
	r.GET("/user", func(c *gin.Context) {
		var user User

		// Preload("Wallet") tells GORM to also fetch the attached Wallet data
		if err := DB.Preload("Wallet").Where("email = ?", "test@roboadvisor.com").First(&user).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Return the data as JSON to Postman/Frontend
		c.JSON(http.StatusOK, gin.H{
			"user_id":            user.ID,
			"email":              user.Email,
			"risk_tolerance":     user.RiskTolerance,
			"investment_horizon": user.InvestmentHorizon,
			"wallet_balance":     user.Wallet.Balance,
		})
	})

	fmt.Println("🚀 Operational Node (Go) starting on port 8080...")
	r.Run(":8080")
}
