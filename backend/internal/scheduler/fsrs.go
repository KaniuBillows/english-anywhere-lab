package scheduler

import (
	"math"
	"time"
)

// CardStatus represents the FSRS card state.
type CardStatus string

const (
	StatusNew        CardStatus = "new"
	StatusLearning   CardStatus = "learning"
	StatusReview     CardStatus = "review"
	StatusRelearning CardStatus = "relearning"
	StatusSuspended  CardStatus = "suspended"
)

// Rating represents user review rating.
type Rating string

const (
	RatingAgain Rating = "again"
	RatingHard  Rating = "hard"
	RatingGood  Rating = "good"
	RatingEasy  Rating = "easy"
)

// CardState holds the current scheduling state for a card.
type CardState struct {
	Status        CardStatus
	DueAt         time.Time
	Reps          int
	Lapses        int
	Stability     float64
	Difficulty    float64
	ElapsedDays   int
	ScheduledDays int
	LastReviewAt  time.Time
}

// ScheduleResult is the output of a scheduling calculation.
type ScheduleResult struct {
	Status        CardStatus
	DueAt         time.Time
	Reps          int
	Lapses        int
	Stability     float64
	Difficulty    float64
	ElapsedDays   int
	ScheduledDays int
}

// FSRS implements the fsrs_mvp_v1 scheduling spec from doc/08.
type FSRS struct{}

func NewFSRS() *FSRS {
	return &FSRS{}
}

// Schedule computes the next card state given a rating.
func (f *FSRS) Schedule(state CardState, rating Rating, now time.Time) ScheduleResult {
	result := ScheduleResult{
		Reps:       state.Reps + 1,
		Lapses:     state.Lapses,
		Stability:  state.Stability,
		Difficulty: state.Difficulty,
	}

	switch state.Status {
	case StatusNew, StatusLearning:
		result = f.scheduleLearning(state, rating, now, result)
	case StatusReview, StatusRelearning:
		result = f.scheduleReview(state, rating, now, result)
	}

	return result
}

// scheduleLearning handles new/learning phase cards (short intervals).
// Rules from doc/08 section 6.2:
//   - again -> due_at = now + 10m
//   - hard  -> due_at = now + 30m
//   - good  -> due_at = now + 1d, status = review
//   - easy  -> due_at = now + 3d, status = review
func (f *FSRS) scheduleLearning(state CardState, rating Rating, now time.Time, result ScheduleResult) ScheduleResult {
	switch rating {
	case RatingAgain:
		result.Status = StatusLearning
		result.DueAt = now.Add(10 * time.Minute)
		result.ScheduledDays = 0
	case RatingHard:
		result.Status = StatusLearning
		result.DueAt = now.Add(30 * time.Minute)
		result.ScheduledDays = 0
	case RatingGood:
		result.Status = StatusReview
		result.DueAt = now.Add(24 * time.Hour)
		result.ScheduledDays = 1
		result.Stability = 1.0
	case RatingEasy:
		result.Status = StatusReview
		result.DueAt = now.Add(3 * 24 * time.Hour)
		result.ScheduledDays = 3
		result.Stability = 3.0
	}

	if !state.LastReviewAt.IsZero() {
		result.ElapsedDays = int(now.Sub(state.LastReviewAt).Hours() / 24)
	}

	return result
}

// scheduleReview handles review/relearning phase cards (FSRS fallback).
// Rules from doc/08 section 6.3:
//   - again: interval = 1d, lapses+1, status = relearning
//   - hard:  interval = max(1, round(prev * 1.2))
//   - good:  interval = max(1, round(prev * 2.2))
//   - easy:  interval = max(2, round(prev * 3.0))
func (f *FSRS) scheduleReview(state CardState, rating Rating, now time.Time, result ScheduleResult) ScheduleResult {
	prevInterval := state.ScheduledDays
	if prevInterval < 1 {
		prevInterval = 1
	}

	if !state.LastReviewAt.IsZero() {
		result.ElapsedDays = int(now.Sub(state.LastReviewAt).Hours() / 24)
	}

	switch rating {
	case RatingAgain:
		result.Status = StatusRelearning
		result.ScheduledDays = 1
		result.DueAt = now.Add(24 * time.Hour)
		result.Lapses = state.Lapses + 1
		result.Stability = 1.0
	case RatingHard:
		result.Status = StatusReview
		newInterval := int(math.Max(1, math.Round(float64(prevInterval)*1.2)))
		result.ScheduledDays = newInterval
		result.DueAt = now.Add(time.Duration(newInterval) * 24 * time.Hour)
		result.Stability = float64(newInterval)
	case RatingGood:
		result.Status = StatusReview
		newInterval := int(math.Max(1, math.Round(float64(prevInterval)*2.2)))
		result.ScheduledDays = newInterval
		result.DueAt = now.Add(time.Duration(newInterval) * 24 * time.Hour)
		result.Stability = float64(newInterval)
	case RatingEasy:
		result.Status = StatusReview
		newInterval := int(math.Max(2, math.Round(float64(prevInterval)*3.0)))
		result.ScheduledDays = newInterval
		result.DueAt = now.Add(time.Duration(newInterval) * 24 * time.Hour)
		result.Stability = float64(newInterval)
	}

	return result
}
