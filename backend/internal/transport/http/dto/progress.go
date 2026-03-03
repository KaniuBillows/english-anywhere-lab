package dto

type ProgressSummaryResponse struct {
	Range          string  `json:"range"`
	TotalMinutes   int     `json:"total_minutes"`
	ActiveDays     int     `json:"active_days"`
	ReviewAccuracy float64 `json:"review_accuracy"`
	CardsReviewed  int     `json:"cards_reviewed"`
	StreakCount     int     `json:"streak_count"`
}

type ProgressDailyPoint struct {
	Date           string   `json:"date"`
	MinutesLearned int      `json:"minutes_learned"`
	CardsReviewed  int      `json:"cards_reviewed"`
	ReviewAccuracy *float64 `json:"review_accuracy,omitempty"`
}

type ProgressDailyResponse struct {
	Points []ProgressDailyPoint `json:"points"`
}
