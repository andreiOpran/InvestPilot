package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// AppSettings holds application configuration populated from environment
type AppSettings struct {
	DatabaseURL           string        `env:"DATABASE_URL,required"`
	AESMasterKey          string        `env:"AES_MASTER_KEY,required"`
	JWTSecret             string        `env:"JWT_SECRET" envDefault:"secret-key"`
	SMTPHost              string        `env:"SMTP_HOST"`
	SMTPPort              string        `env:"SMTP_PORT"`
	SMTPUser              string        `env:"SMTP_USER,required"`
	SMTPPass              string        `env:"SMTP_PASS,required"`
	SMTPFrom              string        `env:"SMTP_FROM"`
	SMTPTestDestination   string        `env:"SMTP_TEST_DESTINATION"`
	AccessTokenLifetime   time.Duration `env:"ACCESS_TOKEN_LIFETIME" envDefault:"10m"`
	RefreshTokenLifetime  time.Duration `env:"REFRESH_TOKEN_LIFETIME" envDefault:"168h"`
	VerifyEmailLifetime   time.Duration `env:"VERIFY_EMAIL_LIFETIME" envDefault:"24h"`
	ResetPasswordLifetime time.Duration `env:"RESET_PASSWORD_LIFETIME" envDefault:"15m"`
	CleanupBatchSize      int           `env:"CLEANUP_BATCH_SIZE" envDefault:"1000"`
}

// Env is the global configuration instance
var Env AppSettings

// LoadConfig loads environment variables from .env and parses them into Env
func LoadConfig() {
	_ = godotenv.Load()

	if err := env.Parse(&Env); err != nil {
		log.Fatalf("Failed to parse env vars: %v", err)
	}
}
