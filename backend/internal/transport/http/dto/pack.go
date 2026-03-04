package dto

type PackDTO struct {
	ID               string `json:"id"`
	Source           string `json:"source"`
	Title            string `json:"title"`
	Description      string `json:"description,omitempty"`
	Domain           string `json:"domain"`
	Level            string `json:"level"`
	EstimatedMinutes int    `json:"estimated_minutes"`
}

type LessonDTO struct {
	LessonID   string `json:"lesson_id"`
	Title      string `json:"title"`
	LessonType string `json:"lesson_type"`
	Position   int    `json:"position"`
}

type PackListResponse struct {
	Items    []PackDTO `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

type PackDetailResponse struct {
	Pack    PackDTO     `json:"pack"`
	Lessons []LessonDTO `json:"lessons"`
}

type GeneratePackRequest struct {
	Level        string   `json:"level" validate:"required,oneof=A1 A2 B1 B2 C1 C2"`
	Domain       string   `json:"domain" validate:"required"`
	DailyMinutes int      `json:"daily_minutes" validate:"required,min=5,max=180"`
	Days         *int     `json:"days" validate:"omitempty,min=3,max=14"`
	FocusSkills  []string `json:"focus_skills" validate:"dive,oneof=reading listening speaking writing"`
}

type GenerationJobResponse struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
	PackID       string `json:"pack_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type GenericMessage struct {
	Message string `json:"message"`
}
