package sync

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// SyncEvent represents a client event stored for dedup.
type SyncEvent struct {
	ID            int64
	UserID        string
	ClientEventID string
	EventType     string
	OccurredAt    time.Time
	Payload       string
	Status        string
	Reason        string
	ServerSeq     sql.NullInt64
	CreatedAt     time.Time
}

// ChangeLogEntry represents a server-side change for pull sync.
type ChangeLogEntry struct {
	Seq        int64
	UserID     string
	EntityType string
	EntityID   string
	Op         string
	Payload    string
	CreatedAt  time.Time
}

// Repository handles sync persistence.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new sync Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// InsertEventAndChangeLog atomically inserts a sync event and its change log entry.
// Returns (isDuplicate, error). On duplicate the event is silently skipped.
func (r *Repository) InsertEventAndChangeLog(ctx context.Context, userID, clientEventID, eventType, occurredAt, payload string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// Insert event with dedup
	result, err := tx.ExecContext(ctx,
		`INSERT INTO sync_events (user_id, client_event_id, event_type, occurred_at, payload, status)
		 VALUES (?, ?, ?, ?, ?, 'accepted')
		 ON CONFLICT (user_id, client_event_id) DO NOTHING`,
		userID, clientEventID, eventType, occurredAt, payload,
	)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	if rows == 0 {
		// Duplicate — already exists, no change log needed
		return true, nil
	}

	// Insert change log entry in the same transaction
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sync_change_log (user_id, entity_type, entity_id, op, payload)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, "sync_event", clientEventID, "upsert", payload,
	)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return false, nil
}

// InsertRejectedEvent records a rejected event for audit.
func (r *Repository) InsertRejectedEvent(ctx context.Context, userID, clientEventID, eventType, occurredAt, payload, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sync_events (user_id, client_event_id, event_type, occurred_at, payload, status, reason)
		 VALUES (?, ?, ?, ?, ?, 'rejected', ?)
		 ON CONFLICT (user_id, client_event_id) DO NOTHING`,
		userID, clientEventID, eventType, occurredAt, payload, reason,
	)
	return err
}

// MaxChangeLogSeq returns the max seq in sync_change_log for a user, or 0 if none.
func (r *Repository) MaxChangeLogSeq(ctx context.Context, userID string) (int64, error) {
	var seq sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(seq) FROM sync_change_log WHERE user_id = ?`, userID,
	).Scan(&seq)
	if err != nil {
		return 0, err
	}
	if !seq.Valid {
		return 0, nil
	}
	return seq.Int64, nil
}

// QueryChanges returns changes for a user after the given cursor (seq).
// cursor=0 means from the beginning. Returns up to limit entries.
func (r *Repository) QueryChanges(ctx context.Context, userID string, cursor int64, limit int) ([]ChangeLogEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT seq, user_id, entity_type, entity_id, op, payload, created_at
		 FROM sync_change_log
		 WHERE user_id = ? AND seq > ?
		 ORDER BY seq ASC
		 LIMIT ?`,
		userID, cursor, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ChangeLogEntry
	for rows.Next() {
		var e ChangeLogEntry
		var createdAt string
		if err := rows.Scan(&e.Seq, &e.UserID, &e.EntityType, &e.EntityID, &e.Op, &e.Payload, &createdAt); err != nil {
			return nil, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// isValidEventType checks if the event type is in the allowed set.
func isValidEventType(eventType string) bool {
	switch eventType {
	case "review_submitted", "output_submitted", "task_completed", "profile_updated":
		return true
	}
	return false
}

// isUniqueViolation checks if the error is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
