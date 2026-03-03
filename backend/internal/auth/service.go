package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken     = errors.New("email already registered")
	ErrInvalidCreds   = errors.New("invalid email or password")
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidToken   = errors.New("invalid or expired token")
)

type Service struct {
	repo *Repository
	jwt  *JWTManager
}

func NewService(repo *Repository, jwt *JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

type RegisterInput struct {
	Email    string
	Password string
	Locale   string
	Timezone string
}

type AuthResult struct {
	User   *User
	Tokens *TokenPair
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Check if email exists
	_, err := s.repo.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return nil, ErrEmailTaken
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	locale := input.Locale
	if locale == "" {
		locale = "zh-CN"
	}
	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	user := &User{
		ID:           uuid.New().String(),
		Email:        input.Email,
		PasswordHash: string(hash),
		Locale:       locale,
		Timezone:     tz,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	profile := &LearningProfile{
		UserID:         user.ID,
		CurrentLevel:   "A2",
		TargetDomain:   "general",
		DailyMinutes:   20,
		WeeklyGoalDays: 5,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.CreateLearningProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	tokens, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Tokens: tokens}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidCreds
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	tokens, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Tokens: tokens}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Verify user still exists and is active
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !user.IsActive {
		return nil, ErrInvalidToken
	}

	return s.jwt.GenerateTokenPair(userID)
}

func (s *Service) GetMe(ctx context.Context, userID string) (*User, *LearningProfile, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrUserNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	profile, err := s.repo.GetLearningProfile(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	return user, profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, level, domain *string, dailyMinutes, weeklyGoalDays *int) (*User, *LearningProfile, error) {
	if err := s.repo.UpdateLearningProfile(ctx, userID, level, domain, dailyMinutes, weeklyGoalDays); err != nil {
		return nil, nil, err
	}
	return s.GetMe(ctx, userID)
}
