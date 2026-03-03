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
