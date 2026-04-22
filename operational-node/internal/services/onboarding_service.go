package services

import (
	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"github.com/andreiOpran/licenta/operational-node/internal/repositories"
)

type OnboardingService interface {
	GetQuestions() []models.OnboardingQuestion
	SubmitOnboarding(userID uint, req models.OnboardingSubmitRequest) error
}

type onboardingService struct {
	userRepo repositories.UserRepository
}

func NewOnboardingService(userRepo repositories.UserRepository) OnboardingService {
	return &onboardingService{userRepo: userRepo}
}

func (s *onboardingService) GetQuestions() []models.OnboardingQuestion {
	cfg := config.Env.Onboarding

	return []models.OnboardingQuestion{
		{
			ID:   "q1_age",
			Text: "What is your age group?",
			Options: []models.OnboardingOption{
				{ID: "age_20", Text: "Under 30 years old", RiskScore: cfg.RiskScores["age_20"], HorizonYrs: cfg.HorizonYrs["age_20"]},
				{ID: "age_30", Text: "30 - 45 years old", RiskScore: cfg.RiskScores["age_30"], HorizonYrs: cfg.HorizonYrs["age_30"]},
				{ID: "age_45", Text: "45 - 60 years old", RiskScore: cfg.RiskScores["age_45"], HorizonYrs: cfg.HorizonYrs["age_45"]},
				{ID: "age_60", Text: "Over 60 years old", RiskScore: cfg.RiskScores["age_60"], HorizonYrs: cfg.HorizonYrs["age_60"]},
			},
		},
		{
			ID:   "q2_goal",
			Text: "What is your primary investment goal?",
			Options: []models.OnboardingOption{
				{ID: "goal_growth", Text: "Aggressive capital growth", RiskScore: cfg.RiskScores["goal_growth"], HorizonYrs: cfg.HorizonYrs["goal_growth"]},
				{ID: "goal_balanced", Text: "Moderate, medium-term growth", RiskScore: cfg.RiskScores["goal_balanced"], HorizonYrs: cfg.HorizonYrs["goal_balanced"]},
				{ID: "goal_preserve", Text: "Capital preservation (low risk)", RiskScore: cfg.RiskScores["goal_preserve"], HorizonYrs: cfg.HorizonYrs["goal_preserve"]},
			},
		},
		{
			ID:   "q3_drop",
			Text: "If your portfolio dropped 20% in a month, what would you do?",
			Options: []models.OnboardingOption{
				{ID: "drop_buy", Text: "Buy more (it's on discount!)", RiskScore: cfg.RiskScores["drop_buy"], HorizonYrs: cfg.HorizonYrs["drop_buy"]},
				{ID: "drop_hold", Text: "Do nothing, wait for recovery", RiskScore: cfg.RiskScores["drop_hold"], HorizonYrs: cfg.HorizonYrs["drop_hold"]},
				{ID: "drop_sell", Text: "Sell everything to stop the bleeding", RiskScore: cfg.RiskScores["drop_sell"], HorizonYrs: cfg.HorizonYrs["drop_sell"]},
			},
		},
	}
}

func (s *onboardingService) SubmitOnboarding(userID uint, req models.OnboardingSubmitRequest) error {
	totalRisk := 0
	totalHorizon := 0
	answeredRiskQs := 0
	answeredHorizonQs := 0

	questions := s.GetQuestions()

	// validate answers and accumulate scores
	for _, q := range questions {
		optionID, answered := req.Answers[q.ID]
		if !answered {
			return ErrMissingAnswer
		}

		validOption := false
		for _, opt := range q.Options {
			if opt.ID == optionID {
				totalRisk += opt.RiskScore
				answeredRiskQs++

				// only consider questions that contribute to horizon
				if opt.HorizonYrs > 0 {
					totalHorizon += opt.HorizonYrs
					answeredHorizonQs++
				}
				validOption = true
				break
			}
		}

		if !validOption {
			return ErrInvalidOption
		}
	}

	// compute average risk tolerance (clamped between 1 and 5)
	riskTolerance := totalRisk / answeredRiskQs
	if riskTolerance < 1 {
		riskTolerance = 1
	} else if riskTolerance > 5 {
		riskTolerance = 5
	}

	// compute average investment horizon (fallback to config default if empty)
	investmentHorizon := config.Env.Onboarding.DefaultHorizon
	if answeredHorizonQs > 0 {
		investmentHorizon = totalHorizon / answeredHorizonQs
	}

	// update user profile in the db
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	user.RiskTolerance = riskTolerance
	user.InvestmentHorizon = investmentHorizon

	return s.userRepo.Save(user)
}
