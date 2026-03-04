package output

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/llm"
	"github.com/google/uuid"
)

var (
	ErrTaskNotFound       = errors.New("output task not found")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrNotWritingTask     = errors.New("task is not a writing task")
)

// LLMCaller abstracts the LLM chat completion call for testability.
type LLMCaller interface {
	ChatCompletion(ctx context.Context, messages []llm.Message) (string, error)
}

// WritingError represents a single error found in the learner's text.
type WritingError struct {
	Original    string `json:"original"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
}

// WritingFeedback represents the structured AI feedback for a writing submission.
type WritingFeedback struct {
	OverallScore int            `json:"overall_score"`
	Errors       []WritingError `json:"errors"`
	RevisedText  string         `json:"revised_text"`
	NextActions  []string       `json:"next_actions"`
}

// SubmitInput holds the input for a writing submission.
type SubmitInput struct {
	UserID     string
	TaskID     string
	AnswerText string
}

// SubmitResult holds the result of a writing submission including feedback.
type SubmitResult struct {
	SubmissionID string
	TaskID       string
	AnswerText   string
	Feedback     *WritingFeedback
	Score        float64
	SubmittedAt  string
}

// Service provides business logic for output tasks.
type Service struct {
	repo      *Repository
	llmCaller LLMCaller
}

// NewService creates a new output Service.
func NewService(repo *Repository, llmCaller LLMCaller) *Service {
	return &Service{repo: repo, llmCaller: llmCaller}
}

// ListTasks returns writing tasks for a lesson.
func (s *Service) ListTasks(ctx context.Context, lessonID string) ([]OutputTask, error) {
	tasks, err := s.repo.ListByLesson(ctx, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// Submit handles a writing task submission: validates, inserts, calls LLM, updates, and returns.
func (s *Service) Submit(ctx context.Context, input SubmitInput) (*SubmitResult, error) {
	// Get and validate the task
	task, err := s.repo.GetTask(ctx, input.TaskID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	if task.TaskType != "writing" {
		return nil, ErrNotWritingTask
	}

	// Insert submission (without feedback initially)
	now := time.Now().UTC().Format(time.RFC3339)
	sub := &OutputSubmission{
		ID:          uuid.New().String(),
		UserID:      input.UserID,
		TaskID:      input.TaskID,
		AnswerText:  sql.NullString{String: input.AnswerText, Valid: true},
		SubmittedAt: now,
	}
	if err := s.repo.InsertSubmission(ctx, sub); err != nil {
		return nil, fmt.Errorf("insert submission: %w", err)
	}

	// Call LLM for feedback
	messages := BuildWritingFeedbackPrompt(task, input.AnswerText)
	rawFeedback, err := s.llmCaller.ChatCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm call: %w", err)
	}

	// Parse feedback
	var feedback WritingFeedback
	if err := json.Unmarshal([]byte(rawFeedback), &feedback); err != nil {
		return nil, fmt.Errorf("parse feedback: %w", err)
	}

	score := float64(feedback.OverallScore)

	// Update submission with feedback
	if err := s.repo.UpdateSubmissionFeedback(ctx, sub.ID, rawFeedback, score); err != nil {
		return nil, fmt.Errorf("update feedback: %w", err)
	}

	// Increment progress
	if err := s.repo.IncrementWritingTasksCompleted(ctx, input.UserID); err != nil {
		return nil, fmt.Errorf("increment progress: %w", err)
	}

	return &SubmitResult{
		SubmissionID: sub.ID,
		TaskID:       input.TaskID,
		AnswerText:   input.AnswerText,
		Feedback:     &feedback,
		Score:        score,
		SubmittedAt:  now,
	}, nil
}

// GetSubmission retrieves a submission result by ID, scoped to the user.
func (s *Service) GetSubmission(ctx context.Context, submissionID, userID string) (*SubmitResult, error) {
	sub, err := s.repo.GetSubmission(ctx, submissionID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get submission: %w", err)
	}

	result := &SubmitResult{
		SubmissionID: sub.ID,
		TaskID:       sub.TaskID,
		SubmittedAt:  sub.SubmittedAt,
	}
	if sub.AnswerText.Valid {
		result.AnswerText = sub.AnswerText.String
	}
	if sub.Score.Valid {
		result.Score = sub.Score.Float64
	}
	if sub.AIFeedback.Valid {
		var feedback WritingFeedback
		if err := json.Unmarshal([]byte(sub.AIFeedback.String), &feedback); err == nil {
			result.Feedback = &feedback
		}
	}

	return result, nil
}
