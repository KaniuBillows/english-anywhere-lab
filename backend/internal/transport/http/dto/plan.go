package dto

type BootstrapPlanRequest struct {
	Level        string `json:"level" validate:"required,oneof=A1 A2 B1 B2 C1 C2"`
	TargetDomain string `json:"target_domain" validate:"required"`
	DailyMinutes int    `json:"daily_minutes" validate:"required,min=5,max=180"`
	Days         int    `json:"days" validate:"omitempty,min=3,max=14"`
}

type PlanTaskDTO struct {
	TaskID           string `json:"task_id"`
	TaskType         string `json:"task_type"`
	Title            string `json:"title"`
	Status           string `json:"status"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Virtual          bool   `json:"virtual,omitempty"`
}

type DailyPlanDTO struct {
	PlanID              string        `json:"plan_id"`
	Date                string        `json:"date"`
	Mode                string        `json:"mode"`
	TotalEstimatedMins  int           `json:"total_estimated_minutes"`
	Tasks               []PlanTaskDTO `json:"tasks"`
}

type DailyPlanResponse struct {
	DailyPlan DailyPlanDTO `json:"daily_plan"`
}

type WeeklyPlanResponse struct {
	WeekStart  string         `json:"week_start"`
	DailyPlans []DailyPlanDTO `json:"daily_plans"`
}

type CompleteTaskRequest struct {
	CompletedAt     string `json:"completed_at" validate:"required"`
	DurationSeconds *int   `json:"duration_seconds"`
}

type TaskCompletionResponse struct {
	TaskID             string  `json:"task_id"`
	Status             string  `json:"status"`
	NextRecommendation *string `json:"next_recommendation,omitempty"`
}
