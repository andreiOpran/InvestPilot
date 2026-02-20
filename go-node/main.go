package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// investor account
type User struct {
	ID                uint        `gorm:"primaryKey"`
	Email             string      `gorm:"unique;not null"`
	Password          string      `gorm:"not null"`  // TODO: hash later for security
	InvestmentHorizon int         `gorm:"default:5"` // years
	RiskTolerance     int         `gorm:"default:3"` // risk from 1 (min) to 5 (max)
	Wallet            Wallet      // one-to-one relation with financial balance
	Portofolios       []Portfolio // one-to-many reation with assets
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
	err = DB.AutoMigrate(&User{}, &Wallet{}, &Portfolio{})
	if err != nil {
		log.Fatal("Error during table migration: ", err)
	}
	fmt.Println("Database tables migrated successfully.")
}

func main() {
	initDB()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go node works"})
	})

	// endpoint that shows vpc communication
	r.POST("/simulate-investment", func(c *gin.Context) {
		// make a request to the py container using the name of the service from docker-compose
		resp, err := http.Post("http://python-engine:5000/optimize", "application/json", nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error commincating with Py node"})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// forward response to frontend
		c.Data(http.StatusOK, "application/json", body)
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "Server is running",
			"database": "Connected",
		})
	})

	fmt.Println("Operational Node (Go) starting on port 8080...")

	r.Run(":8080")
}
