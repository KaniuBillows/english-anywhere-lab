package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/bennyshi/english-anywhere-lab/internal/output"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

// OutputHandler handles output task endpoints.
type OutputHandler struct {
	svc *output.Service
}

// NewOutputHandler creates a new OutputHandler.
func NewOutputHandler(svc *output.Service) *OutputHandler {
	return &OutputHandler{svc: svc}
}

// ListTasks returns writing tasks for a lesson.
// GET /api/v1/lessons/{lesson_id}/output-tasks
func (h *OutputHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	lessonID := chi.URLParam(r, "lesson_id")

	tasks, err := h.svc.ListTasks(r.Context(), lessonID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to list output tasks")
		return
	}

	items := make([]dto.OutputTaskDTO, 0, len(tasks))
	for _, t := range tasks {
		d := dto.OutputTaskDTO{
			ID:         t.ID,
			TaskType:   t.TaskType,
			PromptText: t.PromptText,
		}
		if t.LessonID.Valid {
			d.LessonID = t.LessonID.String
		}
		if t.ReferenceAnswer.Valid {
			d.ReferenceAnswer = t.ReferenceAnswer.String
		}
		if t.Level.Valid {
			d.Level = t.Level.String
		}
		items = append(items, d)
	}

	writeJSON(w, http.StatusOK, dto.OutputTaskListResponse{Items: items})
}

// SubmitWriting handles writing task submission with synchronous LLM feedback.
// POST /api/v1/output-tasks/{task_id}/submit
func (h *OutputHandler) SubmitWriting(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "task_id")

	var req dto.SubmitWritingRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	result, err := h.svc.Submit(r.Context(), output.SubmitInput{
		UserID:     userID,
		TaskID:     taskID,
		AnswerText: req.AnswerText,
	})
	if errors.Is(err, output.ErrTaskNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "output task not found")
		return
	}
	if errors.Is(err, output.ErrNotWritingTask) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "task is not a writing task")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to submit writing")
		return
	}

	writeJSON(w, http.StatusOK, toSubmissionResponse(result))
}

// GetSubmission returns a submission by ID.
// GET /api/v1/output-tasks/submissions/{submission_id}
func (h *OutputHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	submissionID := chi.URLParam(r, "submission_id")

	result, err := h.svc.GetSubmission(r.Context(), submissionID, userID)
	if errors.Is(err, output.ErrSubmissionNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "submission not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to get submission")
		return
	}

	writeJSON(w, http.StatusOK, toSubmissionResponse(result))
}

func toSubmissionResponse(r *output.SubmitResult) dto.SubmissionResponse {
	resp := dto.SubmissionResponse{
		SubmissionID: r.SubmissionID,
		TaskID:       r.TaskID,
		AnswerText:   r.AnswerText,
		Score:        r.Score,
		SubmittedAt:  r.SubmittedAt,
	}
	if r.Feedback != nil {
		errs := make([]dto.WritingErrorDTO, 0, len(r.Feedback.Errors))
		for _, e := range r.Feedback.Errors {
			errs = append(errs, dto.WritingErrorDTO{
				Original:    e.Original,
				Correction:  e.Correction,
				Explanation: e.Explanation,
			})
		}
		resp.Feedback = &dto.WritingFeedbackDTO{
			OverallScore: r.Feedback.OverallScore,
			Errors:       errs,
			RevisedText:  r.Feedback.RevisedText,
			NextActions:  r.Feedback.NextActions,
		}
	}
	return resp
}
