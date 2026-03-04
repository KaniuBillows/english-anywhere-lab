package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type GeneratedPack struct {
	Title            string            `json:"title" validate:"required"`
	Description      string            `json:"description"`
	EstimatedMinutes int               `json:"estimated_minutes" validate:"required,min=1"`
	Lessons          []GeneratedLesson `json:"lessons" validate:"required,min=1,dive"`
}

type GeneratedLesson struct {
	Title            string                `json:"title" validate:"required"`
	LessonType       string                `json:"lesson_type" validate:"required,oneof=reading listening speaking writing mixed"`
	Position         int                   `json:"position" validate:"required,min=1"`
	EstimatedMinutes int                   `json:"estimated_minutes" validate:"required,min=1"`
	Cards            []GeneratedCard       `json:"cards" validate:"required,min=1,dive"`
	OutputTasks      []GeneratedOutputTask `json:"output_tasks,omitempty" validate:"dive"`
}

type GeneratedCard struct {
	FrontText   string `json:"front_text" validate:"required"`
	BackText    string `json:"back_text" validate:"required"`
	ExampleText string `json:"example_text,omitempty"`
}

type GeneratedOutputTask struct {
	TaskType        string `json:"task_type" validate:"required,oneof=reading listening speaking writing mixed"`
	PromptText      string `json:"prompt_text" validate:"required"`
	ReferenceAnswer string `json:"reference_answer,omitempty"`
}

// ParseAndValidate unmarshals JSON into a GeneratedPack and validates it.
func ParseAndValidate(raw string) (*GeneratedPack, error) {
	// Strip markdown code fences if present
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		// Remove opening fence (e.g. ```json)
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(cleaned, "```"); idx != -1 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)
	}

	var pack GeneratedPack
	if err := json.Unmarshal([]byte(cleaned), &pack); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if err := validate.Struct(pack); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &pack, nil
}
