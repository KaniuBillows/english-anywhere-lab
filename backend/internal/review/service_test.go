package review

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/db"
	"github.com/bennyshi/english-anywhere-lab/internal/scheduler"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.Exec("PRAGMA foreign_keys = ON")

	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func seedTestData(t *testing.T, database *sql.DB) (userID, cardID, stateID string) {
	t.Helper()
	userID = uuid.New().String()
	cardID = uuid.New().String()
	stateID = uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Create user
	_, err := database.Exec(
		`INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		userID, "review-test@example.com", "hash", now, now,
	)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create card
	_, err = database.Exec(
		`INSERT INTO cards (id, front_text, back_text, created_at) VALUES (?, ?, ?, ?)`,
		cardID, "hello", "你好", now,
	)
	if err != nil {
		t.Fatalf("create card: %v", err)
	}

	// Create user_card_state
	_, err = database.Exec(
		`INSERT INTO user_card_states (id, user_id, card_id, status, due_at, reps, lapses, scheduled_days, created_at, updated_at)
		 VALUES (?, ?, ?, 'new', ?, 0, 0, 0, ?, ?)`,
		stateID, userID, cardID, now, now, now,
	)
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	return userID, cardID, stateID
}

func TestSubmitReview_NewCardGood(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	userID, cardID, stateID := seedTestData(t, database)

	repo := NewRepository(database)
	fsrs := scheduler.NewFSRS()
	svc := NewService(repo, fsrs)

	ctx := context.Background()
	now := time.Now().UTC()

	result, err := svc.SubmitReview(ctx, SubmitInput{
		UserID:          userID,
		CardID:          cardID,
		UserCardStateID: stateID,
		Rating:          "good",
		ReviewedAt:      now,
		ClientEventID:   uuid.New().String(),
		IdempotencyKey:  uuid.New().String(),
	})
	if err != nil {
		t.Fatalf("submit review: %v", err)
	}

	if !result.Accepted {
		t.Error("expected accepted=true")
	}
	if result.Status != "review" {
		t.Errorf("expected status=review, got %s", result.Status)
	}
	if result.ScheduledDays != 1 {
		t.Errorf("expected scheduled_days=1, got %d", result.ScheduledDays)
	}
}

func TestSubmitReview_Idempotency(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	userID, cardID, stateID := seedTestData(t, database)

	repo := NewRepository(database)
	fsrs := scheduler.NewFSRS()
	svc := NewService(repo, fsrs)

	ctx := context.Background()
	now := time.Now().UTC()
	idempotencyKey := uuid.New().String()

	// First submission
	result1, err := svc.SubmitReview(ctx, SubmitInput{
		UserID:          userID,
		CardID:          cardID,
		UserCardStateID: stateID,
		Rating:          "good",
		ReviewedAt:      now,
		ClientEventID:   uuid.New().String(),
		IdempotencyKey:  idempotencyKey,
	})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}

	// Second submission with same idempotency key
	result2, err := svc.SubmitReview(ctx, SubmitInput{
		UserID:          userID,
		CardID:          cardID,
		UserCardStateID: stateID,
		Rating:          "good",
		ReviewedAt:      now,
		ClientEventID:   uuid.New().String(),
		IdempotencyKey:  idempotencyKey,
	})
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}

	if !result2.Accepted {
		t.Error("expected accepted=true for duplicate")
	}

	// Verify only one review log exists
	var count int
	database.QueryRow("SELECT COUNT(*) FROM review_logs WHERE user_id = ?", userID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 review log, got %d", count)
	}

	_ = result1
}

func TestGetDueCards(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	userID, _, _ := seedTestData(t, database)

	repo := NewRepository(database)
	fsrs := scheduler.NewFSRS()
	svc := NewService(repo, fsrs)

	ctx := context.Background()

	result, err := svc.GetDueCards(ctx, userID, 30)
	if err != nil {
		t.Fatalf("get due cards: %v", err)
	}

	if result.DueCount != 1 {
		t.Errorf("expected due_count=1, got %d", result.DueCount)
	}
	if len(result.Cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result.Cards))
	}
	if result.Cards[0].FrontText != "hello" {
		t.Errorf("expected front_text=hello, got %s", result.Cards[0].FrontText)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
