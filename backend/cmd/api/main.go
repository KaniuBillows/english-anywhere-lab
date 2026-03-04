package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/output"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/plan"
	"github.com/bennyshi/english-anywhere-lab/internal/progress"
	"github.com/bennyshi/english-anywhere-lab/internal/review"
	"github.com/bennyshi/english-anywhere-lab/internal/scheduler"
	router "github.com/bennyshi/english-anywhere-lab/internal/transport/http"
)

func main() {
	application, err := app.New()
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer application.Close()

	// Wire dependencies
	authRepo := auth.NewRepository(application.DB)
	authJWT := auth.NewJWTManager(application.Config)
	authSvc := auth.NewService(authRepo, authJWT)

	fsrs := scheduler.NewFSRS()

	reviewRepo := review.NewRepository(application.DB)
	reviewSvc := review.NewService(reviewRepo, fsrs)

	planRepo := plan.NewRepository(application.DB)
	planSvc := plan.NewService(planRepo)

	progressRepo := progress.NewRepository(application.DB)
	progressSvc := progress.NewService(progressRepo)

	packRepo := pack.NewRepository(application.DB)
	packSvc := pack.NewService(packRepo, application.DB)

	llmClient := llm.NewClient(application.Config)
	outputRepo := output.NewRepository(application.DB)
	outputSvc := output.NewService(outputRepo, llmClient)

	r := router.NewRouter(application, authSvc, authJWT, reviewSvc, planSvc, progressSvc, packSvc, outputSvc, filesLocalRoot(application.Config))

	srv := &http.Server{
		Addr:         application.Config.HTTPAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		application.Logger.Info("starting server", "addr", application.Config.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	application.Logger.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func filesLocalRoot(cfg *config.Config) string {
	if cfg.FilesProvider == "local" {
		return cfg.FilesLocalRoot
	}
	return ""
}
