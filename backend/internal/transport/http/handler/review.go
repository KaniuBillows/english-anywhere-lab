package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/review"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

type ReviewHandler struct {
	svc *review.Service
}

func NewReviewHandler(svc *review.Service) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) GetQueue(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	result, err := h.svc.GetDueCards(r.Context(), userID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get review queue")
		return
	}

	cards := make([]dto.ReviewCardDTO, 0, len(result.Cards))
	for _, c := range result.Cards {
		card := dto.ReviewCardDTO{
			CardID:         c.CardID,
			UserCardStateID: c.UserCardStateID,
			FrontText:      c.FrontText,
			BackText:       c.BackText,
			DueAt:          c.DueAt.UTC().Format(time.RFC3339),
		}
		if c.ExampleText.Valid {
			card.ExampleText = c.ExampleText.String
		}
		cards = append(cards, card)
	}

	writeJSON(w, http.StatusOK, dto.ReviewQueueResponse{
		DueCount: result.DueCount,
		Cards:    cards,
	})
}

func (h *ReviewHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	var req dto.ReviewSubmitRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	reviewedAt, err := time.Parse(time.RFC3339, req.ReviewedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid reviewed_at format")
		return
	}

	result, err := h.svc.SubmitReview(r.Context(), review.SubmitInput{
		UserID:          userID,
		CardID:          req.CardID,
		UserCardStateID: req.UserCardStateID,
		Rating:          req.Rating,
		ReviewedAt:      reviewedAt,
		ResponseMs:      req.ResponseMs,
		ClientEventID:   req.ClientEventID,
		IdempotencyKey:  idempotencyKey,
	})
	if errors.Is(err, review.ErrCardStateNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "card state not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to submit review")
		return
	}

	writeJSON(w, http.StatusOK, dto.ReviewSubmitResponse{
		Accepted:      result.Accepted,
		CardID:        result.CardID,
		NextDueAt:     result.NextDueAt.UTC().Format(time.RFC3339),
		ScheduledDays: result.ScheduledDays,
		Status:        result.Status,
	})
}
