package auth

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Locale       string
	Timezone     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type LearningProfile struct {
	UserID         string
	CurrentLevel   string
	TargetDomain   string
	DailyMinutes   int
	WeeklyGoalDays int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, password_hash, locale, timezone, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.Locale, user.Timezone,
		user.CreatedAt.UTC().Format(time.RFC3339),
		user.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *Repository) CreateLearningProfile(ctx context.Context, profile *LearningProfile) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_learning_profiles (user_id, current_level, target_domain, daily_minutes, weekly_goal_days, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		profile.UserID, profile.CurrentLevel, profile.TargetDomain,
		profile.DailyMinutes, profile.WeeklyGoalDays,
		profile.CreatedAt.UTC().Format(time.RFC3339),
		profile.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, locale, timezone, is_active, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Locale, &u.Timezone, &u.IsActive, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, locale, timezone, is_active, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Locale, &u.Timezone, &u.IsActive, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return u, nil
}

func (r *Repository) GetLearningProfile(ctx context.Context, userID string) (*LearningProfile, error) {
	p := &LearningProfile{}
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, current_level, target_domain, daily_minutes, weekly_goal_days, created_at, updated_at
		 FROM user_learning_profiles WHERE user_id = ?`, userID,
	).Scan(&p.UserID, &p.CurrentLevel, &p.TargetDomain, &p.DailyMinutes, &p.WeeklyGoalDays, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return p, nil
}

func (r *Repository) UpdateLearningProfile(ctx context.Context, userID string, level, domain *string, dailyMinutes, weeklyGoalDays *int) error {
	// Build dynamic update
	query := "UPDATE user_learning_profiles SET updated_at = ? "
	args := []any{time.Now().UTC().Format(time.RFC3339)}

	if level != nil {
		query += ", current_level = ? "
		args = append(args, *level)
	}
	if domain != nil {
		query += ", target_domain = ? "
		args = append(args, *domain)
	}
	if dailyMinutes != nil {
		query += ", daily_minutes = ? "
		args = append(args, *dailyMinutes)
	}
	if weeklyGoalDays != nil {
		query += ", weekly_goal_days = ? "
		args = append(args, *weeklyGoalDays)
	}

	query += "WHERE user_id = ?"
	args = append(args, userID)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
