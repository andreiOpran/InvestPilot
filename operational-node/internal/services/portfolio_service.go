package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type PortfolioService interface {
	Invest(userID uint, amount float64) error
}

type portfolioService struct {
	portfolioRepo repositories.PortfolioRepository
	userRepo      repositories.UserRepository
}

func NewPortfolioService(portfolioRepo repositories.PortfolioRepository, userRepo repositories.UserRepository) PortfolioService {
	return &portfolioService{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
	}
}

func (s *portfolioService) Invest(userID uint, amount float64) error {
	// domain check: fetch wallet and validate balance
	wallet, err := s.userRepo.FindWalletByUserID(userID)
	if err != nil {
		return err
	}
	if wallet.Balance < amount {
		return ErrInsufficientBalance
	}

	// domain action: deduct money locally
	wallet.Balance -= amount

	// domain action: create transaction
	txRecord := &models.Transaction{
		UserID: userID,
		Type:   "invest",
		Amount: amount,
	}

	oldRound, err := s.portfolioRepo.GetActiveRoundWithHoldings(userID)
	if err != nil {
		return err
	}

	var newTotalValue float64
	var newHoldings []models.Holding
	usdFound := false

	// domain logic: adjust holdings
	if oldRound != nil {
		oldRound.IsActive = false // retire
		newTotalValue = oldRound.TotalValue + amount

		for _, h := range oldRound.Holdings {
			// copy old asset tracking
			newH := models.Holding{
				UserID:          userID,
				Ticker:          h.Ticker,
				Weight:          h.Weight,
				Shares:          h.Shares,
				PurchasePrice:   h.PurchasePrice,
				AllocatedAmount: h.AllocatedAmount,
			}

			// add funds to USD bucket if it exists
			if h.Ticker == "USD" {
				newH.Shares += amount
				newH.AllocatedAmount += amount
				usdFound = true
			}
			newHoldings = append(newHoldings, newH)
		}
	} else {
		newTotalValue = amount
	}

	// if no USD position existed previously, initialize it here
	if !usdFound {
		newHoldings = append(newHoldings, models.Holding{
			UserID:          userID,
			Ticker:          "USD",
			Weight:          amount / newTotalValue, // roughly 1.0
			Shares:          amount,
			PurchasePrice:   1.0,
			AllocatedAmount: amount,
		})
	}

	// build final new InvestmentRound object
	newRound := &models.InvestmentRound{
		UserID:     userID,
		TotalValue: newTotalValue,
		IsActive:   true,
		Holdings:   newHoldings, // Repo handles writing all these objects natively via GORM cascade
	}

	// give prepared domain models to repo to execute as one transaction
	return s.portfolioRepo.ExecuteInvestTransaction(wallet, txRecord, oldRound, newRound)
}
