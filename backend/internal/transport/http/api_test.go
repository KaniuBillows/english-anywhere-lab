package http_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/db"
	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/bennyshi/english-anywhere-lab/internal/output"
	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/plan"
	"github.com/bennyshi/english-anywhere-lab/internal/progress"
	"github.com/bennyshi/english-anywhere-lab/internal/review"
	"github.com/bennyshi/english-anywhere-lab/internal/scheduler"
	appSync "github.com/bennyshi/english-anywhere-lab/internal/sync"
	httptransport "github.com/bennyshi/english-anywhere-lab/internal/transport/http"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// mockLLMCaller returns canned WritingFeedback JSON for testing.
type mockLLMCaller struct{}

func (m *mockLLMCaller) ChatCompletion(_ context.Context, _ []llm.Message) (string, error) {
	return `{"overall_score":85,"errors":[{"original":"I go to school yesterday","correction":"I went to school yesterday","explanation":"Use past tense for past events"}],"revised_text":"I went to school yesterday.","next_actions":["Practice past tense verbs"]}`, nil
}

// testEnv holds a running httptest.Server backed by the full application stack.
type testEnv struct {
	server *httptest.Server
	db     *sql.DB
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Use a named shared-cache in-memory DB so all connections from the pool
	// see the same database. Each test gets a unique name to avoid interference.
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

	cfg := &config.Config{
		JWTSignKey:    "test-secret-key-at-least-32-characters-long",
		JWTIssuer:     "test",
		JWTAccessTTL:  60 * time.Minute,
		JWTRefreshTTL: 720 * time.Hour,
	}

	application := &app.App{
		Config: cfg,
		DB:     database,
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	authRepo := auth.NewRepository(database)
	authJWT := auth.NewJWTManager(cfg)
	authSvc := auth.NewService(authRepo, authJWT)

	fsrs := scheduler.NewFSRS()
	reviewRepo := review.NewRepository(database)
	reviewSvc := review.NewService(reviewRepo, fsrs)

	planRepo := plan.NewRepository(database)
	planSvc := plan.NewService(planRepo)

	progressRepo := progress.NewRepository(database)
	progressSvc := progress.NewService(progressRepo)

	packRepo := pack.NewRepository(database)
	packSvc := pack.NewService(packRepo, database)

	outputRepo := output.NewRepository(database)
	outputSvc := output.NewService(outputRepo, &mockLLMCaller{})

	syncRepo := appSync.NewRepository(database)
	syncSvc := appSync.NewService(syncRepo, application.Logger)

	router := httptransport.NewRouter(application, authSvc, authJWT, reviewSvc, planSvc, progressSvc, packSvc, outputSvc, syncSvc, httptransport.StaticFilesConfig{})
	server := httptest.NewServer(router)

	t.Cleanup(func() {
		server.Close()
		database.Close()
	})

	return &testEnv{server: server, db: database}
}

// doRequest performs an HTTP request against the test server.
func (e *testEnv) doRequest(t *testing.T, method, path, token string, body any, headers map[string]string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, e.server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

// registerUser registers a user and returns the auth response.
func (e *testEnv) registerUser(t *testing.T, email, password string) dto.AuthResponse {
	t.Helper()
	resp := e.doRequest(t, "POST", "/api/v1/auth/register", "", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, body)
	}
	return decodeJSON[dto.AuthResponse](t, resp)
}

// loginUser logs in and returns the auth response.
func (e *testEnv) loginUser(t *testing.T, email, password string) dto.AuthResponse {
	t.Helper()
	resp := e.doRequest(t, "POST", "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, body)
	}
	return decodeJSON[dto.AuthResponse](t, resp)
}

// seedReviewData inserts cards + user_card_states directly into the DB.
func (e *testEnv) seedReviewData(t *testing.T, userID string, count int) []struct{ CardID, StateID string } {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	var result []struct{ CardID, StateID string }

	for i := 0; i < count; i++ {
		cardID := uuid.New().String()
		stateID := uuid.New().String()

		_, err := e.db.Exec(
			`INSERT INTO cards (id, front_text, back_text, created_at) VALUES (?, ?, ?, ?)`,
			cardID, fmt.Sprintf("front-%d", i), fmt.Sprintf("back-%d", i), now,
		)
		if err != nil {
			t.Fatalf("seed card %d: %v", i, err)
		}

		_, err = e.db.Exec(
			`INSERT INTO user_card_states (id, user_id, card_id, status, due_at, reps, lapses, scheduled_days, created_at, updated_at)
			 VALUES (?, ?, ?, 'new', ?, 0, 0, 0, ?, ?)`,
			stateID, userID, cardID, now, now, now,
		)
		if err != nil {
			t.Fatalf("seed state %d: %v", i, err)
		}

		result = append(result, struct{ CardID, StateID string }{cardID, stateID})
	}
	return result
}

// seedProgressData inserts 7 days of progress_daily rows for the user.
func (e *testEnv) seedProgressData(t *testing.T, userID string) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		_, err := e.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, cards_reviewed, review_accuracy, streak_count)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			userID, date, 20+i, 10+i, 0.85, i+1,
		)
		if err != nil {
			t.Fatalf("seed progress day %d: %v", i, err)
		}
	}
}

// decodeJSON decodes the response body into T.
func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return v
}

func TestAPI(t *testing.T) {
	env := newTestEnv(t)

	const testEmail = "integration@test.com"
	const testPassword = "securepassword123"

	var accessToken string
	var refreshToken string
	var userID string

	// ==================== 1. Health ====================
	t.Run("Health", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/health", "", nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON[map[string]string](t, resp)
		if body["status"] != "ok" {
			t.Fatalf("expected status=ok, got %s", body["status"])
		}
	})

	// ==================== 2. Auth Flow ====================
	t.Run("Auth", func(t *testing.T) {
		t.Run("Register", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/register", "", map[string]string{
				"email":    testEmail,
				"password": testPassword,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
			}

			authResp := decodeJSON[dto.AuthResponse](t, resp)
			if authResp.User.ID == "" {
				t.Fatal("expected non-empty user.id")
			}
			if authResp.User.Email != testEmail {
				t.Fatalf("expected email=%s, got %s", testEmail, authResp.User.Email)
			}
			if authResp.Tokens.AccessToken == "" {
				t.Fatal("expected non-empty access_token")
			}
			if authResp.Tokens.RefreshToken == "" {
				t.Fatal("expected non-empty refresh_token")
			}
		})

		t.Run("Register duplicate email", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/register", "", map[string]string{
				"email":    testEmail,
				"password": testPassword,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("expected 409, got %d", resp.StatusCode)
			}
			errResp := decodeJSON[dto.ErrorResponse](t, resp)
			if errResp.Code != "CONFLICT" {
				t.Fatalf("expected code=CONFLICT, got %s", errResp.Code)
			}
		})

		t.Run("Register bad password", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/register", "", map[string]string{
				"email":    "short@test.com",
				"password": "short",
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})

		t.Run("Login", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/login", "", map[string]string{
				"email":    testEmail,
				"password": testPassword,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			authResp := decodeJSON[dto.AuthResponse](t, resp)
			accessToken = authResp.Tokens.AccessToken
			refreshToken = authResp.Tokens.RefreshToken
			userID = authResp.User.ID

			if accessToken == "" {
				t.Fatal("expected non-empty access_token")
			}
			if userID == "" {
				t.Fatal("expected non-empty user.id")
			}
		})

		t.Run("Login wrong password", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/login", "", map[string]string{
				"email":    testEmail,
				"password": "wrongpassword123",
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
		})

		t.Run("Refresh token", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/refresh", "", map[string]string{
				"refresh_token": refreshToken,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			tokens := decodeJSON[dto.AuthTokensDTO](t, resp)
			if tokens.AccessToken == "" {
				t.Fatal("expected non-empty access_token from refresh")
			}
			if tokens.RefreshToken == "" {
				t.Fatal("expected non-empty refresh_token from refresh")
			}
		})

		t.Run("Refresh invalid token", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/auth/refresh", "", map[string]string{
				"refresh_token": "invalid-token-value",
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
		})
	})

	// ==================== 3. Profile ====================
	t.Run("Profile", func(t *testing.T) {
		t.Run("GET /me without token", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/me", "", nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
		})

		t.Run("GET /me", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/me", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			me := decodeJSON[dto.MeResponse](t, resp)
			if me.User.Email != testEmail {
				t.Fatalf("expected email=%s, got %s", testEmail, me.User.Email)
			}
			if me.LearningProfile.CurrentLevel != "A2" {
				t.Fatalf("expected default level=A2, got %s", me.LearningProfile.CurrentLevel)
			}
			if me.LearningProfile.DailyMinutes != 20 {
				t.Fatalf("expected default daily_minutes=20, got %d", me.LearningProfile.DailyMinutes)
			}
		})

		t.Run("PATCH /me/profile", func(t *testing.T) {
			resp := env.doRequest(t, "PATCH", "/api/v1/me/profile", accessToken, map[string]any{
				"current_level":  "B1",
				"daily_minutes":  30,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			me := decodeJSON[dto.MeResponse](t, resp)
			if me.LearningProfile.CurrentLevel != "B1" {
				t.Fatalf("expected level=B1, got %s", me.LearningProfile.CurrentLevel)
			}
			if me.LearningProfile.DailyMinutes != 30 {
				t.Fatalf("expected daily_minutes=30, got %d", me.LearningProfile.DailyMinutes)
			}
		})

		t.Run("PATCH /me/profile invalid level", func(t *testing.T) {
			resp := env.doRequest(t, "PATCH", "/api/v1/me/profile", accessToken, map[string]any{
				"current_level": "Z9",
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	})

	// ==================== 4. Plan Flow ====================
	var planID string
	var firstTaskID string

	t.Run("Plan", func(t *testing.T) {
		t.Run("Bootstrap plan", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/plans/bootstrap", accessToken, map[string]any{
				"level":         "B1",
				"target_domain": "general",
				"daily_minutes": 20,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			weeklyPlan := decodeJSON[dto.WeeklyPlanResponse](t, resp)
			if weeklyPlan.WeekStart == "" {
				t.Fatal("expected non-empty week_start")
			}
			if len(weeklyPlan.DailyPlans) != 7 {
				t.Fatalf("expected 7 daily_plans, got %d", len(weeklyPlan.DailyPlans))
			}
			for i, dp := range weeklyPlan.DailyPlans {
				if len(dp.Tasks) != 2 {
					t.Fatalf("day %d: expected 2 tasks, got %d", i, len(dp.Tasks))
				}
			}

			// Capture IDs for subsequent tests
			planID = weeklyPlan.DailyPlans[0].PlanID
			firstTaskID = weeklyPlan.DailyPlans[0].Tasks[0].TaskID
		})

		t.Run("Bootstrap duplicate", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/plans/bootstrap", accessToken, map[string]any{
				"level":         "B1",
				"target_domain": "general",
				"daily_minutes": 20,
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("expected 409, got %d", resp.StatusCode)
			}
		})

		t.Run("Get today", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/plans/today", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			todayResp := decodeJSON[dto.DailyPlanResponse](t, resp)
			if todayResp.DailyPlan.Date == "" {
				t.Fatal("expected non-empty date")
			}
		})

		t.Run("Complete task", func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/plans/%s/tasks/%s/complete", planID, firstTaskID)
			resp := env.doRequest(t, "POST", path, accessToken, map[string]any{
				"completed_at": time.Now().UTC().Format(time.RFC3339),
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			result := decodeJSON[dto.TaskCompletionResponse](t, resp)
			if result.Status != "completed" {
				t.Fatalf("expected status=completed, got %s", result.Status)
			}
		})

		t.Run("Complete nonexistent task", func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/plans/%s/tasks/%s/complete", planID, uuid.New().String())
			resp := env.doRequest(t, "POST", path, accessToken, map[string]any{
				"completed_at": time.Now().UTC().Format(time.RFC3339),
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", resp.StatusCode)
			}
		})
	})

	// ==================== 5. Review Flow ====================
	t.Run("Review", func(t *testing.T) {
		cards := env.seedReviewData(t, userID, 3)

		t.Run("GET /reviews/queue", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/reviews/queue", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			queue := decodeJSON[dto.ReviewQueueResponse](t, resp)
			if queue.DueCount != 3 {
				t.Fatalf("expected due_count=3, got %d", queue.DueCount)
			}
			if len(queue.Cards) != 3 {
				t.Fatalf("expected 3 cards, got %d", len(queue.Cards))
			}
		})

		t.Run("Submit without Idempotency-Key", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/reviews/submit", accessToken, map[string]any{
				"card_id":           cards[0].CardID,
				"user_card_state_id": cards[0].StateID,
				"rating":            "good",
				"reviewed_at":       time.Now().UTC().Format(time.RFC3339),
				"client_event_id":   uuid.New().String(),
			}, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})

		idempotencyKey := uuid.New().String()

		t.Run("Submit review good", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/reviews/submit", accessToken, map[string]any{
				"card_id":           cards[0].CardID,
				"user_card_state_id": cards[0].StateID,
				"rating":            "good",
				"reviewed_at":       time.Now().UTC().Format(time.RFC3339),
				"client_event_id":   uuid.New().String(),
			}, map[string]string{
				"Idempotency-Key": idempotencyKey,
			})
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			result := decodeJSON[dto.ReviewSubmitResponse](t, resp)
			if !result.Accepted {
				t.Fatal("expected accepted=true")
			}
			if result.Status != "review" {
				t.Fatalf("expected status=review, got %s", result.Status)
			}
			if result.ScheduledDays != 1 {
				t.Fatalf("expected scheduled_days=1, got %d", result.ScheduledDays)
			}
		})

		t.Run("Submit same idempotency key", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/reviews/submit", accessToken, map[string]any{
				"card_id":           cards[0].CardID,
				"user_card_state_id": cards[0].StateID,
				"rating":            "good",
				"reviewed_at":       time.Now().UTC().Format(time.RFC3339),
				"client_event_id":   uuid.New().String(),
			}, map[string]string{
				"Idempotency-Key": idempotencyKey,
			})
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}

			result := decodeJSON[dto.ReviewSubmitResponse](t, resp)
			if !result.Accepted {
				t.Fatal("expected accepted=true for duplicate")
			}
		})

		t.Run("Submit nonexistent card", func(t *testing.T) {
			resp := env.doRequest(t, "POST", "/api/v1/reviews/submit", accessToken, map[string]any{
				"card_id":           uuid.New().String(),
				"user_card_state_id": uuid.New().String(),
				"rating":            "good",
				"reviewed_at":       time.Now().UTC().Format(time.RFC3339),
				"client_event_id":   uuid.New().String(),
			}, map[string]string{
				"Idempotency-Key": uuid.New().String(),
			})
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", resp.StatusCode)
			}
		})

		t.Run("GET /reviews/queue after submit", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/reviews/queue", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			queue := decodeJSON[dto.ReviewQueueResponse](t, resp)
			if queue.DueCount != 2 {
				t.Fatalf("expected due_count=2 after reviewing one card, got %d", queue.DueCount)
			}
		})
	})

	// ==================== 6. Progress Flow ====================
	t.Run("Progress", func(t *testing.T) {
		env.seedProgressData(t, userID)

		t.Run("GET /progress/summary range=7d", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/progress/summary?range=7d", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			summary := decodeJSON[dto.ProgressSummaryResponse](t, resp)
			if summary.TotalMinutes <= 0 {
				t.Fatalf("expected total_minutes > 0, got %d", summary.TotalMinutes)
			}
			if summary.ActiveDays <= 0 {
				t.Fatalf("expected active_days > 0, got %d", summary.ActiveDays)
			}
		})

		t.Run("GET /progress/summary no range", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/progress/summary", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})

		t.Run("GET /progress/daily with params", func(t *testing.T) {
			now := time.Now().UTC()
			from := now.AddDate(0, 0, -7).Format("2006-01-02")
			to := now.Format("2006-01-02")

			resp := env.doRequest(t, "GET", fmt.Sprintf("/api/v1/progress/daily?from=%s&to=%s", from, to), accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
			}

			daily := decodeJSON[dto.ProgressDailyResponse](t, resp)
			if len(daily.Points) == 0 {
				t.Fatal("expected non-empty points array")
			}
		})

		t.Run("GET /progress/daily no params", func(t *testing.T) {
			resp := env.doRequest(t, "GET", "/api/v1/progress/daily", accessToken, nil, nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	})
}

// ==================== Pack seed helpers ====================

// seedPack inserts a resource_pack row.
func (e *testEnv) seedPack(t *testing.T, id, source, title, domain, level string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := e.db.Exec(
		`INSERT INTO resource_packs (id, source, title, description, domain, level, estimated_minutes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 20, ?)`,
		id, source, title, title+" description", domain, level, now,
	)
	if err != nil {
		t.Fatalf("seed pack %s: %v", id, err)
	}
}

// seedLesson inserts a lesson row.
func (e *testEnv) seedLesson(t *testing.T, id, packID, title, lessonType string, position int) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := e.db.Exec(
		`INSERT INTO lessons (id, pack_id, title, lesson_type, position, estimated_minutes, created_at)
		 VALUES (?, ?, ?, ?, ?, 10, ?)`,
		id, packID, title, lessonType, position, now,
	)
	if err != nil {
		t.Fatalf("seed lesson %s: %v", id, err)
	}
}

// seedCard inserts a card row attached to a lesson.
func (e *testEnv) seedCard(t *testing.T, id, lessonID, front, back string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := e.db.Exec(
		`INSERT INTO cards (id, lesson_id, front_text, back_text, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, lessonID, front, back, now,
	)
	if err != nil {
		t.Fatalf("seed card %s: %v", id, err)
	}
}

// ==================== Pack API Tests ====================

func TestPackAPI(t *testing.T) {
	env := newTestEnv(t)

	authResp := env.registerUser(t, "pack-test@test.com", "securepassword123")
	accessToken := authResp.Tokens.AccessToken

	// 1. List packs (empty)
	t.Run("List packs empty", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackListResponse](t, resp)
		if result.Total != 0 {
			t.Fatalf("expected total=0, got %d", result.Total)
		}
		if len(result.Items) != 0 {
			t.Fatalf("expected 0 items, got %d", len(result.Items))
		}
	})

	// 2. Seed 3 packs with lessons and cards
	pack1ID := uuid.New().String()
	pack2ID := uuid.New().String()
	pack3ID := uuid.New().String()

	lesson1ID := uuid.New().String()
	lesson2ID := uuid.New().String()
	lesson3ID := uuid.New().String()

	card1ID := uuid.New().String()
	card2ID := uuid.New().String()
	card3ID := uuid.New().String()

	env.seedPack(t, pack1ID, "official", "Tech Pack", "tech", "B1")
	env.seedPack(t, pack2ID, "official", "Travel Pack", "travel", "A2")
	env.seedPack(t, pack3ID, "ai", "General Pack", "general", "B1")

	env.seedLesson(t, lesson1ID, pack1ID, "Lesson 1", "reading", 1)
	env.seedLesson(t, lesson2ID, pack1ID, "Lesson 2", "listening", 2)
	env.seedLesson(t, lesson3ID, pack2ID, "Lesson 1", "mixed", 1)

	env.seedCard(t, card1ID, lesson1ID, "front-1", "back-1")
	env.seedCard(t, card2ID, lesson1ID, "front-2", "back-2")
	env.seedCard(t, card3ID, lesson2ID, "front-3", "back-3")

	// 3. List packs (all)
	t.Run("List packs all", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackListResponse](t, resp)
		if result.Total != 3 {
			t.Fatalf("expected total=3, got %d", result.Total)
		}
		if len(result.Items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(result.Items))
		}
	})

	// 4. List packs filter domain=tech
	t.Run("List packs filter domain", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs?domain=tech", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackListResponse](t, resp)
		if result.Total != 1 {
			t.Fatalf("expected total=1, got %d", result.Total)
		}
	})

	// 5. List packs filter level=B1
	t.Run("List packs filter level", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs?level=B1", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackListResponse](t, resp)
		if result.Total != 2 {
			t.Fatalf("expected total=2, got %d", result.Total)
		}
	})

	// 6. List packs pagination page_size=2
	t.Run("List packs pagination", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs?page_size=2", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackListResponse](t, resp)
		if result.Total != 3 {
			t.Fatalf("expected total=3, got %d", result.Total)
		}
		if len(result.Items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result.Items))
		}
		if result.PageSize != 2 {
			t.Fatalf("expected page_size=2, got %d", result.PageSize)
		}
	})

	// 7. Get pack detail
	t.Run("Get pack detail", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs/"+pack1ID, accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.PackDetailResponse](t, resp)
		if result.Pack.ID != pack1ID {
			t.Fatalf("expected pack id=%s, got %s", pack1ID, result.Pack.ID)
		}
		if result.Pack.Title != "Tech Pack" {
			t.Fatalf("expected title=Tech Pack, got %s", result.Pack.Title)
		}
		if len(result.Lessons) != 2 {
			t.Fatalf("expected 2 lessons, got %d", len(result.Lessons))
		}
		if result.Lessons[0].Position != 1 {
			t.Fatalf("expected first lesson position=1, got %d", result.Lessons[0].Position)
		}
	})

	// 8. Get nonexistent pack
	t.Run("Get nonexistent pack", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs/"+uuid.New().String(), accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	// 9. Enroll pack
	t.Run("Enroll pack", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/"+pack1ID+"/enroll", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.GenericMessage](t, resp)
		if result.Message == "" {
			t.Fatal("expected non-empty message")
		}
	})

	// 10. Enroll same pack again (idempotent)
	t.Run("Enroll same pack again", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/"+pack1ID+"/enroll", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 (idempotent), got %d: %s", resp.StatusCode, body)
		}
	})

	// 11. Review queue shows enrolled cards
	t.Run("Review queue shows enrolled cards", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/reviews/queue", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		queue := decodeJSON[dto.ReviewQueueResponse](t, resp)
		if queue.DueCount < 3 {
			t.Fatalf("expected due_count >= 3, got %d", queue.DueCount)
		}
	})

	// 12. Create generation job (with days omitted -> default 7)
	var jobID string
	t.Run("Create generation job without days", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 20,
			"focus_skills":  []string{"reading", "listening"},
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 202, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.GenerationJobResponse](t, resp)
		if result.JobID == "" {
			t.Fatal("expected non-empty job_id")
		}
		if result.Status != "queued" {
			t.Fatalf("expected status=queued, got %s", result.Status)
		}
		jobID = result.JobID
	})

	// 12b. Create generation job with explicit days
	t.Run("Create generation job with days", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 20,
			"days":          7,
			"focus_skills":  []string{"reading", "listening"},
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 202, got %d: %s", resp.StatusCode, body)
		}
	})

	// 12c. Third generation job → rate limit (2 per day)
	t.Run("Create generation job rate limited", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 20,
			"focus_skills":  []string{"reading"},
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 429, got %d: %s", resp.StatusCode, body)
		}

		errResp := decodeJSON[dto.ErrorResponse](t, resp)
		if errResp.Code != "RATE_LIMIT" {
			t.Fatalf("expected code=RATE_LIMIT, got %s", errResp.Code)
		}
	})

	// 13. Create generation job bad input (invalid level)
	t.Run("Create generation job bad input", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level": "INVALID",
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 13b. Create generation job daily_minutes too low (4 < min 5)
	t.Run("Create generation job daily_minutes too low", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 4,
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 13c. Create generation job days too low (2 < min 3)
	t.Run("Create generation job days too low", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 20,
			"days":          2,
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 13d. Create generation job days too high (15 > max 14)
	t.Run("Create generation job days too high", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/packs/generate", accessToken, map[string]any{
			"level":         "B1",
			"domain":        "tech",
			"daily_minutes": 20,
			"days":          15,
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 14. Get generation job
	t.Run("Get generation job", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs/generation-jobs/"+jobID, accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.GenerationJobResponse](t, resp)
		if result.JobID != jobID {
			t.Fatalf("expected job_id=%s, got %s", jobID, result.JobID)
		}
		if result.Status != "queued" {
			t.Fatalf("expected status=queued, got %s", result.Status)
		}
	})

	// 15. Get nonexistent job
	t.Run("Get nonexistent job", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/packs/generation-jobs/"+uuid.New().String(), accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})
}

// ==================== Concurrency Race Tests ====================

// TestReviewSubmitRace fires N concurrent review submissions with the same
// idempotency key. All must return 200 (no 500), and exactly one real insert
// should occur (the rest are idempotent replays).
func TestReviewSubmitRace(t *testing.T) {
	env := newTestEnv(t)

	auth := env.registerUser(t, "race-review@test.com", "securepassword123")
	token := auth.Tokens.AccessToken
	userID := auth.User.ID

	cards := env.seedReviewData(t, userID, 1)
	card := cards[0]

	idempotencyKey := uuid.New().String()
	clientEventID := uuid.New().String()

	const goroutines = 10
	var wg sync.WaitGroup
	statuses := make([]int, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			resp := env.doRequest(t, "POST", "/api/v1/reviews/submit", token, map[string]any{
				"card_id":            card.CardID,
				"user_card_state_id": card.StateID,
				"rating":             "good",
				"reviewed_at":        time.Now().UTC().Format(time.RFC3339),
				"client_event_id":    clientEventID,
			}, map[string]string{
				"Idempotency-Key": idempotencyKey,
			})
			defer resp.Body.Close()
			io.ReadAll(resp.Body) // drain
			statuses[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	for i, code := range statuses {
		if code != http.StatusOK {
			t.Errorf("goroutine %d: expected 200, got %d", i, code)
		}
	}

	// Verify exactly 1 review log was inserted
	var count int
	err := env.db.QueryRow(
		"SELECT COUNT(*) FROM review_logs WHERE idempotency_key = ?", idempotencyKey,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count review logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 review_log row, got %d", count)
	}
}

// TestBootstrapPlanRace fires N concurrent bootstrap requests for the same user.
// Exactly one should succeed with 200; all others must get 409 (no 500).
func TestBootstrapPlanRace(t *testing.T) {
	env := newTestEnv(t)

	auth := env.registerUser(t, "race-plan@test.com", "securepassword123")
	token := auth.Tokens.AccessToken

	const goroutines = 10
	var wg sync.WaitGroup
	statuses := make([]int, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			resp := env.doRequest(t, "POST", "/api/v1/plans/bootstrap", token, map[string]any{
				"level":         "B1",
				"target_domain": "general",
				"daily_minutes": 20,
			}, nil)
			defer resp.Body.Close()
			io.ReadAll(resp.Body) // drain
			statuses[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	okCount := 0
	conflictCount := 0
	for i, code := range statuses {
		switch code {
		case http.StatusOK:
			okCount++
		case http.StatusConflict:
			conflictCount++
		default:
			t.Errorf("goroutine %d: unexpected status %d", i, code)
		}
	}

	if okCount != 1 {
		t.Errorf("expected exactly 1 success (200), got %d", okCount)
	}
	if conflictCount != goroutines-1 {
		t.Errorf("expected %d conflicts (409), got %d", goroutines-1, conflictCount)
	}
}

// TestGenerateJobRace fires N concurrent generation requests for the same user.
// At most 2 should succeed with 202; the rest must get 429 (no 500).
func TestGenerateJobRace(t *testing.T) {
	env := newTestEnv(t)

	auth := env.registerUser(t, "race-generate@test.com", "securepassword123")
	token := auth.Tokens.AccessToken

	const goroutines = 10
	var wg sync.WaitGroup
	statuses := make([]int, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			resp := env.doRequest(t, "POST", "/api/v1/packs/generate", token, map[string]any{
				"level":         "B1",
				"domain":        "tech",
				"daily_minutes": 20,
				"focus_skills":  []string{"reading"},
			}, nil)
			defer resp.Body.Close()
			io.ReadAll(resp.Body) // drain
			statuses[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	acceptedCount := 0
	rateLimitedCount := 0
	for i, code := range statuses {
		switch code {
		case http.StatusAccepted:
			acceptedCount++
		case http.StatusTooManyRequests:
			rateLimitedCount++
		default:
			t.Errorf("goroutine %d: unexpected status %d", i, code)
		}
	}

	if acceptedCount > 2 {
		t.Errorf("expected at most 2 accepted (202), got %d", acceptedCount)
	}
	if acceptedCount < 1 {
		t.Errorf("expected at least 1 accepted (202), got %d", acceptedCount)
	}

	// Verify DB has at most 2 jobs
	var jobCount int
	err := env.db.QueryRow("SELECT COUNT(*) FROM ai_generation_jobs").Scan(&jobCount)
	if err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if jobCount > 2 {
		t.Fatalf("expected at most 2 jobs in DB, got %d", jobCount)
	}
}

// ==================== Output Task seed helpers ====================

// seedOutputTask inserts an output_task row.
func (e *testEnv) seedOutputTask(t *testing.T, id, lessonID, taskType, promptText, referenceAnswer, level string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	var refAnswer, lvl sql.NullString
	if referenceAnswer != "" {
		refAnswer = sql.NullString{String: referenceAnswer, Valid: true}
	}
	if level != "" {
		lvl = sql.NullString{String: level, Valid: true}
	}
	_, err := e.db.Exec(
		`INSERT INTO output_tasks (id, lesson_id, task_type, prompt_text, reference_answer, level, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, lessonID, taskType, promptText, refAnswer, lvl, now,
	)
	if err != nil {
		t.Fatalf("seed output task %s: %v", id, err)
	}
}

// ==================== Output Task API Tests ====================

func TestOutputTaskAPI(t *testing.T) {
	env := newTestEnv(t)

	authResp := env.registerUser(t, "output-test@test.com", "securepassword123")
	accessToken := authResp.Tokens.AccessToken
	userID := authResp.User.ID

	// Seed a pack, lesson, and output tasks
	packID := uuid.New().String()
	lessonID := uuid.New().String()
	emptyLessonID := uuid.New().String()
	writingTaskID := uuid.New().String()
	readingTaskID := uuid.New().String()

	env.seedPack(t, packID, "official", "Output Test Pack", "general", "B1")
	env.seedLesson(t, lessonID, packID, "Lesson with tasks", "writing", 1)
	env.seedLesson(t, emptyLessonID, packID, "Empty lesson", "reading", 2)

	env.seedOutputTask(t, writingTaskID, lessonID, "writing", "Write a paragraph about your daily routine.", "I wake up at 7am every morning.", "B1")
	env.seedOutputTask(t, readingTaskID, lessonID, "reading", "Read and summarize the passage.", "", "B1")

	// 1. List writing tasks for lesson → 200, only writing tasks returned
	t.Run("List writing tasks for lesson", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/lessons/"+lessonID+"/output-tasks", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.OutputTaskListResponse](t, resp)
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 writing task, got %d", len(result.Items))
		}
		if result.Items[0].ID != writingTaskID {
			t.Fatalf("expected task id=%s, got %s", writingTaskID, result.Items[0].ID)
		}
		if result.Items[0].TaskType != "writing" {
			t.Fatalf("expected task_type=writing, got %s", result.Items[0].TaskType)
		}
	})

	// 2. List tasks for empty lesson → 200, empty array
	t.Run("List tasks for empty lesson", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/lessons/"+emptyLessonID+"/output-tasks", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.OutputTaskListResponse](t, resp)
		if len(result.Items) != 0 {
			t.Fatalf("expected 0 tasks, got %d", len(result.Items))
		}
	})

	// 3. Submit writing → 200, feedback + score present
	var submissionID string
	t.Run("Submit writing", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/output-tasks/"+writingTaskID+"/submit", accessToken, map[string]any{
			"answer_text": "I go to school yesterday and I eat lunch with my friend.",
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.SubmissionResponse](t, resp)
		if result.SubmissionID == "" {
			t.Fatal("expected non-empty submission_id")
		}
		if result.Feedback == nil {
			t.Fatal("expected non-nil feedback")
		}
		if result.Feedback.OverallScore != 85 {
			t.Fatalf("expected overall_score=85, got %d", result.Feedback.OverallScore)
		}
		if result.Score != 85 {
			t.Fatalf("expected score=85, got %f", result.Score)
		}
		if len(result.Feedback.Errors) == 0 {
			t.Fatal("expected at least 1 error in feedback")
		}
		if result.Feedback.RevisedText == "" {
			t.Fatal("expected non-empty revised_text")
		}
		submissionID = result.SubmissionID
	})

	// 4. Submit nonexistent task → 404
	t.Run("Submit nonexistent task", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/output-tasks/"+uuid.New().String()+"/submit", accessToken, map[string]any{
			"answer_text": "Some text.",
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	// 5. Submit non-writing task → 400
	t.Run("Submit non-writing task", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/output-tasks/"+readingTaskID+"/submit", accessToken, map[string]any{
			"answer_text": "Some text.",
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 6. Submit empty answer → 400
	t.Run("Submit empty answer", func(t *testing.T) {
		resp := env.doRequest(t, "POST", "/api/v1/output-tasks/"+writingTaskID+"/submit", accessToken, map[string]any{
			"answer_text": "",
		}, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 7. Get submission → 200
	t.Run("Get submission", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/output-tasks/submissions/"+submissionID, accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.SubmissionResponse](t, resp)
		if result.SubmissionID != submissionID {
			t.Fatalf("expected submission_id=%s, got %s", submissionID, result.SubmissionID)
		}
		if result.Feedback == nil {
			t.Fatal("expected non-nil feedback")
		}
		if result.Score != 85 {
			t.Fatalf("expected score=85, got %f", result.Score)
		}
	})

	// 8. Get nonexistent submission → 404
	t.Run("Get nonexistent submission", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/output-tasks/submissions/"+uuid.New().String(), accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	// 9. Get submission as wrong user → 404
	t.Run("Get submission as wrong user", func(t *testing.T) {
		otherAuth := env.registerUser(t, "other-user@test.com", "securepassword123")
		otherToken := otherAuth.Tokens.AccessToken

		resp := env.doRequest(t, "GET", "/api/v1/output-tasks/submissions/"+submissionID, otherToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.StatusCode)
		}
	})

	// 10. Verify progress_daily.writing_tasks_completed incremented
	t.Run("Verify progress incremented", func(t *testing.T) {
		today := time.Now().UTC().Format("2006-01-02")
		var count int
		err := env.db.QueryRow(
			"SELECT writing_tasks_completed FROM progress_daily WHERE user_id = ? AND progress_date = ?",
			userID, today,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query progress: %v", err)
		}
		if count < 1 {
			t.Fatalf("expected writing_tasks_completed >= 1, got %d", count)
		}
	})
}

// ==================== Sync Endpoint Tests ====================

func TestSyncEvents_PushAndPull(t *testing.T) {
	env := newTestEnv(t)
	authResp := env.registerUser(t, "sync@test.com", "password123")
	token := authResp.Tokens.AccessToken

	// Push events
	body := map[string]any{
		"events": []map[string]any{
			{
				"client_event_id": uuid.New().String(),
				"event_type":      "review_submitted",
				"occurred_at":     time.Now().UTC().Format(time.RFC3339),
				"payload":         map[string]string{"card_id": "abc", "rating": "good"},
			},
			{
				"client_event_id": uuid.New().String(),
				"event_type":      "task_completed",
				"occurred_at":     time.Now().UTC().Format(time.RFC3339),
				"payload":         map[string]string{"task_id": "xyz"},
			},
		},
	}

	resp := env.doRequest(t, "POST", "/api/v1/sync/events", token, body, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var syncResp dto.SyncEventsResponse
	json.NewDecoder(resp.Body).Decode(&syncResp)

	if len(syncResp.Acks) != 2 {
		t.Fatalf("expected 2 acks, got %d", len(syncResp.Acks))
	}
	for _, ack := range syncResp.Acks {
		if ack.Status != "accepted" {
			t.Fatalf("expected accepted, got %s", ack.Status)
		}
	}
	if syncResp.ServerCursor == "" {
		t.Fatal("expected non-empty server_cursor")
	}

	// Pull changes
	pullResp := env.doRequest(t, "GET", "/api/v1/sync/changes", token, nil, nil)
	defer pullResp.Body.Close()

	if pullResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", pullResp.StatusCode)
	}

	var changesResp dto.SyncChangesResponse
	json.NewDecoder(pullResp.Body).Decode(&changesResp)

	if len(changesResp.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changesResp.Changes))
	}
	if changesResp.NextCursor == "" {
		t.Fatal("expected non-empty next_cursor")
	}
}

func TestSyncEvents_DuplicateDedup(t *testing.T) {
	env := newTestEnv(t)
	authResp := env.registerUser(t, "syncdedup@test.com", "password123")
	token := authResp.Tokens.AccessToken

	clientEventID := uuid.New().String()
	body := map[string]any{
		"events": []map[string]any{
			{
				"client_event_id": clientEventID,
				"event_type":      "review_submitted",
				"occurred_at":     time.Now().UTC().Format(time.RFC3339),
				"payload":         map[string]string{"card_id": "abc"},
			},
		},
	}

	// First push
	resp1 := env.doRequest(t, "POST", "/api/v1/sync/events", token, body, nil)
	defer resp1.Body.Close()

	var syncResp1 dto.SyncEventsResponse
	json.NewDecoder(resp1.Body).Decode(&syncResp1)
	if syncResp1.Acks[0].Status != "accepted" {
		t.Fatalf("expected accepted, got %s", syncResp1.Acks[0].Status)
	}

	// Second push (duplicate)
	resp2 := env.doRequest(t, "POST", "/api/v1/sync/events", token, body, nil)
	defer resp2.Body.Close()

	var syncResp2 dto.SyncEventsResponse
	json.NewDecoder(resp2.Body).Decode(&syncResp2)
	if syncResp2.Acks[0].Status != "duplicate" {
		t.Fatalf("expected duplicate, got %s", syncResp2.Acks[0].Status)
	}
}

func TestSyncEvents_RejectInvalidType(t *testing.T) {
	env := newTestEnv(t)
	authResp := env.registerUser(t, "syncreject@test.com", "password123")
	token := authResp.Tokens.AccessToken

	body := map[string]any{
		"events": []map[string]any{
			{
				"client_event_id": uuid.New().String(),
				"event_type":      "invalid_type",
				"occurred_at":     time.Now().UTC().Format(time.RFC3339),
				"payload":         map[string]string{"foo": "bar"},
			},
		},
	}

	resp := env.doRequest(t, "POST", "/api/v1/sync/events", token, body, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var syncResp dto.SyncEventsResponse
	json.NewDecoder(resp.Body).Decode(&syncResp)
	if syncResp.Acks[0].Status != "rejected" {
		t.Fatalf("expected rejected, got %s", syncResp.Acks[0].Status)
	}
	if syncResp.Acks[0].Reason != "UNKNOWN_EVENT_TYPE" {
		t.Fatalf("expected UNKNOWN_EVENT_TYPE, got %s", syncResp.Acks[0].Reason)
	}
}

func TestSyncEvents_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	body := map[string]any{
		"events": []map[string]any{
			{
				"client_event_id": uuid.New().String(),
				"event_type":      "review_submitted",
				"occurred_at":     time.Now().UTC().Format(time.RFC3339),
				"payload":         map[string]string{"card_id": "abc"},
			},
		},
	}

	// No auth token
	resp := env.doRequest(t, "POST", "/api/v1/sync/events", "", body, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestSyncChanges_CursorPagination(t *testing.T) {
	env := newTestEnv(t)
	authResp := env.registerUser(t, "syncpage@test.com", "password123")
	token := authResp.Tokens.AccessToken

	// Push 3 events
	for i := 0; i < 3; i++ {
		body := map[string]any{
			"events": []map[string]any{
				{
					"client_event_id": uuid.New().String(),
					"event_type":      "review_submitted",
					"occurred_at":     time.Now().UTC().Format(time.RFC3339),
					"payload":         map[string]string{"card_id": fmt.Sprintf("card_%d", i)},
				},
			},
		}
		resp := env.doRequest(t, "POST", "/api/v1/sync/events", token, body, nil)
		resp.Body.Close()
	}

	// Pull with limit=2
	pullResp := env.doRequest(t, "GET", "/api/v1/sync/changes?limit=2", token, nil, nil)
	defer pullResp.Body.Close()

	var page1 dto.SyncChangesResponse
	json.NewDecoder(pullResp.Body).Decode(&page1)
	if len(page1.Changes) != 2 {
		t.Fatalf("expected 2 changes in page 1, got %d", len(page1.Changes))
	}

	// Pull remaining
	pullResp2 := env.doRequest(t, "GET", "/api/v1/sync/changes?cursor="+page1.NextCursor+"&limit=2", token, nil, nil)
	defer pullResp2.Body.Close()

	var page2 dto.SyncChangesResponse
	json.NewDecoder(pullResp2.Body).Decode(&page2)
	if len(page2.Changes) != 1 {
		t.Fatalf("expected 1 change in page 2, got %d", len(page2.Changes))
	}
}

// ==================== Weekly Report Tests ====================

func TestWeeklyReport(t *testing.T) {
	env := newTestEnv(t)

	authResp := env.registerUser(t, "weekly-report@test.com", "securepassword123")
	accessToken := authResp.Tokens.AccessToken
	userID := authResp.User.ID

	// Use a known Monday for deterministic tests
	weekStart := "2026-02-23" // Monday
	prevWeekStart := "2026-02-16"

	// Seed current week progress_daily (Mon-Sun: Feb 23–Mar 1)
	for i := 0; i < 5; i++ {
		date := time.Date(2026, 2, 23+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		_, err := env.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
			 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, date, 30, 2, 5, 10+i, 0.85, 10, 1, 1, i+1,
		)
		if err != nil {
			t.Fatalf("seed current week day %d: %v", i, err)
		}
	}

	// Seed previous week progress_daily (Mon-Sun: Feb 16–22)
	for i := 0; i < 3; i++ {
		date := time.Date(2026, 2, 16+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		_, err := env.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
			 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, date, 20, 1, 3, 8, 0.80, 5, 0, 0, i+1,
		)
		if err != nil {
			t.Fatalf("seed previous week day %d: %v", i, err)
		}
	}

	// Seed review_logs in current week
	cards := env.seedReviewData(t, userID, 1)
	cardID := cards[0].CardID
	stateID := cards[0].StateID
	ratings := []string{"again", "hard", "good", "good", "easy"}
	for i, rating := range ratings {
		reviewedAt := time.Date(2026, 2, 23+i%5, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
		_, err := env.db.Exec(
			`INSERT INTO review_logs (user_id, card_id, user_card_state_id, rating, reviewed_at, client_event_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			userID, cardID, stateID, rating, reviewedAt, uuid.New().String(),
		)
		if err != nil {
			t.Fatalf("seed review_log %d: %v", i, err)
		}
	}

	// Set weekly_goal_days in user_learning_profiles
	_, err := env.db.Exec(
		`UPDATE user_learning_profiles SET weekly_goal_days = 4 WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		t.Fatalf("update learning profile: %v", err)
	}

	// Test 1: Valid week_start with seeded data → 200
	t.Run("Valid weekly report with data", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report?week_start="+weekStart, accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.WeeklyReportResponse](t, resp)

		if result.WeekStart != weekStart {
			t.Fatalf("expected week_start=%s, got %s", weekStart, result.WeekStart)
		}
		// 5 days * 30 minutes = 150
		if result.TotalMinutes != 150 {
			t.Fatalf("expected total_minutes=150, got %d", result.TotalMinutes)
		}
		if result.ActiveDays != 5 {
			t.Fatalf("expected active_days=5, got %d", result.ActiveDays)
		}
		if result.LessonsCompleted != 10 {
			t.Fatalf("expected lessons_completed=10, got %d", result.LessonsCompleted)
		}
		if result.Streak != 5 {
			t.Fatalf("expected streak=5, got %d", result.Streak)
		}
		if result.WeeklyGoalDays != 4 {
			t.Fatalf("expected weekly_goal_days=4, got %d", result.WeeklyGoalDays)
		}
		if !result.GoalAchieved {
			t.Fatal("expected goal_achieved=true")
		}

		// Review health
		if result.ReviewHealth.Again != 1 {
			t.Fatalf("expected again=1, got %d", result.ReviewHealth.Again)
		}
		if result.ReviewHealth.Hard != 1 {
			t.Fatalf("expected hard=1, got %d", result.ReviewHealth.Hard)
		}
		if result.ReviewHealth.Good != 2 {
			t.Fatalf("expected good=2, got %d", result.ReviewHealth.Good)
		}
		if result.ReviewHealth.Easy != 1 {
			t.Fatalf("expected easy=1, got %d", result.ReviewHealth.Easy)
		}
		if result.ReviewHealth.Total != 5 {
			t.Fatalf("expected total=5, got %d", result.ReviewHealth.Total)
		}
		if result.ReviewHealth.Accuracy == nil {
			t.Fatal("expected non-nil accuracy")
		}
		// accuracy = (1+2+1)/5 = 0.8
		if *result.ReviewHealth.Accuracy < 0.79 || *result.ReviewHealth.Accuracy > 0.81 {
			t.Fatalf("expected accuracy~0.8, got %f", *result.ReviewHealth.Accuracy)
		}

		// Daily breakdown
		if len(result.DailyBreakdown) != 5 {
			t.Fatalf("expected 5 daily points, got %d", len(result.DailyBreakdown))
		}

		// Previous week comparison present
		if result.PreviousWeekComparison == nil {
			t.Fatal("expected previous_week_comparison to be present")
		}
		// current 150min - prev 60min = 90
		if result.PreviousWeekComparison.MinutesDelta != 90 {
			t.Fatalf("expected minutes_delta=90, got %d", result.PreviousWeekComparison.MinutesDelta)
		}
		// current 5 active - prev 3 active = 2
		if result.PreviousWeekComparison.ActiveDaysDelta != 2 {
			t.Fatalf("expected active_days_delta=2, got %d", result.PreviousWeekComparison.ActiveDaysDelta)
		}
	})

	// Test 2: Missing week_start → 400
	t.Run("Missing week_start", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// Test 3: Non-Monday week_start → 400
	t.Run("Non-Monday week_start", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report?week_start=2026-02-24", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// Test 4: No data for week → 200 with zeroed aggregates
	t.Run("No data for week", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report?week_start=2025-01-06", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.WeeklyReportResponse](t, resp)
		if result.TotalMinutes != 0 {
			t.Fatalf("expected total_minutes=0, got %d", result.TotalMinutes)
		}
		if result.ActiveDays != 0 {
			t.Fatalf("expected active_days=0, got %d", result.ActiveDays)
		}
		if result.GoalAchieved {
			t.Fatal("expected goal_achieved=false")
		}
		if len(result.DailyBreakdown) != 0 {
			t.Fatalf("expected 0 daily points, got %d", len(result.DailyBreakdown))
		}
		if result.PreviousWeekComparison != nil {
			t.Fatal("expected previous_week_comparison to be nil")
		}
	})

	// Test 5: Previous week has data, verify comparison for prev week start
	t.Run("Report for previous week has no comparison", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report?week_start="+prevWeekStart, accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.WeeklyReportResponse](t, resp)
		if result.TotalMinutes != 60 {
			t.Fatalf("expected total_minutes=60, got %d", result.TotalMinutes)
		}
		if result.PreviousWeekComparison != nil {
			t.Fatal("expected previous_week_comparison to be nil (no week before)")
		}
	})

	// Test 6: Previous week has rows with zero minutes but non-zero non-minute metrics
	t.Run("Prev week zero minutes but non-zero writing tasks", func(t *testing.T) {
		// Seed a week where prev week (Mar 2) has zero minutes but writing_tasks_completed > 0
		// Current week: Mar 9 (Monday), prev week: Mar 2 (Monday)
		_, err := env.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
			 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
			 VALUES (?, '2026-03-02', 0, 0, 0, 0, NULL, 0, 0, 3, 0)`,
			userID,
		)
		if err != nil {
			t.Fatalf("seed zero-minutes prev week: %v", err)
		}

		resp := env.doRequest(t, "GET", "/api/v1/progress/weekly-report?week_start=2026-03-09", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.WeeklyReportResponse](t, resp)
		if result.PreviousWeekComparison == nil {
			t.Fatal("expected previous_week_comparison to be present (prev week has rows with non-zero writing_tasks)")
		}
	})
}

func TestMonthlyReport(t *testing.T) {
	env := newTestEnv(t)

	authResp := env.registerUser(t, "monthly-report@test.com", "securepassword123")
	accessToken := authResp.Tokens.AccessToken
	userID := authResp.User.ID

	// Seed February 2026 data (28 days) — 10 days with varied skill data
	for i := 0; i < 10; i++ {
		date := time.Date(2026, 2, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		_, err := env.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
			 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, date, 30, 2, 5, 10, 0.85, 15, 3, 2, i+1,
		)
		if err != nil {
			t.Fatalf("seed feb day %d: %v", i, err)
		}
	}

	// Seed January 2026 data (previous month) — 5 days with less data
	for i := 0; i < 5; i++ {
		date := time.Date(2026, 1, 10+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		_, err := env.db.Exec(
			`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
			 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, date, 20, 1, 3, 8, 0.80, 10, 2, 1, i+1,
		)
		if err != nil {
			t.Fatalf("seed jan day %d: %v", i, err)
		}
	}

	// Seed review_logs in Feb 2026
	cards := env.seedReviewData(t, userID, 1)
	cardID := cards[0].CardID
	stateID := cards[0].StateID
	ratings := []string{"again", "hard", "good", "good", "easy"}
	for i, rating := range ratings {
		reviewedAt := time.Date(2026, 2, 1+i, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
		_, err := env.db.Exec(
			`INSERT INTO review_logs (user_id, card_id, user_card_state_id, rating, reviewed_at, client_event_id)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			userID, cardID, stateID, rating, reviewedAt, uuid.New().String(),
		)
		if err != nil {
			t.Fatalf("seed review_log %d: %v", i, err)
		}
	}

	// Set weekly_goal_days
	_, err := env.db.Exec(
		`UPDATE user_learning_profiles SET weekly_goal_days = 4 WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		t.Fatalf("update learning profile: %v", err)
	}

	// Test 1: Valid month with seeded data → 200
	t.Run("Valid monthly report with data", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2026-02", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.MonthlyReportResponse](t, resp)

		if result.Month != "2026-02" {
			t.Fatalf("expected month=2026-02, got %s", result.Month)
		}
		if result.DaysInMonth != 28 {
			t.Fatalf("expected days_in_month=28, got %d", result.DaysInMonth)
		}
		// 10 days * 30 minutes = 300
		if result.TotalMinutes != 300 {
			t.Fatalf("expected total_minutes=300, got %d", result.TotalMinutes)
		}
		if result.ActiveDays != 10 {
			t.Fatalf("expected active_days=10, got %d", result.ActiveDays)
		}
		// 10 days * 2 lessons = 20
		if result.LessonsCompleted != 20 {
			t.Fatalf("expected lessons_completed=20, got %d", result.LessonsCompleted)
		}
		// 10 days * 10 cards_reviewed = 100
		if result.CardsReviewed != 100 {
			t.Fatalf("expected cards_reviewed=100, got %d", result.CardsReviewed)
		}
		// streak = max(streak_count) = 10
		if result.Streak != 10 {
			t.Fatalf("expected streak=10, got %d", result.Streak)
		}
		// monthly_goal_days = 4 * ceil(28/7) = 4*4 = 16
		if result.MonthlyGoalDays != 16 {
			t.Fatalf("expected monthly_goal_days=16, got %d", result.MonthlyGoalDays)
		}
		// 10 active days < 16 goal → not achieved
		if result.GoalAchieved {
			t.Fatal("expected goal_achieved=false")
		}

		// Review health: 1 again, 1 hard, 2 good, 1 easy
		if result.ReviewHealth.Again != 1 {
			t.Fatalf("expected again=1, got %d", result.ReviewHealth.Again)
		}
		if result.ReviewHealth.Total != 5 {
			t.Fatalf("expected total=5, got %d", result.ReviewHealth.Total)
		}
		if result.ReviewHealth.Accuracy == nil {
			t.Fatal("expected non-nil accuracy")
		}

		// Daily breakdown: 10 rows
		if len(result.DailyBreakdown) != 10 {
			t.Fatalf("expected 10 daily points, got %d", len(result.DailyBreakdown))
		}

		// Skill breakdown: listening=150, speaking=30, writing=20, reading=100+20=120
		if result.SkillBreakdown.Listening.Value != 150 {
			t.Fatalf("expected listening=150, got %d", result.SkillBreakdown.Listening.Value)
		}
		if result.SkillBreakdown.Speaking.Value != 30 {
			t.Fatalf("expected speaking=30, got %d", result.SkillBreakdown.Speaking.Value)
		}
		if result.SkillBreakdown.Writing.Value != 20 {
			t.Fatalf("expected writing=20, got %d", result.SkillBreakdown.Writing.Value)
		}
		if result.SkillBreakdown.Reading.Value != 120 {
			t.Fatalf("expected reading=120, got %d", result.SkillBreakdown.Reading.Value)
		}
		// Percentages should sum to ~1.0
		totalPct := result.SkillBreakdown.Listening.Percentage +
			result.SkillBreakdown.Speaking.Percentage +
			result.SkillBreakdown.Writing.Percentage +
			result.SkillBreakdown.Reading.Percentage
		if totalPct < 0.99 || totalPct > 1.01 {
			t.Fatalf("expected percentages sum to ~1.0, got %f", totalPct)
		}

		// Previous month comparison present (Jan has data)
		if result.PreviousMonthComparison == nil {
			t.Fatal("expected previous_month_comparison to be present")
		}
		// current 300min - prev 100min = 200
		if result.PreviousMonthComparison.MinutesDelta != 200 {
			t.Fatalf("expected minutes_delta=200, got %d", result.PreviousMonthComparison.MinutesDelta)
		}
		// current 10 active - prev 5 active = 5
		if result.PreviousMonthComparison.ActiveDaysDelta != 5 {
			t.Fatalf("expected active_days_delta=5, got %d", result.PreviousMonthComparison.ActiveDaysDelta)
		}
	})

	// Test 2: Missing month → 400
	t.Run("Missing month", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// Test 3: Invalid month format → 400
	t.Run("Invalid month format", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2026-02-01", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// Test 3b: Invalid month value → 400
	t.Run("Invalid month value", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2026-13", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})

	// Test 4: No data for month → 200 with zeroed aggregates
	t.Run("No data for month", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2025-06", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.MonthlyReportResponse](t, resp)
		if result.TotalMinutes != 0 {
			t.Fatalf("expected total_minutes=0, got %d", result.TotalMinutes)
		}
		if result.ActiveDays != 0 {
			t.Fatalf("expected active_days=0, got %d", result.ActiveDays)
		}
		if result.GoalAchieved {
			t.Fatal("expected goal_achieved=false")
		}
		if len(result.DailyBreakdown) != 0 {
			t.Fatalf("expected 0 daily points, got %d", len(result.DailyBreakdown))
		}
		if len(result.Weaknesses) != 0 {
			t.Fatalf("expected 0 weaknesses (no activity at all), got %d", len(result.Weaknesses))
		}
		if result.PreviousMonthComparison != nil {
			t.Fatal("expected previous_month_comparison to be nil")
		}
	})

	// Test 5: Zero listening but non-zero writing → weakness for listening
	t.Run("Weakness detected for zero-activity skill", func(t *testing.T) {
		// Seed March 2026 with zero listening but non-zero writing
		for i := 0; i < 3; i++ {
			date := time.Date(2026, 3, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			_, err := env.db.Exec(
				`INSERT INTO progress_daily (user_id, progress_date, minutes_learned, lessons_completed, cards_new, cards_reviewed,
				 review_accuracy, listening_minutes, speaking_tasks_completed, writing_tasks_completed, streak_count)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userID, date, 20, 2, 3, 10, 0.90, 0, 0, 5, i+1,
			)
			if err != nil {
				t.Fatalf("seed march day %d: %v", i, err)
			}
		}

		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2026-03", accessToken, nil, nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		result := decodeJSON[dto.MonthlyReportResponse](t, resp)

		// Should have weaknesses for listening and speaking (both zero)
		foundListening := false
		foundSpeaking := false
		for _, w := range result.Weaknesses {
			if w.Skill == "listening" && w.Reason == "low_activity" {
				foundListening = true
			}
			if w.Skill == "speaking" && w.Reason == "low_activity" {
				foundSpeaking = true
			}
		}
		if !foundListening {
			t.Fatal("expected weakness for listening (low_activity)")
		}
		if !foundSpeaking {
			t.Fatal("expected weakness for speaking (low_activity)")
		}

		// Previous month (Feb) has data → comparison present
		if result.PreviousMonthComparison == nil {
			t.Fatal("expected previous_month_comparison to be present")
		}
	})

	// Test 6: Feb 2026 has 28 days
	t.Run("February days_in_month", func(t *testing.T) {
		resp := env.doRequest(t, "GET", "/api/v1/progress/monthly-report?month=2026-02", accessToken, nil, nil)
		defer resp.Body.Close()

		result := decodeJSON[dto.MonthlyReportResponse](t, resp)
		if result.DaysInMonth != 28 {
			t.Fatalf("expected days_in_month=28 for Feb 2026, got %d", result.DaysInMonth)
		}
	})
}
