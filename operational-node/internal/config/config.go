package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// AppSettings holds application configuration populated from environment
type AppSettings struct {
	DatabaseURL              string           `env:"DATABASE_URL,required"`
	AESMasterKey             string           `env:"AES_MASTER_KEY,required"`
	JWTSecret                string           `env:"JWT_SECRET" envDefault:"secret-key"`
	SMTPHost                 string           `env:"SMTP_HOST"`
	SMTPPort                 string           `env:"SMTP_PORT"`
	SMTPUser                 string           `env:"SMTP_USER,required"`
	SMTPPass                 string           `env:"SMTP_PASS,required"`
	SMTPFrom                 string           `env:"SMTP_FROM"`
	SMTPTestDestination      string           `env:"SMTP_TEST_DESTINATION"`
	AccessTokenLifetime      time.Duration    `env:"ACCESS_TOKEN_LIFETIME" envDefault:"10m"`
	RefreshTokenLifetime     time.Duration    `env:"REFRESH_TOKEN_LIFETIME" envDefault:"168h"`
	VerifyEmailLifetime      time.Duration    `env:"VERIFY_EMAIL_LIFETIME" envDefault:"24h"`
	ResetPasswordLifetime    time.Duration    `env:"RESET_PASSWORD_LIFETIME" envDefault:"15m"`
	CleanupCronSchedule      string           `env:"CLEANUP_CRON" envDefault:"0 3 * * *"`
	CleanupBatchSize         int              `env:"CLEANUP_BATCH_SIZE" envDefault:"1000"`
	ServerPort               string           `env:"PORT" envDefault:"8080"`
	BcryptCost               int              `env:"BCRYPT_COST" envDefault:"14"`
	SecureTokenBytes         int              `env:"SECURE_TOKEN_BYTES" envDefault:"32"`
	FamilyIDBytes            int              `env:"FAMILY_ID_BYTES" envDefault:"16"`
	TimingAttackTarget       time.Duration    `env:"TIMING_ATTACK_TARGET" envDefault:"100ms"`
	TimingAttackNoise        int              `env:"TIMING_ATTACK_NOISE" envDefault:"20"`
	CronBatchSleep           time.Duration    `env:"CRON_BATCH_SLEEP" envDefault:"100ms"`
	DataPipelineCronSchedule string           `env:"DATA_PIPELINE_CRON" envDefault:"0 22 * * *"`
	APIBaseURL               string           `env:"API_BASE_URL" envDefault:"http://localhost:8080/api/v1"`
	FrontendBaseURL          string           `env:"FRONTEND_BASE_URL" envDefault:"http://localhost:8081"`
	PythonNodeURL            string           `env:"PYTHON_NODE_URL" envDefault:"http://python-engine:5000"`
	PythonClientTimeout      time.Duration    `env:"PYTHON_CLIENT_TIMEOUT" envDefault:"5s"`
	RabbitMQURL              string           `env:"RABBITMQ_URL,required"`
	Investment               InvestmentConfig `envPrefix:""`
}

type InvestmentConfig struct {
	EquityTickers           []string           `json:"equity_tickers" env:"INVEST_EQUITY_TICKERS" envDefault:"VTI,VOO,QQQ,VTV,VUG,IWM,VEA,VWO,VNQ,VNQI,XLF,XLV,XLE,XLK"`
	BondTickers             []string           `json:"bond_tickers" env:"INVEST_BOND_TICKERS" envDefault:"BND,TLT,LQD,HYG,BNDX"`
	BaseEquityAllocation    map[int]float64    `json:"base_equity_allocation" env:"INVEST_BASE_EQUITY_ALLOC" envDefault:"1:0.20,2:0.40,3:0.60,4:0.80,5:0.90"`
	HorizonShortMax         int                `json:"horizon_short_max" env:"HORIZON_SHORT_MAX" envDefault:"2"`
	HorizonMediumMax        int                `json:"horizon_medium_max" env:"HORIZON_MEDIUM_MAX" envDefault:"6"`
	HorizonMultipliers      map[string]float64 `json:"horizon_multipliers" env:"INVEST_HORIZON_MULTIPLIERS" envDefault:"short:0.70,medium:1.00,long:1.10"`
	MaxEquityCap            float64            `json:"max_equity_cap" env:"INVEST_MAX_EQUITY_CAP" envDefault:"0.95"`
	TopNEquities            int                `json:"top_n_equities" env:"INVEST_TOP_N_EQUITIES" envDefault:"6"`
	WeightThreshold         float64            `json:"weight_threshold" env:"INVEST_WEIGHT_THRESHOLD" envDefault:"0.02"`
	RebalanceDeltaThreshold float64            `json:"rebalance_delta_threshold" env:"INVEST_REBALANCE_DELTA" envDefault:"0.02"`
	CashFirstEnabled        bool               `json:"cash_first_enabled" env:"INVEST_CASH_FIRST_ENABLED" envDefault:"true"`
	PriceStalenessDays      int                `json:"price_staleness_days" env:"INVEST_PRICE_STALENESS_DAYS" envDefault:"2"`
	Verbose                 bool               `json:"verbose" env:"VERBOSE_DECISIONAL_NODE" envDefault:"false"`
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
