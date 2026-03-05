package dto

import "encoding/json"

// SyncEventDTO represents a single sync event from the client.
type SyncEventDTO struct {
	ClientEventID string          `json:"client_event_id" validate:"required"`
	EventType     string          `json:"event_type" validate:"required"`
	OccurredAt    string          `json:"occurred_at" validate:"required"`
	Payload       json.RawMessage `json:"payload" validate:"required"`
}

// SyncEventsRequest is the request body for POST /sync/events.
type SyncEventsRequest struct {
	Events []SyncEventDTO `json:"events" validate:"required,dive"`
}

// SyncEventAckDTO represents per-event ack in the response.
type SyncEventAckDTO struct {
	ClientEventID string `json:"client_event_id"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
}

// SyncEventsResponse is the response body for POST /sync/events.
type SyncEventsResponse struct {
	Acks         []SyncEventAckDTO `json:"acks"`
	ServerCursor string            `json:"server_cursor"`
}

// SyncChangeDTO represents a single server-side change.
type SyncChangeDTO struct {
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Op         string          `json:"op"`
	Payload    json.RawMessage `json:"payload"`
}

// SyncChangesResponse is the response body for GET /sync/changes.
type SyncChangesResponse struct {
	NextCursor string          `json:"next_cursor"`
	Changes    []SyncChangeDTO `json:"changes"`
}
