package services

import (
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/internal/mailer"
)

// init runs once before any tests to configure environment variables and bypass panics
func init() {
	config.Env = config.AppSettings{
		BcryptCost:                4, // use lowest cost for fast test execution
		SecureTokenBytes:          32,
		FamilyIDBytes:             16,
		VerifyEmailLifetime:       24 * time.Hour,
		ResetPasswordLifetime:     15 * time.Minute,
		AccessTokenLifetime:       15 * time.Minute,
		RefreshTokenLifetimeHours: 24 * time.Hour,
		AESMasterKey:              "0123456789abcdef0123456789abcdef",
		JWTSecret:                 "mock-secret-key",
		APIBaseURL:                "http://localhost",
		FrontendBaseURL:           "http://localhost",
		TimingAttackTarget:        1 * time.Millisecond,
		TimingAttackNoise:         1,
		SMTPHost:                  "localhost",
		SMTPPort:                  "1025",
	}

	// initialize mailer with dummy values
	mailer.InitEmailer()
}
