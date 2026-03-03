package http_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/app"
	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/db"
	"github.com/bennyshi/english-anywhere-lab/internal/plan"
	"github.com/bennyshi/english-anywhere-lab/internal/progress"
	"github.com/bennyshi/english-anywhere-lab/internal/review"
	"github.com/bennyshi/english-anywhere-lab/internal/scheduler"
	httptransport "github.com/bennyshi/english-anywhere-lab/internal/transport/http"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// testEnv holds a running httptest.Server backed by the full application stack.
type testEnv struct {
	server *httptest.Server
	db     *sql.DB
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
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

	router := httptransport.NewRouter(application, authSvc, authJWT, reviewSvc, planSvc, progressSvc)
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
