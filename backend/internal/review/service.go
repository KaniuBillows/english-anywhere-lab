package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/scheduler"
)

var (
	ErrCardStateNotFound = errors.New("card state not found")
	ErrDuplicateReview   = errors.New("duplicate review submission")
)

type Service struct {
	repo *Repository
	fsrs *scheduler.FSRS
}

func NewService(repo *Repository, fsrs *scheduler.FSRS) *Service {
	return &Service{repo: repo, fsrs: fsrs}
}

type ReviewQueueResult struct {
	DueCount int
	Cards    []DueCard
}

func (s *Service) GetDueCards(ctx context.Context, userID string, limit int) (*ReviewQueueResult, error) {
	now := time.Now().UTC()

	count, err := s.repo.GetDueCount(ctx, userID, now)
	if err != nil {
		return nil, fmt.Errorf("get due count: %w", err)
	}

	cards, err := s.repo.QueryDueCards(ctx, userID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("query due cards: %w", err)
	}

	return &ReviewQueueResult{DueCount: count, Cards: cards}, nil
}

type SubmitInput struct {
	UserID         string
	CardID         string
	UserCardStateID string
	Rating         string
	ReviewedAt     time.Time
	ResponseMs     *int
	ClientEventID  string
	IdempotencyKey string
}

type SubmitResult struct {
	Accepted      bool
	CardID        string
	NextDueAt     time.Time
	ScheduledDays int
	Status        string
}

func (s *Service) SubmitReview(ctx context.Context, input SubmitInput) (*SubmitResult, error) {
	// Check idempotency
	if input.IdempotencyKey != "" {
		existing, err := s.repo.CheckIdempotencyKey(ctx, input.IdempotencyKey)
		if err == nil && existing != nil {
			// Parse state_after for cached result
			var after map[string]any
			json.Unmarshal([]byte(existing.StateAfter), &after)
			dueAt, _ := time.Parse(time.RFC3339, fmt.Sprintf("%v", after["due_at"]))
			scheduledDays := 0
			if v, ok := after["scheduled_days"].(float64); ok {
				scheduledDays = int(v)
			}
			status := fmt.Sprintf("%v", after["status"])
			return &SubmitResult{
				Accepted:      true,
				CardID:        existing.CardID,
				NextDueAt:     dueAt,
				ScheduledDays: scheduledDays,
				Status:        status,
			}, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("check idempotency: %w", err)
		}
	}

	// Get current card state
	cardState, err := s.repo.GetUserCardState(ctx, input.UserCardStateID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCardStateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get card state: %w", err)
	}

	// Build FSRS input
	fsrsState := scheduler.CardState{
		Status:        scheduler.CardStatus(cardState.Status),
		DueAt:         cardState.DueAt,
		Reps:          cardState.Reps,
		Lapses:        cardState.Lapses,
		Stability:     cardState.Stability,
		Difficulty:    cardState.Difficulty,
		ScheduledDays: cardState.ScheduledDays,
	}
	if cardState.LastReviewAt.Valid {
		fsrsState.LastReviewAt = cardState.LastReviewAt.Time
	}

	result := s.fsrs.Schedule(fsrsState, scheduler.Rating(input.Rating), input.ReviewedAt)

	// Serialize state snapshots
	stateBefore, _ := json.Marshal(map[string]any{
		"status":         cardState.Status,
		"due_at":         cardState.DueAt.UTC().Format(time.RFC3339),
		"reps":           cardState.Reps,
		"lapses":         cardState.Lapses,
		"scheduled_days": cardState.ScheduledDays,
		"stability":      cardState.Stability,
		"difficulty":     cardState.Difficulty,
	})
	stateAfter, _ := json.Marshal(map[string]any{
		"status":         string(result.Status),
		"due_at":         result.DueAt.UTC().Format(time.RFC3339),
		"reps":           result.Reps,
		"lapses":         result.Lapses,
		"scheduled_days": result.ScheduledDays,
		"stability":      result.Stability,
		"difficulty":     result.Difficulty,
	})

	// Single transaction: insert log + update state
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	logEntry := &ReviewLogEntry{
		UserID:          input.UserID,
		CardID:          input.CardID,
		UserCardStateID: input.UserCardStateID,
		Rating:          input.Rating,
		ResponseMs:      input.ResponseMs,
		StateBefore:     string(stateBefore),
		StateAfter:      string(stateAfter),
		ClientEventID:   input.ClientEventID,
		IdempotencyKey:  input.IdempotencyKey,
		ReviewedAt:      input.ReviewedAt,
	}

	if err := s.repo.InsertReviewLog(ctx, tx, logEntry); err != nil {
		return nil, fmt.Errorf("insert review log: %w", err)
	}

	if err := s.repo.UpdateCardState(ctx, tx, input.UserCardStateID,
		string(result.Status), result.DueAt,
		result.Reps, result.Lapses, result.ScheduledDays, result.ElapsedDays,
		result.Stability, result.Difficulty,
		input.ReviewedAt,
	); err != nil {
		return nil, fmt.Errorf("update card state: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &SubmitResult{
		Accepted:      true,
		CardID:        input.CardID,
		NextDueAt:     result.DueAt,
		ScheduledDays: result.ScheduledDays,
		Status:        string(result.Status),
	}, nil
}
