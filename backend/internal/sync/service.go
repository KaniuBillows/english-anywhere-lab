package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// EventInput represents a single event from the client batch.
type EventInput struct {
	ClientEventID string
	EventType     string
	OccurredAt    string
	Payload       json.RawMessage
}

// EventAck represents the per-event ack sent back to the client.
type EventAck struct {
	ClientEventID string
	Status        string // accepted, duplicate, rejected
	Reason        string // non-empty only when rejected
}

// ChangeEntry represents a single change in the pull response.
type ChangeEntry struct {
	EntityType string
	EntityID   string
	Op         string
	Payload    json.RawMessage
}

// PullResult holds the result of a pull sync operation.
type PullResult struct {
	NextCursor string
	Changes    []ChangeEntry
}

// Service handles sync business logic.
type Service struct {
	repo   *Repository
	logger *slog.Logger
}

// NewService creates a new sync Service.
func NewService(repo *Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// PushEvents processes a batch of client events.
// Returns per-event acks and a server cursor.
func (s *Service) PushEvents(ctx context.Context, userID string, events []EventInput) ([]EventAck, string, error) {
	if len(events) > 500 {
		return nil, "", fmt.Errorf("batch size exceeds 500")
	}

	acks := make([]EventAck, 0, len(events))

	for _, evt := range events {
		ack := s.processEvent(ctx, userID, evt)
		acks = append(acks, ack)
	}

	// server_cursor: current timestamp as cursor for the push response
	cursor := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return acks, cursor, nil
}

// processEvent handles a single event: validate → dedup → store.
func (s *Service) processEvent(ctx context.Context, userID string, evt EventInput) EventAck {
	// Validate client_event_id
	if evt.ClientEventID == "" {
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}
	}

	// Validate event type
	if !isValidEventType(evt.EventType) {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload), "UNKNOWN_EVENT_TYPE")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "UNKNOWN_EVENT_TYPE",
		}
	}

	// Validate occurred_at
	if evt.OccurredAt == "" {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, "", string(evt.Payload), "INVALID_PAYLOAD")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}
	}

	// Validate payload is valid JSON
	if len(evt.Payload) == 0 || !json.Valid(evt.Payload) {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload), "INVALID_PAYLOAD")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}
	}

	// Insert with dedup
	_, isDuplicate, err := s.repo.InsertEvent(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload))
	if err != nil {
		s.logger.Error("insert sync event failed", "user_id", userID, "client_event_id", evt.ClientEventID, "error", err)
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}
	}

	if isDuplicate {
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "duplicate",
		}
	}

	// Write change log entry for other devices to pull
	_ = s.repo.InsertChangeLogEntry(ctx, userID, "sync_event", evt.ClientEventID, "upsert", string(evt.Payload))

	return EventAck{
		ClientEventID: evt.ClientEventID,
		Status:        "accepted",
	}
}

// PullChanges returns server-side changes after the given cursor.
func (s *Service) PullChanges(ctx context.Context, userID, cursor string, limit int) (*PullResult, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var cursorSeq int64
	if cursor != "" {
		var err error
		cursorSeq, err = strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	entries, err := s.repo.QueryChanges(ctx, userID, cursorSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("query changes: %w", err)
	}

	changes := make([]ChangeEntry, 0, len(entries))
	var lastSeq int64
	for _, e := range entries {
		changes = append(changes, ChangeEntry{
			EntityType: e.EntityType,
			EntityID:   e.EntityID,
			Op:         e.Op,
			Payload:    json.RawMessage(e.Payload),
		})
		lastSeq = e.Seq
	}

	nextCursor := cursor
	if lastSeq > 0 {
		nextCursor = strconv.FormatInt(lastSeq, 10)
	}

	return &PullResult{
		NextCursor: nextCursor,
		Changes:    changes,
	}, nil
}
