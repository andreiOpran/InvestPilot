package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/clients"
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
	"github.com/google/uuid"
)

type ForecastService interface {
	RequestForecast(userID uint, req models.ForecastRequest) (string, error)
	GetForecastByTaskID(taskID string) (*models.ForecastResult, error)
}

type forecastService struct {
	forecastRepo  repositories.ForecastRepository
	portfolioRepo repositories.PortfolioRepository
}

func NewForecastService(fr repositories.ForecastRepository, pr repositories.PortfolioRepository) ForecastService {
	return &forecastService{
		forecastRepo:  fr,
		portfolioRepo: pr,
	}
}

func (s *forecastService) RequestForecast(userID uint, req models.ForecastRequest) (string, error) {
	// get user current holdings
	round, err := s.portfolioRepo.GetActiveRoundWithHoldings(userID)
	if err != nil {
		return "", err
	}
	if round == nil || len(round.Holdings) == 0 {
		return "", ErrForecastUserNoActivePortfolio
	}

	// extract weights (ignore USD because we are making market projection)
	weights := make(map[string]float64)
	totalInvested := 0.0

	for _, h := range round.Holdings {
		if h.Ticker != "USD" {
			weights[h.Ticker] = h.AllocatedAmount
			totalInvested += h.AllocatedAmount
		}
	}

	if totalInvested == 0 {
		return "", ErrForecastNoAssetsOnlyCash
	}

	// normalize remaining weights to sum to 1.0
	for ticker, amount := range weights {
		weights[ticker] = amount / totalInvested
	}

	// generate uuid for async task
	taskID := uuid.New().String()

	// create pending db record first
	if err := s.forecastRepo.CreatePendingForecast(taskID); err != nil {
		return "", err
	}

	// construct payload for decisional node
	payload := models.ForecastPayload{
		TaskID:              taskID,
		Weights:             weights,
		InitialInvestment:   req.InitialInvestment,
		MonthlyContribution: req.MonthlyContribution,
		Years:               req.Years,
		Verbose:             config.Env.Investment.Verbose,
	}

	// publish async command for decisional node via rabbitmq
	if err := clients.Publisher.PublishCommand("CMD_FORECAST", payload); err != nil {
		return "", err
	}

	return taskID, nil
}

func (s *forecastService) GetForecastByTaskID(taskID string) (*models.ForecastResult, error) {
	return s.forecastRepo.GetForecastByTaskID(taskID)
}
