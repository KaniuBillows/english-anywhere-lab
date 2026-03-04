package pack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPackNotFound = errors.New("pack not found")
	ErrJobNotFound  = errors.New("generation job not found")
)

type Service struct {
	repo *Repository
	db   *sql.DB
}

func NewService(repo *Repository, db *sql.DB) *Service {
	return &Service{repo: repo, db: db}
}

// ListInput holds query parameters for listing packs.
type ListInput struct {
	Domain   string
	Level    string
	Source   string
	Page     int
	PageSize int
}

// ListResult holds the paginated pack list response data.
type ListResult struct {
	Packs    []Pack
	Page     int
	PageSize int
	Total    int
}

func (s *Service) ListPacks(ctx context.Context, input ListInput) (*ListResult, error) {
	packs, total, err := s.repo.ListPacks(ctx, input.Domain, input.Level, input.Source, input.Page, input.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list packs: %w", err)
	}
	return &ListResult{
		Packs:    packs,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}

// DetailResult holds a pack with its lessons.
type DetailResult struct {
	Pack    Pack
	Lessons []Lesson
}

func (s *Service) GetDetail(ctx context.Context, packID string) (*DetailResult, error) {
	p, err := s.repo.GetPack(ctx, packID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPackNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pack: %w", err)
	}

	lessons, err := s.repo.GetLessonsByPack(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("get lessons: %w", err)
	}

	return &DetailResult{Pack: *p, Lessons: lessons}, nil
}

func (s *Service) Enroll(ctx context.Context, userID, packID string) error {
	// Verify pack exists
	_, err := s.repo.GetPack(ctx, packID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPackNotFound
	}
	if err != nil {
		return fmt.Errorf("get pack: %w", err)
	}

	// Fetch cards belonging to this pack
	cards, err := s.repo.GetCardsByPack(ctx, packID)
	if err != nil {
		return fmt.Errorf("get cards: %w", err)
	}

	if len(cards) == 0 {
		return nil // nothing to enroll
	}

	// Transaction: insert user_card_states for each card (idempotent)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range cards {
		stateID := uuid.New().String()
		_, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO user_card_states (id, user_id, card_id, status, due_at, reps, lapses, scheduled_days, created_at, updated_at)
			 VALUES (?, ?, ?, 'new', ?, 0, 0, 0, ?, ?)`,
			stateID, userID, c.ID, now, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert card state: %w", err)
		}
	}

	return tx.Commit()
}

// GenerateInput holds parameters for creating a generation job.
type GenerateInput struct {
	UserID       string
	Level        string
	Domain       string
	DailyMinutes int
	Days         int
	FocusSkills  []string
}

func (s *Service) CreateGenerationJob(ctx context.Context, input GenerateInput) (*GenerationJob, error) {
	payload, _ := json.Marshal(map[string]any{
		"level":         input.Level,
		"domain":        input.Domain,
		"daily_minutes": input.DailyMinutes,
		"days":          input.Days,
		"focus_skills":  input.FocusSkills,
	})

	job := &GenerationJob{
		ID:              uuid.New().String(),
		UserID:          input.UserID,
		JobType:         "pack_generation",
		Domain:          input.Domain,
		Level:           input.Level,
		TemplateVersion: "v1",
		RequestPayload:  string(payload),
		Status:          "queued",
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.repo.CreateGenerationJob(ctx, job); err != nil {
		return nil, fmt.Errorf("create generation job: %w", err)
	}

	return job, nil
}

func (s *Service) GetGenerationJob(ctx context.Context, jobID, userID string) (*GenerationJob, error) {
	job, err := s.repo.GetGenerationJob(ctx, jobID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get generation job: %w", err)
	}
	return job, nil
}
