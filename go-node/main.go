package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// investor account
type User struct {
	ID                uint        `gorm:"primaryKey"`
	Email             string      `gorm:"unique;not null"`
	Password          string      `gorm:"not null"`  // brcypt hashed
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

// struct to read incoming json data from request
type DepositRequst struct {
	Amount float64 `json:"amount" binding:"required,gt=0"` // greater than 0
}

type RegisterRequest struct {
	Email             string `json:"email" binding:"required,email"`
	Password          string `json:"password" binding:"required,min=6"`
	RiskTolerance     int    `json:"risk_tolerance"`
	InvestmentHorizon int    `json:"investment_horizon"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TODO: in production should be retrieved from env var
var jwtSecret = []byte("secret-key")

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
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

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// read Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// header must be in format "Bearer <token>"
		// strings.TrimPrefix strips "Bearer ", leaving just the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // TrimPrefix returns original if prefix not found
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must start with Bearer"})
			return
		}

		// parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// ensure signing method is what we expect (prevent algorithm substitution attacks)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// inject user ID into the context, so handlers can read it
		c.Set("userID", claims.UserID)

		// continue to actual handler
		c.Next()
	}
}

func main() {
	initDB()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Go node works"})
	})

	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "Server is running",
			"database": "Connected",
		})
	})

	v1 := r.Group("/api/v1")
	{

		// endpoint that shows vpc communication
		v1.POST("/simulate-investment", func(c *gin.Context) {
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

		v1.POST("/register", func(c *gin.Context) {
			var req RegisterRequest

			// validate incoming json
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// hash the password with cost 14
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
				return
			}

			// build user with an empty wallet
			user := User{
				Email:             req.Email,
				Password:          string(hashedPassword),
				RiskTolerance:     req.RiskTolerance,
				InvestmentHorizon: req.InvestmentHorizon,
				Wallet:            Wallet{Balance: 0.0},
			}

			// save to DB (will fail if email already exists)
			if err := DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message": "User registered successfully",
				"user_id": user.ID,
			})
		})

		v1.POST("/login", func(c *gin.Context) {
			var req LoginRequest

			// validate incoming json
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// look up user by email
			var user User
			if err := DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
				// vague, do not reveal whether email exists
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
				return
			}

			// compare provided password against stored bcrypt hash
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
				// vague, do not reveal whether email exists
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
				return
			}

			// build JWT claims, token expires in 24 hours
			claims := Claims{
				UserID: user.ID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}

			// sign the token with HS256
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString(jwtSecret)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"token": tokenString,
			})
		})

		// protected: JWT required for all routes inside
		protected := v1.Group("/", authMiddleware())
		{
			protected.GET("/user", func(c *gin.Context) {
				var user User
				userID := c.MustGet("userID").(uint)

				// Preload("Wallet") tells GORM to also fetch the attached Wallet data
				if err := DB.Preload("Wallet").First(&user, userID).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"user_id":            user.ID,
					"email":              user.Email,
					"risk_tolerance":     user.RiskTolerance,
					"investment_horizon": user.InvestmentHorizon,
					"wallet_balance":     user.Wallet.Balance,
				})
			})

			protected.POST("/deposit", func(c *gin.Context) {
				var req DepositRequst
				userID := c.MustGet("userID").(uint)

				// 1. read and validate the JSON body from the request
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide a valid amount greater than 0"})
					return
				}

				var user User
				// 2. find the authenticated user and their attached wallet
				if err := DB.Preload("Wallet").First(&user, userID).Error; err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
					return
				}

				// 3. add simulated money to the wallet
				user.Wallet.Balance += req.Amount

				user.Wallet.UserId = user.ID

				// 4. save updated walet to the database
				DB.Save(&user.Wallet)

				// 5. send a succes response back
				c.JSON(http.StatusOK, gin.H{
					"message":     "Paper trading deposit successful.",
					"added":       req.Amount,
					"new_balance": user.Wallet.Balance,
				})
			})
		}
	}

	fmt.Println("Operational Node (Go) starting on port 8080...")

	r.Run(":8080")
}
