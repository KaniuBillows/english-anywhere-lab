package sync_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/db"
	appSync "github.com/bennyshi/english-anywhere-lab/internal/sync"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if _, err := database.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		t.Fatalf("set busy timeout: %v", err)
	}
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func seedUser(t *testing.T, database *sql.DB) string {
	t.Helper()
	userID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := database.Exec(
		`INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		userID, fmt.Sprintf("%s@test.com", userID[:8]), "hash", now, now,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPushEvents_Accept(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc","rating":"good"}`),
		},
	}

	acks, cursor, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	if cursor == "" {
		t.Fatal("expected non-empty cursor")
	}
	if len(acks) != 1 {
		t.Fatalf("expected 1 ack, got %d", len(acks))
	}
	if acks[0].Status != "accepted" {
		t.Fatalf("expected status=accepted, got %s", acks[0].Status)
	}
	if acks[0].ClientEventID != events[0].ClientEventID {
		t.Fatalf("expected client_event_id=%s, got %s", events[0].ClientEventID, acks[0].ClientEventID)
	}
}

func TestPushEvents_Duplicate(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	clientEventID := uuid.New().String()
	events := []appSync.EventInput{
		{
			ClientEventID: clientEventID,
			EventType:     "task_completed",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"task_id":"xyz"}`),
		},
	}

	// First push: accepted
	acks1, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events 1: %v", err)
	}
	if acks1[0].Status != "accepted" {
		t.Fatalf("expected accepted, got %s", acks1[0].Status)
	}

	// Second push with same client_event_id: duplicate
	acks2, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events 2: %v", err)
	}
	if acks2[0].Status != "duplicate" {
		t.Fatalf("expected duplicate, got %s", acks2[0].Status)
	}
}

func TestPushEvents_RejectInvalidEventType(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "unknown_type",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"foo":"bar"}`),
		},
	}

	acks, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	if acks[0].Status != "rejected" {
		t.Fatalf("expected rejected, got %s", acks[0].Status)
	}
	if acks[0].Reason != "UNKNOWN_EVENT_TYPE" {
		t.Fatalf("expected reason=UNKNOWN_EVENT_TYPE, got %s", acks[0].Reason)
	}
}

func TestPushEvents_RejectEmptyPayload(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(``),
		},
	}

	acks, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	if acks[0].Status != "rejected" {
		t.Fatalf("expected rejected, got %s", acks[0].Status)
	}
	if acks[0].Reason != "INVALID_PAYLOAD" {
		t.Fatalf("expected reason=INVALID_PAYLOAD, got %s", acks[0].Reason)
	}
}

func TestPushEvents_RejectMissingClientEventID(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := []appSync.EventInput{
		{
			ClientEventID: "",
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc"}`),
		},
	}

	acks, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	if acks[0].Status != "rejected" {
		t.Fatalf("expected rejected, got %s", acks[0].Status)
	}
	if acks[0].Reason != "INVALID_PAYLOAD" {
		t.Fatalf("expected reason=INVALID_PAYLOAD, got %s", acks[0].Reason)
	}
}

func TestPushEvents_BatchMixed(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc"}`),
		},
		{
			ClientEventID: uuid.New().String(),
			EventType:     "bad_type",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{}`),
		},
		{
			ClientEventID: uuid.New().String(),
			EventType:     "output_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"task_id":"xyz","score":85}`),
		},
	}

	acks, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	if len(acks) != 3 {
		t.Fatalf("expected 3 acks, got %d", len(acks))
	}
	if acks[0].Status != "accepted" {
		t.Fatalf("event 0: expected accepted, got %s", acks[0].Status)
	}
	if acks[1].Status != "rejected" {
		t.Fatalf("event 1: expected rejected, got %s", acks[1].Status)
	}
	if acks[2].Status != "accepted" {
		t.Fatalf("event 2: expected accepted, got %s", acks[2].Status)
	}
}

func TestPushEvents_BatchSizeLimit(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	events := make([]appSync.EventInput, 501)
	for i := range events {
		events[i] = appSync.EventInput{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc"}`),
		}
	}

	_, _, err := svc.PushEvents(context.Background(), userID, events)
	if err == nil {
		t.Fatal("expected error for batch size > 500")
	}
}

func TestPullChanges_Empty(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	result, err := svc.PullChanges(context.Background(), userID, "", 200)
	if err != nil {
		t.Fatalf("pull changes: %v", err)
	}
	if len(result.Changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(result.Changes))
	}
}

func TestPullChanges_AfterPush(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	// Push events
	events := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc","rating":"good"}`),
		},
		{
			ClientEventID: uuid.New().String(),
			EventType:     "task_completed",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"task_id":"xyz"}`),
		},
	}

	_, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}

	// Pull changes from beginning
	result, err := svc.PullChanges(context.Background(), userID, "", 200)
	if err != nil {
		t.Fatalf("pull changes: %v", err)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(result.Changes))
	}
	if result.NextCursor == "" {
		t.Fatal("expected non-empty next_cursor")
	}

	// Pull changes after cursor: should be empty
	result2, err := svc.PullChanges(context.Background(), userID, result.NextCursor, 200)
	if err != nil {
		t.Fatalf("pull changes 2: %v", err)
	}
	if len(result2.Changes) != 0 {
		t.Fatalf("expected 0 changes after cursor, got %d", len(result2.Changes))
	}
}

func TestPullChanges_Pagination(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	// Push 5 events
	for i := 0; i < 5; i++ {
		events := []appSync.EventInput{
			{
				ClientEventID: uuid.New().String(),
				EventType:     "review_submitted",
				OccurredAt:    time.Now().UTC().Format(time.RFC3339),
				Payload:       json.RawMessage(fmt.Sprintf(`{"card_id":"card_%d"}`, i)),
			},
		}
		_, _, err := svc.PushEvents(context.Background(), userID, events)
		if err != nil {
			t.Fatalf("push event %d: %v", i, err)
		}
	}

	// Pull with limit 2
	result1, err := svc.PullChanges(context.Background(), userID, "", 2)
	if err != nil {
		t.Fatalf("pull page 1: %v", err)
	}
	if len(result1.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(result1.Changes))
	}

	// Pull next page
	result2, err := svc.PullChanges(context.Background(), userID, result1.NextCursor, 2)
	if err != nil {
		t.Fatalf("pull page 2: %v", err)
	}
	if len(result2.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(result2.Changes))
	}

	// Pull last page
	result3, err := svc.PullChanges(context.Background(), userID, result2.NextCursor, 2)
	if err != nil {
		t.Fatalf("pull page 3: %v", err)
	}
	if len(result3.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result3.Changes))
	}
}

func TestPullChanges_UserIsolation(t *testing.T) {
	database := setupTestDB(t)
	user1 := seedUser(t, database)
	user2 := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	// Push event for user1
	events1 := []appSync.EventInput{
		{
			ClientEventID: uuid.New().String(),
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc"}`),
		},
	}
	_, _, err := svc.PushEvents(context.Background(), user1, events1)
	if err != nil {
		t.Fatalf("push for user1: %v", err)
	}

	// User2 should see nothing
	result, err := svc.PullChanges(context.Background(), user2, "", 200)
	if err != nil {
		t.Fatalf("pull for user2: %v", err)
	}
	if len(result.Changes) != 0 {
		t.Fatalf("user2 should see 0 changes, got %d", len(result.Changes))
	}

	// User1 should see their own
	result1, err := svc.PullChanges(context.Background(), user1, "", 200)
	if err != nil {
		t.Fatalf("pull for user1: %v", err)
	}
	if len(result1.Changes) != 1 {
		t.Fatalf("user1 should see 1 change, got %d", len(result1.Changes))
	}
}

func TestPushEvents_DuplicateDoesNotCreateChangeLog(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	clientEventID := uuid.New().String()
	events := []appSync.EventInput{
		{
			ClientEventID: clientEventID,
			EventType:     "review_submitted",
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"card_id":"abc"}`),
		},
	}

	// First push
	_, _, _ = svc.PushEvents(context.Background(), userID, events)

	// Second push (duplicate)
	_, _, _ = svc.PushEvents(context.Background(), userID, events)

	// Should only have 1 change log entry, not 2
	result, err := svc.PullChanges(context.Background(), userID, "", 200)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change (no double counting), got %d", len(result.Changes))
	}
}

func TestPushEvents_AllEventTypes(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	repo := appSync.NewRepository(database)
	svc := appSync.NewService(repo, testLogger())

	eventTypes := []string{"review_submitted", "output_submitted", "task_completed", "profile_updated"}
	events := make([]appSync.EventInput, len(eventTypes))
	for i, et := range eventTypes {
		events[i] = appSync.EventInput{
			ClientEventID: uuid.New().String(),
			EventType:     et,
			OccurredAt:    time.Now().UTC().Format(time.RFC3339),
			Payload:       json.RawMessage(`{"key":"value"}`),
		}
	}

	acks, _, err := svc.PushEvents(context.Background(), userID, events)
	if err != nil {
		t.Fatalf("push events: %v", err)
	}
	for i, ack := range acks {
		if ack.Status != "accepted" {
			t.Fatalf("event %d (%s): expected accepted, got %s", i, eventTypes[i], ack.Status)
		}
	}
}
