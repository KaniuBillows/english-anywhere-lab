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

// InsertEvent inserts a sync event and returns whether it was a duplicate.
// Returns (eventID, isDuplicate, error).
func (r *Repository) InsertEvent(ctx context.Context, userID, clientEventID, eventType, occurredAt, payload string) (int64, bool, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO sync_events (user_id, client_event_id, event_type, occurred_at, payload, status)
		 VALUES (?, ?, ?, ?, ?, 'accepted')
		 ON CONFLICT (user_id, client_event_id) DO NOTHING`,
		userID, clientEventID, eventType, occurredAt, payload,
	)
	if err != nil {
		return 0, false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, false, err
	}

	if rows == 0 {
		// Duplicate — already exists
		return 0, true, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	return id, false, nil
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

// InsertChangeLogEntry records a server-side change for pull sync.
func (r *Repository) InsertChangeLogEntry(ctx context.Context, userID, entityType, entityID, op, payload string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sync_change_log (user_id, entity_type, entity_id, op, payload)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, entityType, entityID, op, payload,
	)
	return err
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
