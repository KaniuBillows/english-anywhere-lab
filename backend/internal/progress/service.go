package progress

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"time"
)

// ValidationError represents a client input validation error.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

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
		return nil, &ValidationError{Message: fmt.Sprintf("invalid week_start date format: %s", weekStartStr)}
	}
	if weekStart.Weekday() != time.Monday {
		return nil, &ValidationError{Message: fmt.Sprintf("week_start must be a Monday, got %s", weekStart.Weekday())}
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

	// Previous week comparison — omit only if no rows exist
	if prevAgg.RowCount > 0 {
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

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

type SkillMetric struct {
	Skill      string
	Value      int
	Percentage float64
}

type SkillBreakdownResult struct {
	Listening  SkillMetric
	Speaking   SkillMetric
	Writing    SkillMetric
	Reading    SkillMetric
}

type WeaknessItem struct {
	Skill     string
	Reason    string
	Value     int
	PrevValue *int
}

type MonthlyComparisonResult struct {
	MinutesDelta       int
	ActiveDaysDelta    int
	CardsReviewedDelta int
	LessonsDelta       int
	AccuracyDelta      *float64
}

type MonthlyReportResult struct {
	Month            string
	DaysInMonth      int
	TotalMinutes     int
	ActiveDays       int
	CardsReviewed    int
	CardsNew         int
	LessonsCompleted int
	ListeningMinutes int
	SpeakingTasks    int
	WritingTasks     int
	Streak           int
	MonthlyGoalDays  int
	GoalAchieved     bool
	ReviewHealth     *ReviewRatingCountsResult
	DailyBreakdown   []DailyPointFullResult
	SkillBreakdown   SkillBreakdownResult
	Weaknesses       []WeaknessItem
	PrevMonth        *MonthlyComparisonResult
}

func (s *Service) GetMonthlyReport(ctx context.Context, userID, monthStr string) (*MonthlyReportResult, error) {
	if !monthRe.MatchString(monthStr) {
		return nil, &ValidationError{Message: fmt.Sprintf("invalid month format: %s (expected YYYY-MM)", monthStr)}
	}

	// Parse month boundaries
	t, _ := time.Parse("2006-01", monthStr)
	year, month := t.Year(), t.Month()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC) // day 0 of next month = last day of this month
	daysInMonth := lastDay.Day()

	prevFirstDay := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
	prevLastDay := time.Date(year, month, 0, 0, 0, 0, 0, time.UTC)

	fdStr := firstDay.Format("2006-01-02")
	ldStr := lastDay.Format("2006-01-02")
	pfdStr := prevFirstDay.Format("2006-01-02")
	pldStr := prevLastDay.Format("2006-01-02")

	currAgg, err := s.repo.GetWeeklyAggregate(ctx, userID, fdStr, ldStr)
	if err != nil {
		return nil, err
	}

	prevAgg, err := s.repo.GetWeeklyAggregate(ctx, userID, pfdStr, pldStr)
	if err != nil {
		return nil, err
	}

	ratings, err := s.repo.GetReviewRatingCounts(ctx, userID, fdStr, ldStr)
	if err != nil {
		return nil, err
	}

	dailyRows, err := s.repo.GetDailyFull(ctx, userID, fdStr, ldStr)
	if err != nil {
		return nil, err
	}

	goalDays, err := s.repo.GetWeeklyGoalDays(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Monthly goal = weekly_goal_days × ceil(days_in_month / 7)
	weeksInMonth := int(math.Ceil(float64(daysInMonth) / 7.0))
	monthlyGoal := goalDays * weeksInMonth

	// Review health
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

	// Daily breakdown
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
			StreakCount:      d.StreakCount,
		}
		if d.ReviewAccuracy.Valid {
			dp.ReviewAccuracy = &d.ReviewAccuracy.Float64
		}
		daily = append(daily, dp)
	}

	// Skill breakdown: use aggregate fields
	listening := currAgg.ListeningMinutes
	speaking := currAgg.SpeakingTasks
	writing := currAgg.WritingTasks
	reading := currAgg.CardsReviewed + currAgg.LessonsCompleted

	skillTotal := listening + speaking + writing + reading
	pct := func(v int) float64 {
		if skillTotal == 0 {
			return 0
		}
		return float64(v) / float64(skillTotal)
	}

	skillBreakdown := SkillBreakdownResult{
		Listening: SkillMetric{Skill: "listening", Value: listening, Percentage: pct(listening)},
		Speaking:  SkillMetric{Skill: "speaking", Value: speaking, Percentage: pct(speaking)},
		Writing:   SkillMetric{Skill: "writing", Value: writing, Percentage: pct(writing)},
		Reading:   SkillMetric{Skill: "reading", Value: reading, Percentage: pct(reading)},
	}

	// Weakness detection
	weaknesses := detectWeaknesses(currAgg, prevAgg)

	result := &MonthlyReportResult{
		Month:            monthStr,
		DaysInMonth:      daysInMonth,
		TotalMinutes:     currAgg.TotalMinutes,
		ActiveDays:       currAgg.ActiveDays,
		CardsReviewed:    currAgg.CardsReviewed,
		CardsNew:         currAgg.CardsNew,
		LessonsCompleted: currAgg.LessonsCompleted,
		ListeningMinutes: currAgg.ListeningMinutes,
		SpeakingTasks:    currAgg.SpeakingTasks,
		WritingTasks:     currAgg.WritingTasks,
		Streak:           currAgg.MaxStreak,
		MonthlyGoalDays:  monthlyGoal,
		GoalAchieved:     currAgg.ActiveDays >= monthlyGoal,
		ReviewHealth:     reviewHealth,
		DailyBreakdown:   daily,
		SkillBreakdown:   skillBreakdown,
		Weaknesses:       weaknesses,
	}

	// Previous month comparison
	if prevAgg.RowCount > 0 {
		comp := &MonthlyComparisonResult{
			MinutesDelta:       currAgg.TotalMinutes - prevAgg.TotalMinutes,
			ActiveDaysDelta:    currAgg.ActiveDays - prevAgg.ActiveDays,
			CardsReviewedDelta: currAgg.CardsReviewed - prevAgg.CardsReviewed,
			LessonsDelta:       currAgg.LessonsCompleted - prevAgg.LessonsCompleted,
		}
		if currAgg.AvgReviewAccuracy.Valid && prevAgg.AvgReviewAccuracy.Valid {
			d := currAgg.AvgReviewAccuracy.Float64 - prevAgg.AvgReviewAccuracy.Float64
			comp.AccuracyDelta = &d
		}
		result.PrevMonth = comp
	}

	return result, nil
}

// detectWeaknesses identifies per-skill weaknesses.
func detectWeaknesses(curr, prev *WeeklyAggregate) []WeaknessItem {
	type skillPair struct {
		name     string
		currVal  int
		prevVal  int
	}

	listening := curr.ListeningMinutes
	speaking := curr.SpeakingTasks
	writing := curr.WritingTasks
	reading := curr.CardsReviewed + curr.LessonsCompleted

	skills := []skillPair{
		{"listening", listening, prev.ListeningMinutes},
		{"speaking", speaking, prev.SpeakingTasks},
		{"writing", writing, prev.WritingTasks},
		{"reading", reading, prev.CardsReviewed + prev.LessonsCompleted},
	}

	// Check if at least one skill has non-zero activity
	hasActivity := false
	for _, sk := range skills {
		if sk.currVal > 0 {
			hasActivity = true
			break
		}
	}

	var weaknesses []WeaknessItem

	for _, sk := range skills {
		// "low_activity": skill with 0 activity when at least one other skill has data
		if sk.currVal == 0 && hasActivity {
			weaknesses = append(weaknesses, WeaknessItem{
				Skill:  sk.name,
				Reason: "low_activity",
				Value:  sk.currVal,
			})
			continue
		}

		// "declining": metric decreased vs previous month (only when prev month has data)
		if prev.RowCount > 0 && sk.prevVal > 0 && sk.currVal < sk.prevVal {
			pv := sk.prevVal
			weaknesses = append(weaknesses, WeaknessItem{
				Skill:     sk.name,
				Reason:    "declining",
				Value:     sk.currVal,
				PrevValue: &pv,
			})
		}
	}

	return weaknesses
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
