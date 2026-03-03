package scheduler

import (
	"testing"
	"time"
)

func TestSchedule_CaseA_NewCardGood(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{
		Status: StatusNew,
		DueAt:  now,
	}

	result := fsrs.Schedule(state, RatingGood, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	expectedDue := now.Add(24 * time.Hour)
	if !result.DueAt.Equal(expectedDue) {
		t.Errorf("expected due_at %v, got %v", expectedDue, result.DueAt)
	}
	if result.ScheduledDays != 1 {
		t.Errorf("expected scheduled_days 1, got %d", result.ScheduledDays)
	}
	if result.Reps != 1 {
		t.Errorf("expected reps 1, got %d", result.Reps)
	}
}

func TestSchedule_CaseB_ReviewCardAgain(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	lastReview := now.AddDate(0, 0, -7)

	state := CardState{
		Status:        StatusReview,
		DueAt:         now,
		Reps:          5,
		Lapses:        0,
		ScheduledDays: 7,
		Stability:     7.0,
		LastReviewAt:  lastReview,
	}

	result := fsrs.Schedule(state, RatingAgain, now)

	if result.Status != StatusRelearning {
		t.Errorf("expected status relearning, got %s", result.Status)
	}
	if result.Lapses != 1 {
		t.Errorf("expected lapses 1, got %d", result.Lapses)
	}
	expectedDue := now.Add(24 * time.Hour)
	if !result.DueAt.Equal(expectedDue) {
		t.Errorf("expected due_at %v, got %v", expectedDue, result.DueAt)
	}
	if result.ScheduledDays != 1 {
		t.Errorf("expected scheduled_days 1, got %d", result.ScheduledDays)
	}
}

func TestSchedule_NewCardAgain(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{Status: StatusNew, DueAt: now}
	result := fsrs.Schedule(state, RatingAgain, now)

	if result.Status != StatusLearning {
		t.Errorf("expected status learning, got %s", result.Status)
	}
	expectedDue := now.Add(10 * time.Minute)
	if !result.DueAt.Equal(expectedDue) {
		t.Errorf("expected due_at %v, got %v", expectedDue, result.DueAt)
	}
}

func TestSchedule_NewCardHard(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{Status: StatusNew, DueAt: now}
	result := fsrs.Schedule(state, RatingHard, now)

	if result.Status != StatusLearning {
		t.Errorf("expected status learning, got %s", result.Status)
	}
	expectedDue := now.Add(30 * time.Minute)
	if !result.DueAt.Equal(expectedDue) {
		t.Errorf("expected due_at %v, got %v", expectedDue, result.DueAt)
	}
}

func TestSchedule_NewCardEasy(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{Status: StatusNew, DueAt: now}
	result := fsrs.Schedule(state, RatingEasy, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	expectedDue := now.Add(3 * 24 * time.Hour)
	if !result.DueAt.Equal(expectedDue) {
		t.Errorf("expected due_at %v, got %v", expectedDue, result.DueAt)
	}
	if result.ScheduledDays != 3 {
		t.Errorf("expected scheduled_days 3, got %d", result.ScheduledDays)
	}
}

func TestSchedule_ReviewCardHard(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{
		Status:        StatusReview,
		DueAt:         now,
		Reps:          3,
		ScheduledDays: 10,
		Stability:     10.0,
	}

	result := fsrs.Schedule(state, RatingHard, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	// round(10 * 1.2) = 12
	if result.ScheduledDays != 12 {
		t.Errorf("expected scheduled_days 12, got %d", result.ScheduledDays)
	}
}

func TestSchedule_ReviewCardGood(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{
		Status:        StatusReview,
		DueAt:         now,
		Reps:          3,
		ScheduledDays: 10,
		Stability:     10.0,
	}

	result := fsrs.Schedule(state, RatingGood, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	// round(10 * 2.2) = 22
	if result.ScheduledDays != 22 {
		t.Errorf("expected scheduled_days 22, got %d", result.ScheduledDays)
	}
}

func TestSchedule_ReviewCardEasy(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{
		Status:        StatusReview,
		DueAt:         now,
		Reps:          3,
		ScheduledDays: 10,
		Stability:     10.0,
	}

	result := fsrs.Schedule(state, RatingEasy, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	// round(10 * 3.0) = 30
	if result.ScheduledDays != 30 {
		t.Errorf("expected scheduled_days 30, got %d", result.ScheduledDays)
	}
}

func TestSchedule_RelearningCardGood(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	state := CardState{
		Status:        StatusRelearning,
		DueAt:         now,
		Reps:          5,
		Lapses:        1,
		ScheduledDays: 1,
		Stability:     1.0,
	}

	result := fsrs.Schedule(state, RatingGood, now)

	if result.Status != StatusReview {
		t.Errorf("expected status review, got %s", result.Status)
	}
	// round(1 * 2.2) = 2
	if result.ScheduledDays != 2 {
		t.Errorf("expected scheduled_days 2, got %d", result.ScheduledDays)
	}
}

func TestSchedule_MinimumInterval(t *testing.T) {
	fsrs := NewFSRS()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	// With scheduled_days=1, hard: round(1 * 1.2) = 1 (min 1)
	state := CardState{
		Status:        StatusReview,
		DueAt:         now,
		ScheduledDays: 1,
	}

	result := fsrs.Schedule(state, RatingHard, now)
	if result.ScheduledDays != 1 {
		t.Errorf("expected scheduled_days 1 (minimum), got %d", result.ScheduledDays)
	}

	// Easy minimum is 2
	result = fsrs.Schedule(state, RatingEasy, now)
	if result.ScheduledDays != 3 {
		t.Errorf("expected scheduled_days 3 (round(1*3.0)), got %d", result.ScheduledDays)
	}
}
