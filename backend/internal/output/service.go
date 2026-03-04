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
	ErrInvalidFeedback    = errors.New("LLM returned invalid feedback")
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

// Submit handles a writing task submission: validates task, calls LLM, validates feedback,
// then atomically inserts the submission and increments progress.
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

	// Call LLM for feedback BEFORE any DB writes
	messages := BuildWritingFeedbackPrompt(task, input.AnswerText)
	rawFeedback, err := s.llmCaller.ChatCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm call: %w", err)
	}

	// Parse and validate feedback
	var feedback WritingFeedback
	if err := json.Unmarshal([]byte(rawFeedback), &feedback); err != nil {
		return nil, fmt.Errorf("parse feedback: %w", err)
	}
	if err := validateFeedback(&feedback); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFeedback, err)
	}

	score := float64(feedback.OverallScore)

	// Atomically insert submission + increment progress (only after LLM succeeds)
	now := time.Now().UTC().Format(time.RFC3339)
	sub := &OutputSubmission{
		ID:          uuid.New().String(),
		UserID:      input.UserID,
		TaskID:      input.TaskID,
		AnswerText:  sql.NullString{String: input.AnswerText, Valid: true},
		AIFeedback:  sql.NullString{String: rawFeedback, Valid: true},
		Score:       sql.NullFloat64{Float64: score, Valid: true},
		SubmittedAt: now,
	}
	if err := s.repo.InsertSubmissionWithProgress(ctx, sub); err != nil {
		return nil, fmt.Errorf("insert submission: %w", err)
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

// validateFeedback checks that LLM feedback meets required structure and ranges.
func validateFeedback(f *WritingFeedback) error {
	if f.OverallScore < 0 || f.OverallScore > 100 {
		return fmt.Errorf("overall_score %d out of range [0,100]", f.OverallScore)
	}
	if f.RevisedText == "" {
		return fmt.Errorf("revised_text is empty")
	}
	if len(f.NextActions) == 0 {
		return fmt.Errorf("next_actions is empty")
	}
	if f.Errors == nil {
		f.Errors = []WritingError{}
	}
	return nil
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
