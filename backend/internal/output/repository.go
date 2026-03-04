package output

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// OutputTask represents a row in the output_tasks table.
type OutputTask struct {
	ID              string
	LessonID        sql.NullString
	TaskType        string
	PromptText      string
	ReferenceAnswer sql.NullString
	Level           sql.NullString
	CreatedAt       string
}

// OutputSubmission represents a row in the output_submissions table.
type OutputSubmission struct {
	ID          string
	UserID      string
	TaskID      string
	AnswerText  sql.NullString
	AudioURL    sql.NullString
	AIFeedback  sql.NullString
	Score       sql.NullFloat64
	SubmittedAt string
}

// Repository provides data access for output_tasks and output_submissions.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new output Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListByLesson returns writing tasks for a given lesson.
func (r *Repository) ListByLesson(ctx context.Context, lessonID string) ([]OutputTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, lesson_id, task_type, prompt_text, reference_answer, level, created_at
		 FROM output_tasks
		 WHERE lesson_id = ? AND task_type = 'writing'
		 ORDER BY created_at`,
		lessonID,
	)
	if err != nil {
		return nil, fmt.Errorf("query output tasks: %w", err)
	}
	defer rows.Close()

	var tasks []OutputTask
	for rows.Next() {
		var t OutputTask
		if err := rows.Scan(&t.ID, &t.LessonID, &t.TaskType, &t.PromptText, &t.ReferenceAnswer, &t.Level, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan output task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// GetTask returns a single output task by ID.
func (r *Repository) GetTask(ctx context.Context, taskID string) (*OutputTask, error) {
	var t OutputTask
	err := r.db.QueryRowContext(ctx,
		`SELECT id, lesson_id, task_type, prompt_text, reference_answer, level, created_at
		 FROM output_tasks WHERE id = ?`,
		taskID,
	).Scan(&t.ID, &t.LessonID, &t.TaskType, &t.PromptText, &t.ReferenceAnswer, &t.Level, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// InsertSubmissionWithProgress inserts a submission (with feedback already populated)
// and increments the user's daily writing_tasks_completed counter in a single transaction.
func (r *Repository) InsertSubmissionWithProgress(ctx context.Context, sub *OutputSubmission) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO output_submissions (id, user_id, task_id, answer_text, audio_url, ai_feedback, score, submitted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.ID, sub.UserID, sub.TaskID, sub.AnswerText, sub.AudioURL, sub.AIFeedback, sub.Score, sub.SubmittedAt,
	)
	if err != nil {
		return fmt.Errorf("insert submission: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO progress_daily (user_id, progress_date, writing_tasks_completed, created_at, updated_at)
		 VALUES (?, ?, 1, ?, ?)
		 ON CONFLICT(user_id, progress_date) DO UPDATE SET
		   writing_tasks_completed = writing_tasks_completed + 1,
		   updated_at = ?`,
		sub.UserID, today, now, now, now,
	)
	if err != nil {
		return fmt.Errorf("increment progress: %w", err)
	}

	return tx.Commit()
}

// GetSubmission returns a submission scoped to a specific user.
func (r *Repository) GetSubmission(ctx context.Context, submissionID, userID string) (*OutputSubmission, error) {
	var s OutputSubmission
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, task_id, answer_text, audio_url, ai_feedback, score, submitted_at
		 FROM output_submissions WHERE id = ? AND user_id = ?`,
		submissionID, userID,
	).Scan(&s.ID, &s.UserID, &s.TaskID, &s.AnswerText, &s.AudioURL, &s.AIFeedback, &s.Score, &s.SubmittedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
