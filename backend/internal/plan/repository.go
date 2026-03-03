package plan

import (
	"context"
	"database/sql"
	"time"
)

type Plan struct {
	ID        string
	UserID    string
	WeekStart string
	CreatedAt time.Time
}

type PlanTask struct {
	ID               string
	PlanID           string
	TaskDate         string
	TaskType         string
	Title            string
	Status           string
	EstimatedMinutes int
	CompletedAt      sql.NullString
	DurationSeconds  sql.NullInt64
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePlan(ctx context.Context, plan *Plan) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO plans (id, user_id, week_start, created_at) VALUES (?, ?, ?, ?)`,
		plan.ID, plan.UserID, plan.WeekStart,
		plan.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *Repository) CreatePlanTask(ctx context.Context, task *PlanTask) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO plan_tasks (id, plan_id, task_date, task_type, title, status, estimated_minutes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.PlanID, task.TaskDate, task.TaskType, task.Title, task.Status, task.EstimatedMinutes,
	)
	return err
}

func (r *Repository) GetPlanByUserAndWeek(ctx context.Context, userID, weekStart string) (*Plan, error) {
	p := &Plan{}
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, week_start, created_at FROM plans WHERE user_id = ? AND week_start = ?`,
		userID, weekStart,
	).Scan(&p.ID, &p.UserID, &p.WeekStart, &createdAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return p, nil
}

func (r *Repository) GetLatestPlan(ctx context.Context, userID string) (*Plan, error) {
	p := &Plan{}
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, week_start, created_at FROM plans WHERE user_id = ? ORDER BY week_start DESC LIMIT 1`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.WeekStart, &createdAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return p, nil
}

func (r *Repository) GetTasksByPlanAndDate(ctx context.Context, planID, date string) ([]PlanTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, plan_id, task_date, task_type, title, status, estimated_minutes, completed_at, duration_seconds
		 FROM plan_tasks WHERE plan_id = ? AND task_date = ? ORDER BY ROWID`,
		planID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []PlanTask
	for rows.Next() {
		var t PlanTask
		err := rows.Scan(&t.ID, &t.PlanID, &t.TaskDate, &t.TaskType, &t.Title, &t.Status,
			&t.EstimatedMinutes, &t.CompletedAt, &t.DurationSeconds)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *Repository) GetTasksByPlan(ctx context.Context, planID string) ([]PlanTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, plan_id, task_date, task_type, title, status, estimated_minutes, completed_at, duration_seconds
		 FROM plan_tasks WHERE plan_id = ? ORDER BY task_date, ROWID`,
		planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []PlanTask
	for rows.Next() {
		var t PlanTask
		err := rows.Scan(&t.ID, &t.PlanID, &t.TaskDate, &t.TaskType, &t.Title, &t.Status,
			&t.EstimatedMinutes, &t.CompletedAt, &t.DurationSeconds)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *Repository) CompleteTask(ctx context.Context, taskID, completedAt string, durationSeconds *int) error {
	query := `UPDATE plan_tasks SET status = 'completed', completed_at = ?`
	args := []any{completedAt}

	if durationSeconds != nil {
		query += ", duration_seconds = ?"
		args = append(args, *durationSeconds)
	}

	query += " WHERE id = ?"
	args = append(args, taskID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) GetTask(ctx context.Context, taskID string) (*PlanTask, error) {
	var t PlanTask
	err := r.db.QueryRowContext(ctx,
		`SELECT id, plan_id, task_date, task_type, title, status, estimated_minutes, completed_at, duration_seconds
		 FROM plan_tasks WHERE id = ?`, taskID,
	).Scan(&t.ID, &t.PlanID, &t.TaskDate, &t.TaskType, &t.Title, &t.Status,
		&t.EstimatedMinutes, &t.CompletedAt, &t.DurationSeconds)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTaskWithOwner fetches a task and verifies it belongs to the given plan and user.
func (r *Repository) GetTaskWithOwner(ctx context.Context, taskID, planID, userID string) (*PlanTask, error) {
	var t PlanTask
	err := r.db.QueryRowContext(ctx,
		`SELECT pt.id, pt.plan_id, pt.task_date, pt.task_type, pt.title, pt.status,
		        pt.estimated_minutes, pt.completed_at, pt.duration_seconds
		 FROM plan_tasks pt
		 JOIN plans p ON p.id = pt.plan_id
		 WHERE pt.id = ? AND pt.plan_id = ? AND p.user_id = ?`,
		taskID, planID, userID,
	).Scan(&t.ID, &t.PlanID, &t.TaskDate, &t.TaskType, &t.Title, &t.Status,
		&t.EstimatedMinutes, &t.CompletedAt, &t.DurationSeconds)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
