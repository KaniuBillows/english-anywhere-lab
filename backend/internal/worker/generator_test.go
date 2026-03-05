package worker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/db"
	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/worker"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var validLLMResponse = `{
  "title": "Tech English Pack",
  "description": "A pack for learning tech vocabulary",
  "estimated_minutes": 40,
  "lessons": [
    {
      "title": "Lesson 1: Cloud Computing Basics",
      "lesson_type": "reading",
      "position": 1,
      "estimated_minutes": 20,
      "cards": [
        {
          "front_text": "cloud computing",
          "back_text": "云计算",
          "example_text": "Many companies use cloud computing to store data."
        },
        {
          "front_text": "server",
          "back_text": "服务器",
          "example_text": "The server handles thousands of requests per second."
        },
        {
          "front_text": "deploy",
          "back_text": "部署",
          "example_text": "We need to deploy the new version today."
        }
      ],
      "output_tasks": [
        {
          "task_type": "writing",
          "prompt_text": "Write a short paragraph about cloud computing benefits.",
          "reference_answer": "Cloud computing offers scalability, cost savings, and flexibility."
        }
      ]
    },
    {
      "title": "Lesson 2: API Design",
      "lesson_type": "mixed",
      "position": 2,
      "estimated_minutes": 20,
      "cards": [
        {
          "front_text": "API",
          "back_text": "应用程序接口",
          "example_text": "The API returns JSON data."
        },
        {
          "front_text": "endpoint",
          "back_text": "端点",
          "example_text": "Each endpoint handles a specific resource."
        },
        {
          "front_text": "authentication",
          "back_text": "身份验证",
          "example_text": "Authentication is required to access the API."
        }
      ],
      "output_tasks": [
        {
          "task_type": "speaking",
          "prompt_text": "Explain what a REST API is in your own words."
        }
      ]
    }
  ]
}`

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

func seedJob(t *testing.T, database *sql.DB, userID string) string {
	t.Helper()
	jobID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	payload, _ := json.Marshal(map[string]any{
		"level":         "B1",
		"domain":        "tech",
		"daily_minutes": 20,
		"days":          2,
		"focus_skills":  []string{"reading"},
	})
	_, err := database.Exec(
		`INSERT INTO ai_generation_jobs (id, user_id, job_type, domain, level, template_version, request_payload, status, created_at)
		 VALUES (?, ?, 'pack_generation', 'tech', 'B1', 'v1', ?, 'queued', ?)`,
		jobID, userID, string(payload), now,
	)
	if err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return jobID
}

func TestProcessJob_Success(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	jobID := seedJob(t, database, userID)

	// Mock LLM server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": validLLMResponse}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LLMBaseURL:    mockServer.URL,
		LLMAPIKey:     "test-key",
		LLMModel:      "test-model",
		LLMTimeoutSec: 30,
		LLMMaxRetries: 0,
	}
	llmClient := llm.NewClient(cfg)
	repo := pack.NewRepository(database)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	gen := worker.NewGenerator(repo, llmClient, database, logger)

	// Claim and process
	job, err := repo.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if job.ID != jobID {
		t.Fatalf("expected job_id=%s, got %s", jobID, job.ID)
	}

	// Use exported ProcessJob for testing
	err = gen.ProcessJob(context.Background(), job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	// Verify job status
	var status string
	err = database.QueryRow("SELECT status FROM ai_generation_jobs WHERE id = ?", jobID).Scan(&status)
	if err != nil {
		t.Fatalf("query job status: %v", err)
	}
	if status != "success" {
		t.Fatalf("expected status=success, got %s", status)
	}

	// Verify pack created
	var packCount int
	err = database.QueryRow("SELECT COUNT(*) FROM resource_packs WHERE source = 'ai' AND created_by_user_id = ?", userID).Scan(&packCount)
	if err != nil {
		t.Fatalf("count packs: %v", err)
	}
	if packCount != 1 {
		t.Fatalf("expected 1 pack, got %d", packCount)
	}

	// Verify lessons
	var lessonCount int
	err = database.QueryRow(
		`SELECT COUNT(*) FROM lessons l
		 JOIN resource_packs rp ON rp.id = l.pack_id
		 WHERE rp.created_by_user_id = ?`, userID,
	).Scan(&lessonCount)
	if err != nil {
		t.Fatalf("count lessons: %v", err)
	}
	if lessonCount != 2 {
		t.Fatalf("expected 2 lessons, got %d", lessonCount)
	}

	// Verify cards
	var cardCount int
	err = database.QueryRow(
		`SELECT COUNT(*) FROM cards c
		 JOIN lessons l ON l.id = c.lesson_id
		 JOIN resource_packs rp ON rp.id = l.pack_id
		 WHERE rp.created_by_user_id = ?`, userID,
	).Scan(&cardCount)
	if err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if cardCount != 6 {
		t.Fatalf("expected 6 cards, got %d", cardCount)
	}

	// Verify output tasks
	var taskCount int
	err = database.QueryRow(
		`SELECT COUNT(*) FROM output_tasks ot
		 JOIN lessons l ON l.id = ot.lesson_id
		 JOIN resource_packs rp ON rp.id = l.pack_id
		 WHERE rp.created_by_user_id = ?`, userID,
	).Scan(&taskCount)
	if err != nil {
		t.Fatalf("count output tasks: %v", err)
	}
	if taskCount != 2 {
		t.Fatalf("expected 2 output tasks, got %d", taskCount)
	}
}

func TestProcessJob_InvalidLLMResponse(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	jobID := seedJob(t, database, userID)

	// Mock LLM server returning invalid JSON structure
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": `{"title": "", "lessons": []}`}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LLMBaseURL:    mockServer.URL,
		LLMAPIKey:     "test-key",
		LLMModel:      "test-model",
		LLMTimeoutSec: 30,
		LLMMaxRetries: 0,
	}
	llmClient := llm.NewClient(cfg)
	repo := pack.NewRepository(database)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	gen := worker.NewGenerator(repo, llmClient, database, logger)

	job, err := repo.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}

	err = gen.ProcessJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for invalid LLM response")
	}

	// Simulate the Run loop behavior: mark as failed
	_ = repo.UpdateJobStatus(context.Background(), jobID, "failed", "", err.Error())

	// Verify job status
	var status, errMsg string
	var errMsgNull sql.NullString
	err = database.QueryRow("SELECT status, error_message FROM ai_generation_jobs WHERE id = ?", jobID).Scan(&status, &errMsgNull)
	if err != nil {
		t.Fatalf("query job: %v", err)
	}
	if status != "failed" {
		t.Fatalf("expected status=failed, got %s", status)
	}
	if errMsgNull.Valid {
		errMsg = errMsgNull.String
	}
	if errMsg == "" {
		t.Fatal("expected non-empty error_message")
	}

	// Verify no pack created
	var packCount int
	err = database.QueryRow("SELECT COUNT(*) FROM resource_packs WHERE source = 'ai' AND created_by_user_id = ?", userID).Scan(&packCount)
	if err != nil {
		t.Fatalf("count packs: %v", err)
	}
	if packCount != 0 {
		t.Fatalf("expected 0 packs, got %d", packCount)
	}
}

func TestProcessJob_LLMServerError(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	seedJob(t, database, userID)

	// Mock LLM server that returns 500
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LLMBaseURL:    mockServer.URL,
		LLMAPIKey:     "test-key",
		LLMModel:      "test-model",
		LLMTimeoutSec: 5,
		LLMMaxRetries: 1,
	}
	llmClient := llm.NewClient(cfg)
	repo := pack.NewRepository(database)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	gen := worker.NewGenerator(repo, llmClient, database, logger)

	job, err := repo.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}

	err = gen.ProcessJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for LLM server error")
	}
}

func TestProcessJob_EnqueuesTTSJobs(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	seedJob(t, database, userID)

	// Mock LLM server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": validLLMResponse}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LLMBaseURL:    mockServer.URL,
		LLMAPIKey:     "test-key",
		LLMModel:      "test-model",
		LLMTimeoutSec: 30,
		LLMMaxRetries: 0,
	}
	llmClient := llm.NewClient(cfg)
	repo := pack.NewRepository(database)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	gen := worker.NewGenerator(repo, llmClient, database, logger)

	job, err := repo.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}

	err = gen.ProcessJob(context.Background(), job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	// Verify TTS jobs were created for all 6 cards
	var ttsJobCount int
	err = database.QueryRow("SELECT COUNT(*) FROM ai_generation_jobs WHERE job_type = 'tts_generation'").Scan(&ttsJobCount)
	if err != nil {
		t.Fatalf("count TTS jobs: %v", err)
	}
	if ttsJobCount != 6 {
		t.Fatalf("expected 6 TTS jobs, got %d", ttsJobCount)
	}

	// Verify TTS job payloads have card_id and text
	rows, err := database.Query("SELECT request_payload FROM ai_generation_jobs WHERE job_type = 'tts_generation'")
	if err != nil {
		t.Fatalf("query TTS jobs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			t.Fatalf("scan payload: %v", err)
		}
		var p map[string]string
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if p["card_id"] == "" {
			t.Fatal("TTS job payload missing card_id")
		}
		if p["text"] == "" {
			t.Fatal("TTS job payload missing text")
		}
		if p["field"] != "front_text" {
			t.Fatalf("expected field=front_text, got %s", p["field"])
		}
	}
}

func TestTTSJobs_DoNotPollutePackGenerationDailyLimit(t *testing.T) {
	database := setupTestDB(t)
	userID := seedUser(t, database)
	seedJob(t, database, userID)

	// Mock LLM server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": validLLMResponse}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		LLMBaseURL:    mockServer.URL,
		LLMAPIKey:     "test-key",
		LLMModel:      "test-model",
		LLMTimeoutSec: 30,
		LLMMaxRetries: 0,
	}
	llmClient := llm.NewClient(cfg)
	repo := pack.NewRepository(database)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	gen := worker.NewGenerator(repo, llmClient, database, logger)

	job, err := repo.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}

	err = gen.ProcessJob(context.Background(), job)
	if err != nil {
		t.Fatalf("process job: %v", err)
	}

	// At this point: 1 pack_generation job (success) + 6 tts_generation jobs (queued).
	// Daily limit is 2 for pack_generation.
	// The user should still be able to create a second pack_generation job.
	var totalJobs int
	err = database.QueryRow("SELECT COUNT(*) FROM ai_generation_jobs WHERE user_id = ?", userID).Scan(&totalJobs)
	if err != nil {
		t.Fatalf("count all jobs: %v", err)
	}
	if totalJobs != 7 { // 1 pack + 6 tts
		t.Fatalf("expected 7 total jobs, got %d", totalJobs)
	}

	// CountUserJobsToday should only count pack_generation jobs
	count, err := repo.CountUserJobsToday(context.Background(), userID)
	if err != nil {
		t.Fatalf("count user jobs today: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 pack_generation job counted, got %d", count)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
