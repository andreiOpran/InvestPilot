package services

import (
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
)

func init() {
	config.Env = config.AppSettings{
		BcryptCost:                4,
		SecureTokenBytes:          32,
		FamilyIDBytes:             16,
		VerifyEmailLifetime:       24 * time.Hour,
		ResetPasswordLifetime:     15 * time.Minute,
		AccessTokenLifetime:       15 * time.Minute,
		RefreshTokenLifetimeHours: 24 * time.Hour,
		AESMasterKey:              "0123456789abcdef0123456789abcdef",
		JWTSecret:                 "mock-secret-key",
		FrontendBaseURL:           "http://localhost",
		TimingAttackTarget:        1 * time.Millisecond,
		TimingAttackNoise:         1,
		SMTPHost:                  "localhost",
		SMTPPort:                  "1025",
		PasswordMinLength:         8,
		PasswordMaxLength:         128,
		PasswordMinZxcvbnStrength: 0,
		LockoutThreshold1:         4,
		LockoutDuration1:          1 * time.Minute,
		LockoutThreshold2:         5,
		LockoutDuration2:          3 * time.Minute,
		LockoutThreshold3:         6,
		LockoutDuration3:          15 * time.Minute,
		LoginAttemptScanningLimit: 30,
		TransactionCountLimit:     100,
		TransactionCountDefault:   10,
		Onboarding: config.OnboardingConfig{
			RiskScores: map[string]int{
				"age_20":        5,
				"age_30":        4,
				"age_45":        2,
				"age_60":        1,
				"goal_growth":   5,
				"goal_balanced": 3,
				"goal_preserve": 1,
				"drop_buy":      5,
				"drop_hold":     3,
				"drop_sell":     1,
			},
			HorizonYrs: map[string]int{
				"age_20":        10,
				"age_30":        7,
				"age_45":        4,
				"age_60":        2,
				"goal_growth":   10,
				"goal_balanced": 5,
				"goal_preserve": 1,
				"drop_buy":      0,
				"drop_hold":     0,
				"drop_sell":     0,
			},
			DefaultHorizon: 5,
		},
	}

	mailer.InitEmailer()
}
