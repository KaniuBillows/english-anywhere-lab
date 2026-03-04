package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/bennyshi/english-anywhere-lab/internal/pack"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

type PackHandler struct {
	svc *pack.Service
}

func NewPackHandler(svc *pack.Service) *PackHandler {
	return &PackHandler{svc: svc}
}

func (h *PackHandler) ListPacks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page := 1
	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	pageSize := 20
	if v := q.Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}

	result, err := h.svc.ListPacks(r.Context(), pack.ListInput{
		Domain:   q.Get("domain"),
		Level:    q.Get("level"),
		Source:   q.Get("source"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to list packs")
		return
	}

	items := make([]dto.PackDTO, 0, len(result.Packs))
	for _, p := range result.Packs {
		items = append(items, toPackDTO(p))
	}

	writeJSON(w, http.StatusOK, dto.PackListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (h *PackHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	packID := chi.URLParam(r, "pack_id")

	result, err := h.svc.GetDetail(r.Context(), packID)
	if errors.Is(err, pack.ErrPackNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pack not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get pack")
		return
	}

	lessons := make([]dto.LessonDTO, 0, len(result.Lessons))
	for _, l := range result.Lessons {
		lessons = append(lessons, dto.LessonDTO{
			LessonID:   l.ID,
			Title:      l.Title,
			LessonType: l.LessonType,
			Position:   l.Position,
		})
	}

	writeJSON(w, http.StatusOK, dto.PackDetailResponse{
		Pack:    toPackDTO(result.Pack),
		Lessons: lessons,
	})
}

func (h *PackHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	packID := chi.URLParam(r, "pack_id")

	err := h.svc.Enroll(r.Context(), userID, packID)
	if errors.Is(err, pack.ErrPackNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pack not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to enroll")
		return
	}

	writeJSON(w, http.StatusOK, dto.GenericMessage{Message: "enrolled successfully"})
}

func (h *PackHandler) CreateGenerationJob(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req dto.GeneratePackRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	job, err := h.svc.CreateGenerationJob(r.Context(), pack.GenerateInput{
		UserID:       userID,
		Level:        req.Level,
		Domain:       req.Domain,
		DailyMinutes: req.DailyMinutes,
		Days:         req.Days,
		FocusSkills:  req.FocusSkills,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to create generation job")
		return
	}

	writeJSON(w, http.StatusAccepted, dto.GenerationJobResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *PackHandler) GetGenerationJob(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	jobID := chi.URLParam(r, "job_id")

	job, err := h.svc.GetGenerationJob(r.Context(), jobID, userID)
	if errors.Is(err, pack.ErrJobNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "generation job not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get generation job")
		return
	}

	resp := dto.GenerationJobResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if job.FinishedAt.Valid {
		resp.FinishedAt = job.FinishedAt.String
	}
	if job.ErrorMessage.Valid {
		resp.ErrorMessage = job.ErrorMessage.String
	}

	writeJSON(w, http.StatusOK, resp)
}

func toPackDTO(p pack.Pack) dto.PackDTO {
	d := dto.PackDTO{
		ID:               p.ID,
		Source:           p.Source,
		Title:            p.Title,
		Domain:           p.Domain,
		Level:            p.Level,
		EstimatedMinutes: p.EstimatedMinutes,
	}
	if p.Description.Valid {
		d.Description = p.Description.String
	}
	return d
}
