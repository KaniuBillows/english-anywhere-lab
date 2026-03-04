package pack

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/google/uuid"
)

// Model structs

type Pack struct {
	ID               string
	Source           string
	Title            string
	Description      sql.NullString
	Domain           string
	Level            string
	EstimatedMinutes int
	CreatedByUserID  sql.NullString
	CreatedAt        time.Time
}

type Lesson struct {
	ID               string
	PackID           string
	Title            string
	LessonType       string
	Position         int
	EstimatedMinutes int
	CreatedAt        time.Time
}

type Card struct {
	ID        string
	LessonID  sql.NullString
	FrontText string
	BackText  string
}

type GenerationJob struct {
	ID              string
	UserID          string
	JobType         string
	Domain          string
	Level           string
	TemplateVersion string
	RequestPayload  string
	ResponsePayload sql.NullString
	Status          string
	ErrorMessage    sql.NullString
	CreatedAt       time.Time
	StartedAt       sql.NullString
	FinishedAt      sql.NullString
	PackID          sql.NullString // derived from response_payload if needed
}

type OutputTask struct {
	ID              string
	LessonID        string
	TaskType        string
	PromptText      string
	ReferenceAnswer sql.NullString
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListPacks(ctx context.Context, domain, level, source string, page, pageSize int) ([]Pack, int, error) {
	var conditions []string
	var args []any

	if domain != "" {
		conditions = append(conditions, "domain = ?")
		args = append(args, domain)
	}
	if level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, level)
	}
	if source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, source)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := "SELECT COUNT(*) FROM resource_packs" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count packs: %w", err)
	}

	// Data query
	offset := (page - 1) * pageSize
	dataQuery := "SELECT id, source, title, description, domain, level, estimated_minutes, created_by_user_id, created_at FROM resource_packs" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	dataArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query packs: %w", err)
	}
	defer rows.Close()

	var packs []Pack
	for rows.Next() {
		var p Pack
		var createdAt string
		if err := rows.Scan(&p.ID, &p.Source, &p.Title, &p.Description, &p.Domain, &p.Level, &p.EstimatedMinutes, &p.CreatedByUserID, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan pack: %w", err)
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		packs = append(packs, p)
	}
	return packs, total, rows.Err()
}

func (r *Repository) GetPack(ctx context.Context, packID string) (*Pack, error) {
	var p Pack
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		"SELECT id, source, title, description, domain, level, estimated_minutes, created_by_user_id, created_at FROM resource_packs WHERE id = ?",
		packID,
	).Scan(&p.ID, &p.Source, &p.Title, &p.Description, &p.Domain, &p.Level, &p.EstimatedMinutes, &p.CreatedByUserID, &createdAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &p, nil
}

func (r *Repository) GetLessonsByPack(ctx context.Context, packID string) ([]Lesson, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, pack_id, title, lesson_type, position, estimated_minutes, created_at FROM lessons WHERE pack_id = ? ORDER BY position",
		packID,
	)
	if err != nil {
		return nil, fmt.Errorf("query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []Lesson
	for rows.Next() {
		var l Lesson
		var createdAt string
		if err := rows.Scan(&l.ID, &l.PackID, &l.Title, &l.LessonType, &l.Position, &l.EstimatedMinutes, &createdAt); err != nil {
			return nil, fmt.Errorf("scan lesson: %w", err)
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		lessons = append(lessons, l)
	}
	return lessons, rows.Err()
}

func (r *Repository) GetCardsByPack(ctx context.Context, packID string) ([]Card, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.lesson_id, c.front_text, c.back_text
		 FROM cards c
		 JOIN lessons l ON l.id = c.lesson_id
		 WHERE l.pack_id = ?`,
		packID,
	)
	if err != nil {
		return nil, fmt.Errorf("query cards by pack: %w", err)
	}
	defer rows.Close()

	var cards []Card
	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.LessonID, &c.FrontText, &c.BackText); err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (r *Repository) CreateGenerationJob(ctx context.Context, job *GenerationJob) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ai_generation_jobs (id, user_id, job_type, domain, level, template_version, request_payload, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.UserID, job.JobType, job.Domain, job.Level, job.TemplateVersion, job.RequestPayload, job.Status, job.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *Repository) GetGenerationJob(ctx context.Context, jobID, userID string) (*GenerationJob, error) {
	var j GenerationJob
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, job_type, domain, level, template_version, request_payload, response_payload, status, error_message, created_at, started_at, finished_at
		 FROM ai_generation_jobs WHERE id = ? AND user_id = ?`,
		jobID, userID,
	).Scan(&j.ID, &j.UserID, &j.JobType, &j.Domain, &j.Level, &j.TemplateVersion,
		&j.RequestPayload, &j.ResponsePayload, &j.Status, &j.ErrorMessage,
		&createdAt, &j.StartedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &j, nil
}

// ClaimNextJob atomically claims the oldest queued job by setting status=running.
func (r *Repository) ClaimNextJob(ctx context.Context) (*GenerationJob, error) {
	var j GenerationJob
	var createdAt string
	now := time.Now().UTC().Format(time.RFC3339)

	err := r.db.QueryRowContext(ctx,
		`UPDATE ai_generation_jobs
		 SET status = 'running', started_at = ?
		 WHERE id = (
		   SELECT id FROM ai_generation_jobs WHERE status = 'queued' ORDER BY created_at ASC LIMIT 1
		 )
		 RETURNING id, user_id, job_type, domain, level, template_version, request_payload, response_payload, status, error_message, created_at, started_at, finished_at`,
		now,
	).Scan(&j.ID, &j.UserID, &j.JobType, &j.Domain, &j.Level, &j.TemplateVersion,
		&j.RequestPayload, &j.ResponsePayload, &j.Status, &j.ErrorMessage,
		&createdAt, &j.StartedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &j, nil
}

// UpdateJobStatus updates a job's status, response_payload, and error_message.
func (r *Repository) UpdateJobStatus(ctx context.Context, jobID, status, responsePayload, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var respPayload, errMsg sql.NullString
	if responsePayload != "" {
		respPayload = sql.NullString{String: responsePayload, Valid: true}
	}
	if errorMessage != "" {
		errMsg = sql.NullString{String: errorMessage, Valid: true}
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE ai_generation_jobs SET status = ?, finished_at = ?, response_payload = ?, error_message = ? WHERE id = ?`,
		status, now, respPayload, errMsg, jobID,
	)
	return err
}

// CountUserJobsToday counts the number of generation jobs created today (UTC) by a user.
func (r *Repository) CountUserJobsToday(ctx context.Context, userID string) (int, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ai_generation_jobs WHERE user_id = ? AND created_at >= ?`,
		userID, startOfDay,
	).Scan(&count)
	return count, err
}

// InsertPackWithContent inserts a resource_pack with its lessons, cards, and output_tasks
// within the provided transaction.
func (r *Repository) InsertPackWithContent(ctx context.Context, tx *sql.Tx, userID string, gen *llm.GeneratedPack, level, domain string) (string, error) {
	packID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := tx.ExecContext(ctx,
		`INSERT INTO resource_packs (id, source, title, description, domain, level, estimated_minutes, created_by_user_id, created_at)
		 VALUES (?, 'ai', ?, ?, ?, ?, ?, ?, ?)`,
		packID, gen.Title, gen.Description, domain, level, gen.EstimatedMinutes, userID, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert pack: %w", err)
	}

	for _, lesson := range gen.Lessons {
		lessonID := uuid.New().String()
		_, err := tx.ExecContext(ctx,
			`INSERT INTO lessons (id, pack_id, title, lesson_type, position, estimated_minutes, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			lessonID, packID, lesson.Title, lesson.LessonType, lesson.Position, lesson.EstimatedMinutes, now,
		)
		if err != nil {
			return "", fmt.Errorf("insert lesson: %w", err)
		}

		for _, card := range lesson.Cards {
			cardID := uuid.New().String()
			var exampleText sql.NullString
			if card.ExampleText != "" {
				exampleText = sql.NullString{String: card.ExampleText, Valid: true}
			}
			_, err := tx.ExecContext(ctx,
				`INSERT INTO cards (id, lesson_id, front_text, back_text, example_text, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				cardID, lessonID, card.FrontText, card.BackText, exampleText, now,
			)
			if err != nil {
				return "", fmt.Errorf("insert card: %w", err)
			}
		}

		for _, task := range lesson.OutputTasks {
			taskID := uuid.New().String()
			var refAnswer sql.NullString
			if task.ReferenceAnswer != "" {
				refAnswer = sql.NullString{String: task.ReferenceAnswer, Valid: true}
			}
			_, err := tx.ExecContext(ctx,
				`INSERT INTO output_tasks (id, lesson_id, task_type, prompt_text, reference_answer, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				taskID, lessonID, task.TaskType, task.PromptText, refAnswer, now,
			)
			if err != nil {
				return "", fmt.Errorf("insert output task: %w", err)
			}
		}
	}

	return packID, nil
}
