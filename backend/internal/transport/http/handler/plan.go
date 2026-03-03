package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bennyshi/english-anywhere-lab/internal/plan"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

type PlanHandler struct {
	svc *plan.Service
}

func NewPlanHandler(svc *plan.Service) *PlanHandler {
	return &PlanHandler{svc: svc}
}

func (h *PlanHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req dto.BootstrapPlanRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	result, err := h.svc.BootstrapPlan(r.Context(), plan.BootstrapInput{
		UserID:       userID,
		Level:        req.Level,
		TargetDomain: req.TargetDomain,
		DailyMinutes: req.DailyMinutes,
		Days:         req.Days,
	})
	if errors.Is(err, plan.ErrPlanExists) {
		writeError(w, http.StatusConflict, "CONFLICT", "plan already exists for this week")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to bootstrap plan")
		return
	}

	dailyPlans := make([]dto.DailyPlanDTO, 0, len(result.DailyPlans))
	for _, dp := range result.DailyPlans {
		tasks := make([]dto.PlanTaskDTO, 0, len(dp.Tasks))
		for _, t := range dp.Tasks {
			tasks = append(tasks, dto.PlanTaskDTO{
				TaskID:           t.TaskID,
				TaskType:         t.TaskType,
				Title:            t.Title,
				Status:           t.Status,
				EstimatedMinutes: t.EstimatedMinutes,
				Virtual:          t.Virtual,
			})
		}
		dailyPlans = append(dailyPlans, dto.DailyPlanDTO{
			PlanID:             dp.PlanID,
			Date:               dp.Date,
			Mode:               dp.Mode,
			TotalEstimatedMins: dp.TotalEstimatedMins,
			Tasks:              tasks,
		})
	}

	writeJSON(w, http.StatusOK, dto.WeeklyPlanResponse{
		WeekStart:  result.WeekStart,
		DailyPlans: dailyPlans,
	})
}

func (h *PlanHandler) GetToday(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	timezone := r.URL.Query().Get("timezone")

	result, err := h.svc.GetTodayPlan(r.Context(), userID, timezone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get today plan")
		return
	}

	tasks := make([]dto.PlanTaskDTO, 0, len(result.Tasks))
	for _, t := range result.Tasks {
		tasks = append(tasks, dto.PlanTaskDTO{
			TaskID:           t.TaskID,
			TaskType:         t.TaskType,
			Title:            t.Title,
			Status:           t.Status,
			EstimatedMinutes: t.EstimatedMinutes,
			Virtual:          t.Virtual,
		})
	}

	writeJSON(w, http.StatusOK, dto.DailyPlanResponse{
		DailyPlan: dto.DailyPlanDTO{
			PlanID:             result.PlanID,
			Date:               result.Date,
			Mode:               result.Mode,
			TotalEstimatedMins: result.TotalEstimatedMins,
			Tasks:              tasks,
		},
	})
}

func (h *PlanHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	planID := chi.URLParam(r, "plan_id")
	taskID := chi.URLParam(r, "task_id")

	var req dto.CompleteTaskRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	result, err := h.svc.CompleteTask(r.Context(), userID, planID, taskID, req.CompletedAt, req.DurationSeconds)
	if errors.Is(err, plan.ErrTaskNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to complete task")
		return
	}

	writeJSON(w, http.StatusOK, dto.TaskCompletionResponse{
		TaskID: result.TaskID,
		Status: "completed",
	})
}
