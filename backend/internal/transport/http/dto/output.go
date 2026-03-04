package dto

// OutputTaskDTO represents an output task in API responses.
type OutputTaskDTO struct {
	ID              string `json:"id"`
	LessonID        string `json:"lesson_id,omitempty"`
	TaskType        string `json:"task_type"`
	PromptText      string `json:"prompt_text"`
	ReferenceAnswer string `json:"reference_answer,omitempty"`
	Level           string `json:"level,omitempty"`
}

// OutputTaskListResponse is the response for listing output tasks.
type OutputTaskListResponse struct {
	Items []OutputTaskDTO `json:"items"`
}

// SubmitWritingRequest is the request body for submitting a writing task.
type SubmitWritingRequest struct {
	AnswerText string `json:"answer_text" validate:"required,min=1,max=5000"`
}

// WritingErrorDTO represents a single error in the learner's text.
type WritingErrorDTO struct {
	Original    string `json:"original"`
	Correction  string `json:"correction"`
	Explanation string `json:"explanation"`
}

// WritingFeedbackDTO represents the structured AI feedback.
type WritingFeedbackDTO struct {
	OverallScore int               `json:"overall_score"`
	Errors       []WritingErrorDTO `json:"errors"`
	RevisedText  string            `json:"revised_text"`
	NextActions  []string          `json:"next_actions"`
}

// SubmissionResponse is the response for a writing submission.
type SubmissionResponse struct {
	SubmissionID string              `json:"submission_id"`
	TaskID       string              `json:"task_id"`
	AnswerText   string              `json:"answer_text"`
	Feedback     *WritingFeedbackDTO `json:"feedback,omitempty"`
	Score        float64             `json:"score"`
	SubmittedAt  string              `json:"submitted_at"`
}
