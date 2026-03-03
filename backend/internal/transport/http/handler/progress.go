package handler

import (
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
