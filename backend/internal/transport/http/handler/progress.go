package handler

import (
	"errors"
	"net/http"

	"github.com/bennyshi/english-anywhere-lab/internal/progress"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

type ProgressHandler struct {
	svc *progress.Service
}

func NewProgressHandler(svc *progress.Service) *ProgressHandler {
	return &ProgressHandler{svc: svc}
}

func (h *ProgressHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	rangeStr := r.URL.Query().Get("range")
	if rangeStr == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "range parameter is required (7d, 30d, 90d)")
		return
	}

	result, err := h.svc.GetSummary(r.Context(), userID, rangeStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ProgressSummaryResponse{
		Range:          result.Range,
		TotalMinutes:   result.TotalMinutes,
		ActiveDays:     result.ActiveDays,
		ReviewAccuracy: result.ReviewAccuracy,
		CardsReviewed:  result.CardsReviewed,
		StreakCount:     result.StreakCount,
	})
}

func (h *ProgressHandler) GetDaily(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "from and to date parameters are required")
		return
	}

	result, err := h.svc.GetDaily(r.Context(), userID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get daily progress")
		return
	}

	points := make([]dto.ProgressDailyPoint, 0, len(result.Points))
	for _, p := range result.Points {
		points = append(points, dto.ProgressDailyPoint{
			Date:           p.Date,
			MinutesLearned: p.MinutesLearned,
			CardsReviewed:  p.CardsReviewed,
			ReviewAccuracy: p.ReviewAccuracy,
		})
	}

	writeJSON(w, http.StatusOK, dto.ProgressDailyResponse{Points: points})
}

func (h *ProgressHandler) GetWeeklyReport(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	weekStart := r.URL.Query().Get("week_start")
	if weekStart == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "week_start parameter is required (YYYY-MM-DD, must be a Monday)")
		return
	}

	result, err := h.svc.GetWeeklyReport(r.Context(), userID, weekStart)
	if err != nil {
		var ve *progress.ValidationError
		if errors.As(err, &ve) {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", ve.Message)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get weekly report")
		return
	}

	daily := make([]dto.WeeklyReportDailyPoint, 0, len(result.DailyBreakdown))
	for _, d := range result.DailyBreakdown {
		daily = append(daily, dto.WeeklyReportDailyPoint{
			Date:             d.Date,
			MinutesLearned:   d.MinutesLearned,
			LessonsCompleted: d.LessonsCompleted,
			CardsNew:         d.CardsNew,
			CardsReviewed:    d.CardsReviewed,
			ReviewAccuracy:   d.ReviewAccuracy,
			ListeningMinutes: d.ListeningMinutes,
			SpeakingTasks:    d.SpeakingTasks,
			WritingTasks:     d.WritingTasks,
			StreakCount:       d.StreakCount,
		})
	}

	resp := dto.WeeklyReportResponse{
		WeekStart:        result.WeekStart,
		TotalMinutes:     result.TotalMinutes,
		ActiveDays:       result.ActiveDays,
		CardsReviewed:    result.CardsReviewed,
		CardsNew:         result.CardsNew,
		LessonsCompleted: result.LessonsCompleted,
		ListeningMinutes: result.ListeningMinutes,
		SpeakingTasks:    result.SpeakingTasks,
		WritingTasks:     result.WritingTasks,
		Streak:           result.Streak,
		WeeklyGoalDays:   result.WeeklyGoalDays,
		GoalAchieved:     result.GoalAchieved,
		ReviewHealth: dto.ReviewHealth{
			Again:    result.ReviewHealth.Again,
			Hard:     result.ReviewHealth.Hard,
			Good:     result.ReviewHealth.Good,
			Easy:     result.ReviewHealth.Easy,
			Total:    result.ReviewHealth.Total,
			Accuracy: result.ReviewHealth.Accuracy,
		},
		DailyBreakdown: daily,
	}

	if result.PrevWeek != nil {
		resp.PreviousWeekComparison = &dto.WeeklyComparison{
			MinutesDelta:       result.PrevWeek.MinutesDelta,
			ActiveDaysDelta:    result.PrevWeek.ActiveDaysDelta,
			CardsReviewedDelta: result.PrevWeek.CardsReviewedDelta,
			LessonsDelta:       result.PrevWeek.LessonsDelta,
			AccuracyDelta:      result.PrevWeek.AccuracyDelta,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
