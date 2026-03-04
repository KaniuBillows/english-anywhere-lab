package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/worker"
)

func main() {
	application, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	packRepo := pack.NewRepository(application.DB)
	llmClient := llm.NewClient(application.Config)
	generator := worker.NewGenerator(packRepo, llmClient, application.DB, application.Logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	generator.Run(ctx)
	application.Logger.Info("worker shut down")
}
