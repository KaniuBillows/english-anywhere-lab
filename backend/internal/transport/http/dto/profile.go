package dto

type LearningProfileDTO struct {
	CurrentLevel   string `json:"current_level"`
	TargetDomain   string `json:"target_domain"`
	DailyMinutes   int    `json:"daily_minutes"`
	WeeklyGoalDays int    `json:"weekly_goal_days"`
}

type MeResponse struct {
	User            UserDTO            `json:"user"`
	LearningProfile LearningProfileDTO `json:"learning_profile"`
}

type UpdateProfileRequest struct {
	CurrentLevel   *string `json:"current_level" validate:"omitempty,oneof=A1 A2 B1 B2 C1 C2"`
	TargetDomain   *string `json:"target_domain"`
	DailyMinutes   *int    `json:"daily_minutes" validate:"omitempty,min=5,max=180"`
	WeeklyGoalDays *int    `json:"weekly_goal_days" validate:"omitempty,min=1,max=7"`
}
