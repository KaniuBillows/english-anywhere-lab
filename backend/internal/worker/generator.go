package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
)

type Generator struct {
	repo         *pack.Repository
	llmClient    *llm.Client
	db           *sql.DB
	logger       *slog.Logger
	pollInterval time.Duration
}

func NewGenerator(repo *pack.Repository, llmClient *llm.Client, db *sql.DB, logger *slog.Logger) *Generator {
	return &Generator{
		repo:         repo,
		llmClient:    llmClient,
		db:           db,
		logger:       logger,
		pollInterval: 5 * time.Second,
	}
}

// Run polls for queued jobs until ctx is cancelled.
func (g *Generator) Run(ctx context.Context) {
	g.logger.Info("starting generator")

	for {
		select {
		case <-ctx.Done():
			g.logger.Info("generator stopped")
			return
		default:
		}

		job, err := g.repo.ClaimNextJob(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(g.pollInterval):
				continue
			}
		}
		if err != nil {
			g.logger.Error("claim job failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(g.pollInterval):
				continue
			}
		}

		g.logger.Info("processing job", "job_id", job.ID, "user_id", job.UserID)

		if err := g.ProcessJob(ctx, job); err != nil {
			g.logger.Error("job failed", "job_id", job.ID, "error", err)
			_ = g.repo.UpdateJobStatus(ctx, job.ID, "failed", "", err.Error())
		} else {
			g.logger.Info("job completed", "job_id", job.ID)
		}
	}
}

type requestPayload struct {
	Level        string   `json:"level"`
	Domain       string   `json:"domain"`
	DailyMinutes int      `json:"daily_minutes"`
	Days         int      `json:"days"`
	FocusSkills  []string `json:"focus_skills"`
}

// processJob handles a single job: call LLM → validate → write DB → update status.
func (g *Generator) ProcessJob(ctx context.Context, job *pack.GenerationJob) error {
	var payload requestPayload
	if err := json.Unmarshal([]byte(job.RequestPayload), &payload); err != nil {
		return fmt.Errorf("unmarshal request payload: %w", err)
	}

	messages := llm.BuildPrompt(payload.Level, payload.Domain, payload.DailyMinutes, payload.Days, payload.FocusSkills)

	raw, err := g.llmClient.ChatCompletion(ctx, messages)
	if err != nil {
		return fmt.Errorf("LLM call: %w", err)
	}

	gen, err := llm.ParseAndValidate(raw)
	if err != nil {
		return fmt.Errorf("parse/validate LLM response: %w", err)
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	packID, err := g.repo.InsertPackWithContent(ctx, tx, job.UserID, gen, payload.Level, payload.Domain)
	if err != nil {
		return fmt.Errorf("insert pack content: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	if err := g.repo.UpdateJobStatus(ctx, job.ID, "success", raw, ""); err != nil {
		return fmt.Errorf("update job status to success: %w", err)
	}
	g.logger.Info("pack created", "job_id", job.ID, "pack_id", packID)
	return nil
}
