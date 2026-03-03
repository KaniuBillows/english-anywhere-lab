package dto

type ReviewQueueResponse struct {
	DueCount int             `json:"due_count"`
	Cards    []ReviewCardDTO `json:"cards"`
}

type ReviewCardDTO struct {
	CardID         string `json:"card_id"`
	UserCardStateID string `json:"user_card_state_id"`
	FrontText      string `json:"front_text"`
	BackText       string `json:"back_text"`
	ExampleText    string `json:"example_text,omitempty"`
	DueAt          string `json:"due_at"`
}

type ReviewSubmitRequest struct {
	CardID         string `json:"card_id" validate:"required"`
	UserCardStateID string `json:"user_card_state_id" validate:"required"`
	Rating         string `json:"rating" validate:"required,oneof=again hard good easy"`
	ReviewedAt     string `json:"reviewed_at" validate:"required"`
	ResponseMs     *int   `json:"response_ms"`
	ClientEventID  string `json:"client_event_id" validate:"required"`
}

type ReviewSubmitResponse struct {
	Accepted      bool   `json:"accepted"`
	CardID        string `json:"card_id"`
	NextDueAt     string `json:"next_due_at"`
	ScheduledDays int    `json:"scheduled_days"`
	Status        string `json:"status"`
}
