package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/storage"
	"github.com/bennyshi/english-anywhere-lab/internal/tts"
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

	g, ctx := errgroup.WithContext(ctx)

	// Pack generation worker (always runs)
	g.Go(func() error {
		generator.Run(ctx)
		return nil
	})

	// TTS worker (runs if enabled)
	if application.Config.TTSEnabled {
		objStore, err := buildObjectStore(application.Config)
		if err != nil {
			application.Logger.Error("failed to build object store", "error", err)
			os.Exit(1)
		}
		ttsProvider := buildTTSProvider(application.Config)
		ttsSvc := tts.NewService(ttsProvider, objStore, tts.TTSConfig{
			Voice:        application.Config.TTSVoice,
			Speed:        application.Config.TTSSpeed,
			Format:       application.Config.TTSOutputFormat,
			SampleRate:   application.Config.TTSSampleRate,
			MaxTextChars: application.Config.TTSMaxTextChars,
		})
		ttsWorker := worker.NewTTSWorker(application.DB, ttsSvc, application.Logger, application.Config.TTSRetryMax)

		g.Go(func() error {
			ttsWorker.Run(ctx)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		application.Logger.Error("worker error", "error", err)
	}
	application.Logger.Info("worker shut down")
}

func buildObjectStore(cfg *config.Config) (storage.ObjectStore, error) {
	switch cfg.FilesProvider {
	case "s3":
		return storage.NewS3Store(storage.S3Config{
			Endpoint:       cfg.FilesS3Endpoint,
			Region:         cfg.FilesS3Region,
			Bucket:         cfg.FilesS3Bucket,
			AccessKey:      cfg.FilesS3AccessKey,
			SecretKey:      cfg.FilesS3SecretKey,
			ForcePathStyle: cfg.FilesS3ForcePathStyle,
			PublicURL:      cfg.FilesS3PublicURL,
		})
	default:
		return storage.NewLocalStore(cfg.FilesLocalRoot, cfg.FilesBaseURL)
	}
}

func buildTTSProvider(cfg *config.Config) tts.TTSProvider {
	switch cfg.TTSProvider {
	default:
		return tts.NewStubProvider()
	}
}
