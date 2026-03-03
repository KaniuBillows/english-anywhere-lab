package auth

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/bennyshi/english-anywhere-lab/internal/db"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	cfg := &config.Config{
		SQLitePath:        ":memory:",
		SQLiteWAL:         false,
		SQLiteBusyTimeout: 5000,
	}
	// For in-memory, open directly
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	database.Exec("PRAGMA foreign_keys = ON")

	_ = cfg // used for context

	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func testConfig() *config.Config {
	return &config.Config{
		JWTIssuer:     "test",
		JWTAccessTTL:  60 * time.Minute,
		JWTRefreshTTL: 720 * time.Hour,
		JWTSignKey:    "test-secret-key-at-least-32-chars",
	}
}

func TestRegisterAndLogin(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewRepository(database)
	jwt := NewJWTManager(testConfig())
	svc := NewService(repo, jwt)

	ctx := context.Background()

	// Register
	result, err := svc.Register(ctx, RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
		Locale:   "en-US",
		Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if result.User.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", result.User.Email)
	}
	if result.Tokens.AccessToken == "" {
		t.Error("expected access token")
	}

	// Duplicate email
	_, err = svc.Register(ctx, RegisterInput{
		Email:    "test@example.com",
		Password: "other-password",
	})
	if err != ErrEmailTaken {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}

	// Login success
	loginResult, err := svc.Login(ctx, LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginResult.User.ID != result.User.ID {
		t.Errorf("expected same user ID")
	}

	// Login wrong password
	_, err = svc.Login(ctx, LoginInput{
		Email:    "test@example.com",
		Password: "wrong-password",
	})
	if err != ErrInvalidCreds {
		t.Errorf("expected ErrInvalidCreds, got %v", err)
	}

	// Login non-existent user
	_, err = svc.Login(ctx, LoginInput{
		Email:    "none@example.com",
		Password: "password123",
	})
	if err != ErrInvalidCreds {
		t.Errorf("expected ErrInvalidCreds, got %v", err)
	}
}

func TestRefreshToken(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewRepository(database)
	jwt := NewJWTManager(testConfig())
	svc := NewService(repo, jwt)

	ctx := context.Background()

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "refresh@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Refresh with valid token
	newTokens, err := svc.RefreshToken(ctx, result.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("expected new access token")
	}

	// Refresh with invalid token
	_, err = svc.RefreshToken(ctx, "invalid-token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestGetMe(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewRepository(database)
	jwt := NewJWTManager(testConfig())
	svc := NewService(repo, jwt)

	ctx := context.Background()

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "me@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	user, profile, err := svc.GetMe(ctx, result.User.ID)
	if err != nil {
		t.Fatalf("getMe: %v", err)
	}
	if user.Email != "me@example.com" {
		t.Errorf("expected email me@example.com, got %s", user.Email)
	}
	if profile.CurrentLevel != "A2" {
		t.Errorf("expected default level A2, got %s", profile.CurrentLevel)
	}
	if profile.DailyMinutes != 20 {
		t.Errorf("expected daily_minutes 20, got %d", profile.DailyMinutes)
	}
}

func TestUpdateProfile(t *testing.T) {
	database := setupTestDB(t)
	defer database.Close()

	repo := NewRepository(database)
	jwt := NewJWTManager(testConfig())
	svc := NewService(repo, jwt)

	ctx := context.Background()

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "update@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	level := "B1"
	domain := "business"
	mins := 30

	_, profile, err := svc.UpdateProfile(ctx, result.User.ID, &level, &domain, &mins, nil)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if profile.CurrentLevel != "B1" {
		t.Errorf("expected level B1, got %s", profile.CurrentLevel)
	}
	if profile.TargetDomain != "business" {
		t.Errorf("expected domain business, got %s", profile.TargetDomain)
	}
	if profile.DailyMinutes != 30 {
		t.Errorf("expected daily_minutes 30, got %d", profile.DailyMinutes)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
