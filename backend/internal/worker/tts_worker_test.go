package worker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/bennyshi/english-anywhere-lab/internal/storage"
	"github.com/bennyshi/english-anywhere-lab/internal/tts"
	"github.com/bennyshi/english-anywhere-lab/internal/worker"
	"github.com/google/uuid"
)

func setupTTSService(t *testing.T) *tts.Service {
	t.Helper()
	tmpDir := t.TempDir()
	store, err := storage.NewLocalStore(tmpDir, "/static/files")
	if err != nil {
		t.Fatalf("new local store: %v", err)
	}
	return tts.NewService(tts.NewStubProvider(), store, tts.TTSConfig{
		Voice:        "en_default_female",
		Speed:        1.0,
		Format:       "wav",
		SampleRate:   22050,
		MaxTextChars: 280,
	})
}

func seedTTSJob(t *testing.T, database *sql.DB, userID, cardID, text string) string {
	t.Helper()
	jobID := uuid.New().String()
	payload, _ := json.Marshal(map[string]string{
		"card_id": cardID,
		"text":    text,
		"field":   "front_text",
	})
	_, err := database.Exec(
		`INSERT INTO ai_generation_jobs (id, user_id, job_type, domain, level, template_version, request_payload, status, created_at)
		 VALUES (?, ?, 'tts_generation', 'general', 'A1', 'v1', ?, 'queued', datetime('now'))`,
		jobID, userID, string(payload),
	)
	if err != nil {
		t.Fatalf("seed TTS job: %v", err)
	}
	return jobID
}

func seedCard(t *testing.T, database *sql.DB, lessonID, frontText string) string {
	t.Helper()
	cardID := uuid.New().String()
	_, err := database.Exec(
		`INSERT INTO cards (id, lesson_id, front_text, back_text, created_at) VALUES (?, ?, ?, '翻译', datetime('now'))`,
		cardID, lessonID, frontText,
	)
	if err != nil {
		t.Fatalf("seed card: %v", err)
	}
	return cardID
}

func seedLesson(t *testing.T, database *sql.DB, packID string) string {
	t.Helper()
	lessonID := uuid.New().String()
	_, err := database.Exec(
		`INSERT INTO lessons (id, pack_id, title, lesson_type, position, estimated_minutes, created_at) VALUES (?, ?, 'Test Lesson', 'reading', 1, 10, datetime('now'))`,
		lessonID, packID,
	)
	if err != nil {
		t.Fatalf("seed lesson: %v", err)
	}
	return lessonID
}

func seedPack(t *testing.T, database *sql.DB, userID string) string {
	t.Helper()
	packID := uuid.New().String()
	_, err := database.Exec(
		`INSERT INTO resource_packs (id, source, title, domain, level, estimated_minutes, created_by_user_id, created_at) VALUES (?, 'ai', 'Test Pack', 'tech', 'B1', 20, ?, datetime('now'))`,
		packID, userID,
	)
	if err != nil {
		t.Fatalf("seed pack: %v", err)
	}
	return packID
}

func TestTTSWorker_ProcessJob_Success(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	packID := seedPack(t, database, userID)
	lessonID := seedLesson(t, database, packID)
	cardID := seedCard(t, database, lessonID, "hello world")
	jobID := seedTTSJob(t, database, userID, cardID, "hello world")

	ttsSvc := setupTTSService(t)
	ttsWorker := worker.NewTTSWorker(database, ttsSvc, testLogger(), 2)

	job, err := ttsWorker.ClaimNextTTSJob(context.Background())
	if err != nil {
		t.Fatalf("claim TTS job: %v", err)
	}
	if job.ID != jobID {
		t.Fatalf("expected job_id=%s, got %s", jobID, job.ID)
	}

	err = ttsWorker.ProcessJob(context.Background(), job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	// Verify card.audio_url is set
	var audioURL sql.NullString
	err = database.QueryRow("SELECT audio_url FROM cards WHERE id = ?", cardID).Scan(&audioURL)
	if err != nil {
		t.Fatalf("query audio_url: %v", err)
	}
	if !audioURL.Valid || audioURL.String == "" {
		t.Fatal("expected audio_url to be set")
	}
}

func TestTTSWorker_ProcessJob_Dedup(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	packID := seedPack(t, database, userID)
	lessonID := seedLesson(t, database, packID)
	cardID := seedCard(t, database, lessonID, "hello world")
	seedTTSJob(t, database, userID, cardID, "hello world")

	ttsSvc := setupTTSService(t)
	ttsWorker := worker.NewTTSWorker(database, ttsSvc, testLogger(), 2)

	job, err := ttsWorker.ClaimNextTTSJob(context.Background())
	if err != nil {
		t.Fatalf("claim TTS job: %v", err)
	}

	// Process twice (simulating dedup)
	err = ttsWorker.ProcessJob(context.Background(), job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	// Second job for same text
	jobID2 := seedTTSJob(t, database, userID, cardID, "hello world")
	job2, err := ttsWorker.ClaimNextTTSJob(context.Background())
	if err != nil {
		t.Fatalf("claim second TTS job: %v", err)
	}
	if job2.ID != jobID2 {
		t.Fatalf("expected job2=%s, got %s", jobID2, job2.ID)
	}

	err = ttsWorker.ProcessJob(context.Background(), job2)
	if err != nil {
		t.Fatalf("process job2 (dedup): %v", err)
	}
}

func TestTTSWorker_ProcessJob_NonexistentCard(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	fakeCardID := uuid.New().String()
	seedTTSJob(t, database, userID, fakeCardID, "hello")

	ttsSvc := setupTTSService(t)
	ttsWorker := worker.NewTTSWorker(database, ttsSvc, testLogger(), 2)

	job, err := ttsWorker.ClaimNextTTSJob(context.Background())
	if err != nil {
		t.Fatalf("claim TTS job: %v", err)
	}

	err = ttsWorker.ProcessJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for nonexistent card")
	}
}

func TestTTSWorker_ProcessJob_InvalidPayload(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)

	// Seed a job with invalid payload
	jobID := uuid.New().String()
	_, err := database.Exec(
		`INSERT INTO ai_generation_jobs (id, user_id, job_type, domain, level, template_version, request_payload, status, created_at)
		 VALUES (?, ?, 'tts_generation', 'general', 'A1', 'v1', '{"card_id":"","text":""}', 'queued', datetime('now'))`,
		jobID, userID,
	)
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}

	ttsSvc := setupTTSService(t)
	ttsWorker := worker.NewTTSWorker(database, ttsSvc, testLogger(), 2)

	job, err := ttsWorker.ClaimNextTTSJob(context.Background())
	if err != nil {
		t.Fatalf("claim: %v", err)
	}

	err = ttsWorker.ProcessJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for invalid payload")
	}
}
