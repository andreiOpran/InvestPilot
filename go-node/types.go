package main

import "github.com/golang-jwt/jwt/v5"

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

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// TODO: in production should be retrieved from env var
var jwtSecret = []byte("secret-key")
