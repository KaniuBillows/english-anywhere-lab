package app

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/db"
)

type App struct {
	Config *config.Config
	DB     *sql.DB
	Logger *slog.Logger
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	database, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}

	if err := db.Migrate(database); err != nil {
		database.Close()
		return nil, err
	}

	return &App{
		Config: cfg,
		DB:     database,
		Logger: logger,
	}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
