package handler

import (
	"net/http"
	"strconv"

	"github.com/bennyshi/english-anywhere-lab/internal/sync"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/middleware"
)

// SyncHandler handles sync endpoints.
type SyncHandler struct {
	svc *sync.Service
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(svc *sync.Service) *SyncHandler {
	return &SyncHandler{svc: svc}
}

// PushEvents handles POST /api/v1/sync/events.
func (h *SyncHandler) PushEvents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req dto.SyncEventsRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "events array is required and must not be empty")
		return
	}

	if len(req.Events) > 500 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "batch size exceeds 500")
		return
	}

	events := make([]sync.EventInput, 0, len(req.Events))
	for _, e := range req.Events {
		events = append(events, sync.EventInput{
			ClientEventID: e.ClientEventID,
			EventType:     e.EventType,
			OccurredAt:    e.OccurredAt,
			Payload:       e.Payload,
		})
	}

	acks, cursor, err := h.svc.PushEvents(r.Context(), userID, events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to process sync events")
		return
	}

	ackDTOs := make([]dto.SyncEventAckDTO, 0, len(acks))
	for _, a := range acks {
		ackDTOs = append(ackDTOs, dto.SyncEventAckDTO{
			ClientEventID: a.ClientEventID,
			Status:        a.Status,
			Reason:        a.Reason,
		})
	}

	writeJSON(w, http.StatusOK, dto.SyncEventsResponse{
		Acks:         ackDTOs,
		ServerCursor: cursor,
	})
}

// PullChanges handles GET /api/v1/sync/changes.
func (h *SyncHandler) PullChanges(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	limit := 200
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	result, err := h.svc.PullChanges(r.Context(), userID, cursor, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	changes := make([]dto.SyncChangeDTO, 0, len(result.Changes))
	for _, c := range result.Changes {
		changes = append(changes, dto.SyncChangeDTO{
			EntityType: c.EntityType,
			EntityID:   c.EntityID,
			Op:         c.Op,
			Payload:    c.Payload,
		})
	}

	writeJSON(w, http.StatusOK, dto.SyncChangesResponse{
		NextCursor: result.NextCursor,
		Changes:    changes,
	})
}
