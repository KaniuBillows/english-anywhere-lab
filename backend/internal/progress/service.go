package progress

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type SummaryResult struct {
	Range          string
	TotalMinutes   int
	ActiveDays     int
	ReviewAccuracy float64
	CardsReviewed  int
	StreakCount     int
}

func (s *Service) GetSummary(ctx context.Context, userID, rangeStr string) (*SummaryResult, error) {
	var days int
	switch rangeStr {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		return nil, fmt.Errorf("invalid range: %s", rangeStr)
	}

	fromDate := time.Now().UTC().AddDate(0, 0, -days+1)
	summary, err := s.repo.GetSummary(ctx, userID, fromDate)
	if err != nil {
		return nil, err
	}

	return &SummaryResult{
		Range:          rangeStr,
		TotalMinutes:   summary.TotalMinutes,
		ActiveDays:     summary.ActiveDays,
		ReviewAccuracy: summary.ReviewAccuracy,
		CardsReviewed:  summary.CardsReviewed,
		StreakCount:     summary.StreakCount,
	}, nil
}

type DailyResult struct {
	Points []DailyPointResult
}

type DailyPointResult struct {
	Date           string
	MinutesLearned int
	CardsReviewed  int
	ReviewAccuracy *float64
}

type ReviewRatingCountsResult struct {
	Again    int
	Hard     int
	Good     int
	Easy     int
	Total    int
	Accuracy *float64
}

type WeeklyComparisonResult struct {
	MinutesDelta        int
	ActiveDaysDelta     int
	CardsReviewedDelta  int
	LessonsDelta        int
	AccuracyDelta       *float64
}

type DailyPointFullResult struct {
	Date              string
	MinutesLearned    int
	LessonsCompleted  int
	CardsNew          int
	CardsReviewed     int
	ReviewAccuracy    *float64
	ListeningMinutes  int
	SpeakingTasks     int
	WritingTasks      int
	StreakCount        int
}

type WeeklyReportResult struct {
	WeekStart          string
	TotalMinutes       int
	ActiveDays         int
	CardsReviewed      int
	CardsNew           int
	LessonsCompleted   int
	ListeningMinutes   int
	SpeakingTasks      int
	WritingTasks       int
	Streak             int
	WeeklyGoalDays     int
	GoalAchieved       bool
	ReviewHealth       *ReviewRatingCountsResult
	DailyBreakdown     []DailyPointFullResult
	PrevWeek           *WeeklyComparisonResult
}

func (s *Service) GetWeeklyReport(ctx context.Context, userID, weekStartStr string) (*WeeklyReportResult, error) {
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		return nil, fmt.Errorf("invalid week_start date format: %s", weekStartStr)
	}
	if weekStart.Weekday() != time.Monday {
		return nil, fmt.Errorf("week_start must be a Monday, got %s", weekStart.Weekday())
	}

	weekEnd := weekStart.AddDate(0, 0, 6)
	prevWeekStart := weekStart.AddDate(0, 0, -7)
	prevWeekEnd := weekStart.AddDate(0, 0, -1)

	wsStr := weekStart.Format("2006-01-02")
	weStr := weekEnd.Format("2006-01-02")
	pwsStr := prevWeekStart.Format("2006-01-02")
	pweStr := prevWeekEnd.Format("2006-01-02")

	currAgg, err := s.repo.GetWeeklyAggregate(ctx, userID, wsStr, weStr)
	if err != nil {
		return nil, err
	}

	prevAgg, err := s.repo.GetWeeklyAggregate(ctx, userID, pwsStr, pweStr)
	if err != nil {
		return nil, err
	}

	ratings, err := s.repo.GetReviewRatingCounts(ctx, userID, wsStr, weStr)
	if err != nil {
		return nil, err
	}

	dailyRows, err := s.repo.GetDailyFull(ctx, userID, wsStr, weStr)
	if err != nil {
		return nil, err
	}

	goalDays, err := s.repo.GetWeeklyGoalDays(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build review health
	total := ratings.Again + ratings.Hard + ratings.Good + ratings.Easy
	var accuracy *float64
	if total > 0 {
		v := float64(ratings.Hard+ratings.Good+ratings.Easy) / float64(total)
		accuracy = &v
	}
	reviewHealth := &ReviewRatingCountsResult{
		Again:    ratings.Again,
		Hard:     ratings.Hard,
		Good:     ratings.Good,
		Easy:     ratings.Easy,
		Total:    total,
		Accuracy: accuracy,
	}

	// Build daily breakdown
	daily := make([]DailyPointFullResult, 0, len(dailyRows))
	for _, d := range dailyRows {
		dp := DailyPointFullResult{
			Date:             d.Date,
			MinutesLearned:   d.MinutesLearned,
			LessonsCompleted: d.LessonsCompleted,
			CardsNew:         d.CardsNew,
			CardsReviewed:    d.CardsReviewed,
			ListeningMinutes: d.ListeningMinutes,
			SpeakingTasks:    d.SpeakingTasks,
			WritingTasks:     d.WritingTasks,
			StreakCount:       d.StreakCount,
		}
		if d.ReviewAccuracy.Valid {
			dp.ReviewAccuracy = &d.ReviewAccuracy.Float64
		}
		daily = append(daily, dp)
	}

	result := &WeeklyReportResult{
		WeekStart:        wsStr,
		TotalMinutes:     currAgg.TotalMinutes,
		ActiveDays:       currAgg.ActiveDays,
		CardsReviewed:    currAgg.CardsReviewed,
		CardsNew:         currAgg.CardsNew,
		LessonsCompleted: currAgg.LessonsCompleted,
		ListeningMinutes: currAgg.ListeningMinutes,
		SpeakingTasks:    currAgg.SpeakingTasks,
		WritingTasks:     currAgg.WritingTasks,
		Streak:           currAgg.MaxStreak,
		WeeklyGoalDays:   goalDays,
		GoalAchieved:     currAgg.ActiveDays >= goalDays,
		ReviewHealth:     reviewHealth,
		DailyBreakdown:   daily,
	}

	// Previous week comparison — omit if no data
	if prevAgg.ActiveDays > 0 || prevAgg.TotalMinutes > 0 {
		comp := &WeeklyComparisonResult{
			MinutesDelta:       currAgg.TotalMinutes - prevAgg.TotalMinutes,
			ActiveDaysDelta:    currAgg.ActiveDays - prevAgg.ActiveDays,
			CardsReviewedDelta: currAgg.CardsReviewed - prevAgg.CardsReviewed,
			LessonsDelta:       currAgg.LessonsCompleted - prevAgg.LessonsCompleted,
		}
		// Accuracy delta: only if both weeks have accuracy
		if currAgg.AvgReviewAccuracy.Valid && prevAgg.AvgReviewAccuracy.Valid {
			d := currAgg.AvgReviewAccuracy.Float64 - prevAgg.AvgReviewAccuracy.Float64
			comp.AccuracyDelta = &d
		}
		result.PrevWeek = comp
	}

	return result, nil
}

func (s *Service) GetDaily(ctx context.Context, userID, from, to string) (*DailyResult, error) {
	points, err := s.repo.GetDaily(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	result := &DailyResult{Points: make([]DailyPointResult, 0, len(points))}
	for _, p := range points {
		dp := DailyPointResult{
			Date:           p.Date,
			MinutesLearned: p.MinutesLearned,
			CardsReviewed:  p.CardsReviewed,
		}
		if p.ReviewAccuracy.Valid {
			dp.ReviewAccuracy = &p.ReviewAccuracy.Float64
		}
		result.Points = append(result.Points, dp)
	}
	return result, nil
}
