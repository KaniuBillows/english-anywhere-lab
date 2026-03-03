package review

import (
	"context"
	"database/sql"
	"time"
)

type DueCard struct {
	CardID         string
	UserCardStateID string
	FrontText      string
	BackText       string
	ExampleText    sql.NullString
	Status         string
	DueAt          time.Time
	ScheduledDays  int
	Lapses         int
	Reps           int
	Stability      float64
	Difficulty     float64
	LastReviewAt   sql.NullTime
}

type ReviewLogEntry struct {
	ID              int64
	UserID          string
	CardID          string
	UserCardStateID string
	Rating          string
	ResponseMs      *int
	StateBefore     string
	StateAfter      string
	ClientEventID   string
	IdempotencyKey  string
	ReviewedAt      time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// QueryDueCards fetches cards due for review, with priority: overdue > due > new.
// Limits: review cards by reviewLimit, new cards by newLimit.
func (r *Repository) QueryDueCards(ctx context.Context, userID string, now time.Time, limit int) ([]DueCard, error) {
	nowStr := now.UTC().Format(time.RFC3339)

	query := `
		SELECT
			ucs.card_id, ucs.id, c.front_text, c.back_text, c.example_text,
			ucs.status, ucs.due_at, ucs.scheduled_days, ucs.lapses, ucs.reps,
			COALESCE(ucs.stability, 0), COALESCE(ucs.difficulty, 0), ucs.last_review_at
		FROM user_card_states ucs
		JOIN cards c ON c.id = ucs.card_id
		WHERE ucs.user_id = ?
		  AND ucs.due_at <= ?
		  AND ucs.status != 'suspended'
		ORDER BY
			CASE
				WHEN ucs.status IN ('review','relearning') AND ucs.due_at < ? THEN 0
				WHEN ucs.status IN ('review','relearning') THEN 1
				WHEN ucs.status = 'new' THEN 2
				ELSE 3
			END,
			ucs.due_at ASC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, userID, nowStr, nowStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []DueCard
	for rows.Next() {
		var card DueCard
		var dueAt string
		var lastReview sql.NullString
		var example sql.NullString

		err := rows.Scan(
			&card.CardID, &card.UserCardStateID, &card.FrontText, &card.BackText, &example,
			&card.Status, &dueAt, &card.ScheduledDays, &card.Lapses, &card.Reps,
			&card.Stability, &card.Difficulty, &lastReview,
		)
		if err != nil {
			return nil, err
		}

		card.ExampleText = example
		card.DueAt, _ = time.Parse(time.RFC3339, dueAt)
		if lastReview.Valid {
			t, _ := time.Parse(time.RFC3339, lastReview.String)
			card.LastReviewAt = sql.NullTime{Time: t, Valid: true}
		}

		cards = append(cards, card)
	}

	return cards, rows.Err()
}

// GetUserCardState fetches a single card state by ID.
func (r *Repository) GetUserCardState(ctx context.Context, stateID string) (*DueCard, error) {
	var card DueCard
	var dueAt string
	var lastReview sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT card_id, id, status, due_at, scheduled_days, lapses, reps,
			   COALESCE(stability, 0), COALESCE(difficulty, 0), last_review_at
		FROM user_card_states WHERE id = ?`, stateID,
	).Scan(
		&card.CardID, &card.UserCardStateID, &card.Status, &dueAt,
		&card.ScheduledDays, &card.Lapses, &card.Reps,
		&card.Stability, &card.Difficulty, &lastReview,
	)
	if err != nil {
		return nil, err
	}
	card.DueAt, _ = time.Parse(time.RFC3339, dueAt)
	if lastReview.Valid {
		t, _ := time.Parse(time.RFC3339, lastReview.String)
		card.LastReviewAt = sql.NullTime{Time: t, Valid: true}
	}
	return &card, nil
}

// UpdateCardState updates card state after review.
func (r *Repository) UpdateCardState(ctx context.Context, tx *sql.Tx, stateID, status string, dueAt time.Time, reps, lapses, scheduledDays, elapsedDays int, stability, difficulty float64, reviewedAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE user_card_states
		SET status = ?, due_at = ?, reps = ?, lapses = ?, scheduled_days = ?,
		    elapsed_days = ?, stability = ?, difficulty = ?, last_review_at = ?, updated_at = ?
		WHERE id = ?`,
		status,
		dueAt.UTC().Format(time.RFC3339),
		reps, lapses, scheduledDays, elapsedDays,
		stability, difficulty,
		reviewedAt.UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		stateID,
	)
	return err
}

// InsertReviewLog records a review event.
func (r *Repository) InsertReviewLog(ctx context.Context, tx *sql.Tx, entry *ReviewLogEntry) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO review_logs (user_id, card_id, user_card_state_id, rating, response_ms,
			state_before, state_after, client_event_id, idempotency_key, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.UserID, entry.CardID, entry.UserCardStateID, entry.Rating, entry.ResponseMs,
		entry.StateBefore, entry.StateAfter, entry.ClientEventID, entry.IdempotencyKey,
		entry.ReviewedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// CheckIdempotencyKey returns the existing review log if idempotency key was already used.
func (r *Repository) CheckIdempotencyKey(ctx context.Context, key string) (*ReviewLogEntry, error) {
	var entry ReviewLogEntry
	var reviewedAt string
	var responseMs sql.NullInt64

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, card_id, user_card_state_id, rating, response_ms,
			   state_before, state_after, client_event_id, idempotency_key, reviewed_at
		FROM review_logs WHERE idempotency_key = ?`, key,
	).Scan(
		&entry.ID, &entry.UserID, &entry.CardID, &entry.UserCardStateID,
		&entry.Rating, &responseMs,
		&entry.StateBefore, &entry.StateAfter, &entry.ClientEventID,
		&entry.IdempotencyKey, &reviewedAt,
	)
	if err != nil {
		return nil, err
	}
	entry.ReviewedAt, _ = time.Parse(time.RFC3339, reviewedAt)
	if responseMs.Valid {
		v := int(responseMs.Int64)
		entry.ResponseMs = &v
	}
	return &entry, nil
}

// BeginTx starts a transaction.
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// GetDueCount returns total number of cards due for the user.
func (r *Repository) GetDueCount(ctx context.Context, userID string, now time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_card_states
		WHERE user_id = ? AND due_at <= ? AND status != 'suspended'`,
		userID, now.UTC().Format(time.RFC3339),
	).Scan(&count)
	return count, err
}
