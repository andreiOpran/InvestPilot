package models

// struct to read incoming json data from request
type DepositRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"` // greater than 0
}

// request to move money from wallet to InvestmentRound
type InvestRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type RegisterRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

// struct used for the onboarding form after the user registers
type UpdateProfileRequest struct {
	RiskTolerance     int `json:"risk_tolerance" binding:"required,min=1,max=5"`
	InvestmentHorizon int `json:"investment_horizon" binding:"required,min=1,max=50"`
}

type LoginRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
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
	Email          string `json:"email" binding:"required,email"`
	TurnstileToken string `json:"turnstile_token" binding:"required"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForecastRequest struct {
	InitialInvestment   *float64 `json:"initial_investment" binding:"required,min=0"`
	MonthlyContribution float64  `json:"monthly_contribution" binding:"min=0"`
	Years               int      `json:"years" binding:"required,min=1,max=50"`
}

type DepositIntentRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type CashoutRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}
