package models

import "github.com/golang-jwt/jwt/v5"

// struct to read incoming json data from request
type DepositRequest struct {
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

type Verify2FARequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Token    string `json:"token" binding:"required,len=6"`
}

type Enable2FARequest struct {
	Token string `json:"token" binding:"required,len=6"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
