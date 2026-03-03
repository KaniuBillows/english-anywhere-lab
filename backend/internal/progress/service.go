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
