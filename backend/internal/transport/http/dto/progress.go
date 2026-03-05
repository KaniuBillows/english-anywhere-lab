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

type WeeklyReportDailyPoint struct {
	Date             string   `json:"date"`
	MinutesLearned   int      `json:"minutes_learned"`
	LessonsCompleted int      `json:"lessons_completed"`
	CardsNew         int      `json:"cards_new"`
	CardsReviewed    int      `json:"cards_reviewed"`
	ReviewAccuracy   *float64 `json:"review_accuracy,omitempty"`
	ListeningMinutes int      `json:"listening_minutes"`
	SpeakingTasks    int      `json:"speaking_tasks"`
	WritingTasks     int      `json:"writing_tasks"`
	StreakCount       int      `json:"streak_count"`
}

type ReviewHealth struct {
	Again    int      `json:"again"`
	Hard     int      `json:"hard"`
	Good     int      `json:"good"`
	Easy     int      `json:"easy"`
	Total    int      `json:"total"`
	Accuracy *float64 `json:"accuracy,omitempty"`
}

type WeeklyComparison struct {
	MinutesDelta       int      `json:"minutes_delta"`
	ActiveDaysDelta    int      `json:"active_days_delta"`
	CardsReviewedDelta int      `json:"cards_reviewed_delta"`
	LessonsDelta       int      `json:"lessons_delta"`
	AccuracyDelta      *float64 `json:"accuracy_delta,omitempty"`
}

type WeeklyReportResponse struct {
	WeekStart              string                   `json:"week_start"`
	TotalMinutes           int                      `json:"total_minutes"`
	ActiveDays             int                      `json:"active_days"`
	CardsReviewed          int                      `json:"cards_reviewed"`
	CardsNew               int                      `json:"cards_new"`
	LessonsCompleted       int                      `json:"lessons_completed"`
	ListeningMinutes       int                      `json:"listening_minutes"`
	SpeakingTasks          int                      `json:"speaking_tasks"`
	WritingTasks           int                      `json:"writing_tasks"`
	Streak                 int                      `json:"streak"`
	WeeklyGoalDays         int                      `json:"weekly_goal_days"`
	GoalAchieved           bool                     `json:"goal_achieved"`
	ReviewHealth           ReviewHealth              `json:"review_health"`
	DailyBreakdown         []WeeklyReportDailyPoint `json:"daily_breakdown"`
	PreviousWeekComparison *WeeklyComparison        `json:"previous_week_comparison,omitempty"`
}
