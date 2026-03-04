package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/tts"
)

// TTSWorker polls for TTS generation jobs and processes them.
type TTSWorker struct {
	db           *sql.DB
	ttsSvc       *tts.Service
	logger       *slog.Logger
	pollInterval time.Duration
	maxRetries   int
}

// TTSJob represents a TTS generation job from ai_generation_jobs.
type TTSJob struct {
	ID             string
	UserID         string
	RequestPayload string
	RetryCount     int
}

type ttsJobPayload struct {
	CardID string `json:"card_id"`
	Text   string `json:"text"`
	Field  string `json:"field"`
}

// NewTTSWorker creates a new TTSWorker.
func NewTTSWorker(db *sql.DB, ttsSvc *tts.Service, logger *slog.Logger, maxRetries int) *TTSWorker {
	return &TTSWorker{
		db:           db,
		ttsSvc:       ttsSvc,
		logger:       logger,
		pollInterval: 5 * time.Second,
		maxRetries:   maxRetries,
	}
}

// Run polls for queued TTS jobs until ctx is cancelled.
func (w *TTSWorker) Run(ctx context.Context) {
	w.logger.Info("starting TTS worker")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("TTS worker stopped")
			return
		default:
		}

		job, err := w.ClaimNextTTSJob(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.pollInterval):
				continue
			}
		}
		if err != nil {
			w.logger.Error("claim TTS job failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.pollInterval):
				continue
			}
		}

		w.logger.Info("processing TTS job", "job_id", job.ID)

		if err := w.ProcessJob(ctx, job); err != nil {
			w.logger.Error("TTS job failed", "job_id", job.ID, "error", err)
			w.updateJobStatus(ctx, job.ID, "failed", err.Error())
		} else {
			w.logger.Info("TTS job completed", "job_id", job.ID)
			w.updateJobStatus(ctx, job.ID, "success", "")
		}
	}
}

// ClaimNextTTSJob atomically claims the oldest queued TTS job.
func (w *TTSWorker) ClaimNextTTSJob(ctx context.Context) (*TTSJob, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var j TTSJob
	err := w.db.QueryRowContext(ctx,
		`UPDATE ai_generation_jobs
		 SET status = 'running', started_at = ?
		 WHERE id = (
		   SELECT id FROM ai_generation_jobs
		   WHERE job_type = 'tts_generation' AND status = 'queued'
		   ORDER BY created_at ASC LIMIT 1
		 )
		 RETURNING id, user_id, request_payload`,
		now,
	).Scan(&j.ID, &j.UserID, &j.RequestPayload)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ProcessJob handles a single TTS job: parse payload → synthesize → store → backfill audio_url.
func (w *TTSWorker) ProcessJob(ctx context.Context, job *TTSJob) error {
	var payload ttsJobPayload
	if err := json.Unmarshal([]byte(job.RequestPayload), &payload); err != nil {
		return fmt.Errorf("unmarshal TTS payload: %w", err)
	}

	if payload.CardID == "" || payload.Text == "" {
		return fmt.Errorf("invalid TTS payload: card_id and text are required")
	}

	audioURL, err := w.ttsSvc.SynthesizeAndStore(ctx, payload.Text)
	if err != nil {
		return fmt.Errorf("synthesize and store: %w", err)
	}

	res, err := w.db.ExecContext(ctx,
		`UPDATE cards SET audio_url = ? WHERE id = ?`,
		audioURL, payload.CardID,
	)
	if err != nil {
		return fmt.Errorf("update card audio_url: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("card %s not found", payload.CardID)
	}

	return nil
}

func (w *TTSWorker) updateJobStatus(ctx context.Context, jobID, status, errorMessage string) {
	now := time.Now().UTC().Format(time.RFC3339)
	var errMsg sql.NullString
	if errorMessage != "" {
		errMsg = sql.NullString{String: errorMessage, Valid: true}
	}
	_, err := w.db.ExecContext(ctx,
		`UPDATE ai_generation_jobs SET status = ?, finished_at = ?, error_message = ? WHERE id = ?`,
		status, now, errMsg, jobID,
	)
	if err != nil {
		w.logger.Error("failed to update TTS job status", "job_id", jobID, "error", err)
	}
}
