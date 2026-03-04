package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	AppEnv   string `envconfig:"APP_ENV" default:"dev"`
	HTTPAddr string `envconfig:"HTTP_ADDR" default:":8080"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	JWTIssuer        string        `envconfig:"JWT_ISSUER" default:"english-anywhere-lab"`
	JWTAccessTTL     time.Duration `envconfig:"JWT_ACCESS_TTL_MIN" default:"60m"`
	JWTRefreshTTL    time.Duration `envconfig:"JWT_REFRESH_TTL_HOUR" default:"720h"`
	JWTSignKey       string        `envconfig:"JWT_SIGN_KEY" required:"true"`

	DBDriver          string `envconfig:"DB_DRIVER" default:"sqlite"`
	SQLitePath        string `envconfig:"SQLITE_PATH" default:"./data/app.db"`
	SQLiteWAL         bool   `envconfig:"SQLITE_WAL" default:"true"`
	SQLiteBusyTimeout int    `envconfig:"SQLITE_BUSY_TIMEOUT_MS" default:"5000"`

	// LLM
	LLMBaseURL    string `envconfig:"LLM_BASE_URL" default:"https://api.openai.com/v1"`
	LLMAPIKey     string `envconfig:"LLM_API_KEY"`
	LLMModel      string `envconfig:"LLM_MODEL" default:"gpt-4o-mini"`
	LLMTimeoutSec int    `envconfig:"LLM_TIMEOUT_SEC" default:"60"`
	LLMMaxRetries int    `envconfig:"LLM_MAX_RETRIES" default:"2"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
