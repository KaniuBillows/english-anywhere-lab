package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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
// Returns per-event acks and a server cursor (from sync_change_log.seq domain).
// Returns an error for transient infrastructure failures (DB errors), which
// should be mapped to HTTP 500 so the client can retry the batch.
func (s *Service) PushEvents(ctx context.Context, userID string, events []EventInput) ([]EventAck, string, error) {
	if len(events) > 500 {
		return nil, "", fmt.Errorf("batch size exceeds 500")
	}

	acks := make([]EventAck, 0, len(events))

	for _, evt := range events {
		ack, err := s.processEvent(ctx, userID, evt)
		if err != nil {
			// Infrastructure/transient error — abort batch, let client retry
			return nil, "", fmt.Errorf("process event %s: %w", evt.ClientEventID, err)
		}
		acks = append(acks, ack)
	}

	// server_cursor: max seq from sync_change_log for this user,
	// so it's in the same domain as pull cursors.
	maxSeq, err := s.repo.MaxChangeLogSeq(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("get max change log seq: %w", err)
	}
	cursor := strconv.FormatInt(maxSeq, 10)

	return acks, cursor, nil
}

// processEvent handles a single event: validate → atomic dedup+store.
// Returns (ack, nil) for successful processing (including validation rejections and duplicates).
// Returns (_, error) for transient infrastructure failures that should cause the batch to abort.
func (s *Service) processEvent(ctx context.Context, userID string, evt EventInput) (EventAck, error) {
	// Validate client_event_id
	if evt.ClientEventID == "" {
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}, nil
	}

	// Validate event type
	if !isValidEventType(evt.EventType) {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload), "UNKNOWN_EVENT_TYPE")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "UNKNOWN_EVENT_TYPE",
		}, nil
	}

	// Validate occurred_at
	if evt.OccurredAt == "" {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, "", string(evt.Payload), "INVALID_PAYLOAD")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}, nil
	}

	// Validate payload is valid JSON
	if len(evt.Payload) == 0 || !json.Valid(evt.Payload) {
		_ = s.repo.InsertRejectedEvent(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload), "INVALID_PAYLOAD")
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "rejected",
			Reason:        "INVALID_PAYLOAD",
		}, nil
	}

	// Atomic insert: event + change log in one transaction.
	// Only return "accepted" if both succeed.
	// Infrastructure failures bubble up as errors so the handler returns 500.
	isDuplicate, err := s.repo.InsertEventAndChangeLog(ctx, userID, evt.ClientEventID, evt.EventType, evt.OccurredAt, string(evt.Payload))
	if err != nil {
		s.logger.Error("insert sync event failed", "user_id", userID, "client_event_id", evt.ClientEventID, "error", err)
		return EventAck{}, fmt.Errorf("persist event: %w", err)
	}

	if isDuplicate {
		return EventAck{
			ClientEventID: evt.ClientEventID,
			Status:        "duplicate",
		}, nil
	}

	return EventAck{
		ClientEventID: evt.ClientEventID,
		Status:        "accepted",
	}, nil
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
