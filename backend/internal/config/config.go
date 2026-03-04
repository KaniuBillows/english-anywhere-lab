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

	// Storage
	FilesProvider       string `envconfig:"FILES_PROVIDER" default:"local"`
	FilesLocalRoot      string `envconfig:"FILES_LOCAL_ROOT" default:"./data/files"`
	FilesBaseURL        string `envconfig:"FILES_BASE_URL" default:"/static/files"`
	FilesS3Endpoint     string `envconfig:"FILES_S3_ENDPOINT"`
	FilesS3Region       string `envconfig:"FILES_S3_REGION"`
	FilesS3Bucket       string `envconfig:"FILES_S3_BUCKET"`
	FilesS3AccessKey    string `envconfig:"FILES_S3_ACCESS_KEY"`
	FilesS3SecretKey    string `envconfig:"FILES_S3_SECRET_KEY"`
	FilesS3ForcePathStyle bool  `envconfig:"FILES_S3_FORCE_PATH_STYLE" default:"true"`
	FilesS3PublicURL    string `envconfig:"FILES_S3_PUBLIC_URL"`

	// TTS
	TTSEnabled           bool    `envconfig:"TTS_ENABLED" default:"false"`
	TTSProvider          string  `envconfig:"TTS_PROVIDER" default:"stub"`
	TTSVoice             string  `envconfig:"TTS_VOICE" default:"en_default_female"`
	TTSSampleRate        int     `envconfig:"TTS_SAMPLE_RATE" default:"22050"`
	TTSSpeed             float32 `envconfig:"TTS_SPEED" default:"1.0"`
	TTSOutputFormat      string  `envconfig:"TTS_OUTPUT_FORMAT" default:"wav"`
	TTSMaxTextChars      int     `envconfig:"TTS_MAX_TEXT_CHARS" default:"280"`
	TTSWorkerConcurrency int     `envconfig:"TTS_WORKER_CONCURRENCY" default:"2"`
	TTSRetryMax          int     `envconfig:"TTS_RETRY_MAX" default:"2"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
