package progress

import (
	"context"
	"database/sql"
	"time"
)

type DailyPoint struct {
	Date           string
	MinutesLearned int
	CardsReviewed  int
	ReviewAccuracy sql.NullFloat64
}

type Summary struct {
	TotalMinutes   int
	ActiveDays     int
	ReviewAccuracy float64
	CardsReviewed  int
	StreakCount     int
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSummary(ctx context.Context, userID string, fromDate time.Time) (*Summary, error) {
	s := &Summary{}
	var accuracy sql.NullFloat64

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(minutes_learned), 0),
			COUNT(CASE WHEN minutes_learned > 0 THEN 1 END),
			AVG(CASE WHEN review_accuracy IS NOT NULL THEN review_accuracy END),
			COALESCE(SUM(cards_reviewed), 0),
			COALESCE(MAX(streak_count), 0)
		FROM progress_daily
		WHERE user_id = ? AND progress_date >= ?`,
		userID, fromDate.Format("2006-01-02"),
	).Scan(&s.TotalMinutes, &s.ActiveDays, &accuracy, &s.CardsReviewed, &s.StreakCount)
	if err != nil {
		return nil, err
	}
	if accuracy.Valid {
		s.ReviewAccuracy = accuracy.Float64
	}
	return s, nil
}

type WeeklyAggregate struct {
	RowCount            int
	TotalMinutes        int
	ActiveDays          int
	CardsReviewed       int
	CardsNew            int
	LessonsCompleted    int
	ListeningMinutes    int
	SpeakingTasks       int
	WritingTasks        int
	MaxStreak           int
	AvgReviewAccuracy   sql.NullFloat64
}

type ReviewRatingCounts struct {
	Again int
	Hard  int
	Good  int
	Easy  int
}

type DailyPointFull struct {
	Date              string
	MinutesLearned    int
	LessonsCompleted  int
	CardsNew          int
	CardsReviewed     int
	ReviewAccuracy    sql.NullFloat64
	ListeningMinutes  int
	SpeakingTasks     int
	WritingTasks      int
	StreakCount        int
}

func (r *Repository) GetWeeklyAggregate(ctx context.Context, userID, weekStart, weekEnd string) (*WeeklyAggregate, error) {
	a := &WeeklyAggregate{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(minutes_learned), 0),
			COUNT(CASE WHEN minutes_learned > 0 THEN 1 END),
			COALESCE(SUM(cards_reviewed), 0),
			COALESCE(SUM(cards_new), 0),
			COALESCE(SUM(lessons_completed), 0),
			COALESCE(SUM(listening_minutes), 0),
			COALESCE(SUM(speaking_tasks_completed), 0),
			COALESCE(SUM(writing_tasks_completed), 0),
			COALESCE(MAX(streak_count), 0),
			AVG(CASE WHEN review_accuracy IS NOT NULL THEN review_accuracy END)
		FROM progress_daily
		WHERE user_id = ? AND progress_date >= ? AND progress_date <= ?`,
		userID, weekStart, weekEnd,
	).Scan(
		&a.RowCount,
		&a.TotalMinutes, &a.ActiveDays, &a.CardsReviewed,
		&a.CardsNew, &a.LessonsCompleted, &a.ListeningMinutes,
		&a.SpeakingTasks, &a.WritingTasks, &a.MaxStreak,
		&a.AvgReviewAccuracy,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *Repository) GetReviewRatingCounts(ctx context.Context, userID, fromDate, toDate string) (*ReviewRatingCounts, error) {
	rc := &ReviewRatingCounts{}
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN rating = 'again' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN rating = 'hard'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN rating = 'good'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN rating = 'easy'  THEN 1 ELSE 0 END), 0)
		FROM review_logs
		WHERE user_id = ? AND DATE(reviewed_at) >= ? AND DATE(reviewed_at) <= ?`,
		userID, fromDate, toDate,
	).Scan(&rc.Again, &rc.Hard, &rc.Good, &rc.Easy)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func (r *Repository) GetWeeklyGoalDays(ctx context.Context, userID string) (int, error) {
	var goal int
	err := r.db.QueryRowContext(ctx, `
		SELECT weekly_goal_days FROM user_learning_profiles WHERE user_id = ?`,
		userID,
	).Scan(&goal)
	if err == sql.ErrNoRows {
		return 5, nil
	}
	if err != nil {
		return 0, err
	}
	return goal, nil
}

func (r *Repository) GetDailyFull(ctx context.Context, userID, fromDate, toDate string) ([]DailyPointFull, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
		       review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count
		FROM progress_daily
		WHERE user_id = ? AND progress_date >= ? AND progress_date <= ?
		ORDER BY progress_date ASC`,
		userID, fromDate, toDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DailyPointFull
	for rows.Next() {
		var p DailyPointFull
		if err := rows.Scan(&p.Date, &p.MinutesLearned, &p.LessonsCompleted, &p.CardsNew,
			&p.CardsReviewed, &p.ReviewAccuracy, &p.ListeningMinutes,
			&p.SpeakingTasks, &p.WritingTasks, &p.StreakCount); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

func (r *Repository) GetDaily(ctx context.Context, userID, fromDate, toDate string) ([]DailyPoint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT progress_date, minutes_learned, cards_reviewed, review_accuracy
		FROM progress_daily
		WHERE user_id = ? AND progress_date >= ? AND progress_date <= ?
		ORDER BY progress_date ASC`,
		userID, fromDate, toDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.MinutesLearned, &p.CardsReviewed, &p.ReviewAccuracy); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
