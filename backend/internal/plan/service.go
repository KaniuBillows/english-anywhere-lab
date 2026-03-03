package plan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPlanExists  = errors.New("plan already exists for this week")
	ErrPlanNotFound = errors.New("plan not found")
	ErrTaskNotFound = errors.New("task not found")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type BootstrapInput struct {
	UserID       string
	Level        string
	TargetDomain string
	DailyMinutes int
	Days         int
}

type BootstrapResult struct {
	WeekStart string
	DailyPlans []DailyPlanResult
}

type DailyPlanResult struct {
	PlanID              string
	Date                string
	Mode                string
	TotalEstimatedMins  int
	Tasks               []TaskResult
}

type TaskResult struct {
	TaskID           string
	TaskType         string
	Title            string
	Status           string
	EstimatedMinutes int
	Virtual          bool
}

func (s *Service) BootstrapPlan(ctx context.Context, input BootstrapInput) (*BootstrapResult, error) {
	now := time.Now().UTC()
	weekStart := getWeekStart(now).Format("2006-01-02")

	// Check if plan already exists
	_, err := s.repo.GetPlanByUserAndWeek(ctx, input.UserID, weekStart)
	if err == nil {
		return nil, ErrPlanExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing plan: %w", err)
	}

	days := input.Days
	if days == 0 {
		days = 7
	}

	plan := &Plan{
		ID:        uuid.New().String(),
		UserID:    input.UserID,
		WeekStart: weekStart,
		CreatedAt: now,
	}
	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("create plan: %w", err)
	}

	// Generate tasks for each day
	result := &BootstrapResult{WeekStart: weekStart}
	startDate := getWeekStart(now)

	for d := 0; d < days; d++ {
		date := startDate.AddDate(0, 0, d)
		dateStr := date.Format("2006-01-02")
		mode := "full_20m"
		if input.DailyMinutes <= 10 {
			mode = "fast_10m"
		}

		tasks := s.generateDayTasks(plan.ID, dateStr, input.DailyMinutes, input.Level)
		for _, task := range tasks {
			if err := s.repo.CreatePlanTask(ctx, &task); err != nil {
				return nil, fmt.Errorf("create task: %w", err)
			}
		}

		totalMins := 0
		taskResults := make([]TaskResult, 0, len(tasks))
		for _, t := range tasks {
			totalMins += t.EstimatedMinutes
			taskResults = append(taskResults, TaskResult{
				TaskID:           t.ID,
				TaskType:         t.TaskType,
				Title:            t.Title,
				Status:           t.Status,
				EstimatedMinutes: t.EstimatedMinutes,
			})
		}

		result.DailyPlans = append(result.DailyPlans, DailyPlanResult{
			PlanID:             plan.ID,
			Date:               dateStr,
			Mode:               mode,
			TotalEstimatedMins: totalMins,
			Tasks:              taskResults,
		})
	}

	return result, nil
}

func (s *Service) GetTodayPlan(ctx context.Context, userID, timezone string) (*DailyPlanResult, error) {
	now := time.Now().UTC()
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			now = time.Now().In(loc)
		}
	}

	// Apply day cutoff at 04:00
	if now.Hour() < 4 {
		now = now.AddDate(0, 0, -1)
	}
	today := now.Format("2006-01-02")

	// Find latest plan
	plan, err := s.repo.GetLatestPlan(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		// No plan exists — return empty plan
		return &DailyPlanResult{
			PlanID:             "",
			Date:               today,
			Mode:               "full_20m",
			TotalEstimatedMins: 0,
			Tasks:              []TaskResult{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest plan: %w", err)
	}

	tasks, err := s.repo.GetTasksByPlanAndDate(ctx, plan.ID, today)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}

	totalMins := 0
	taskResults := make([]TaskResult, 0, len(tasks))

	for _, t := range tasks {
		totalMins += t.EstimatedMinutes
		taskResults = append(taskResults, TaskResult{
			TaskID:           t.ID,
			TaskType:         t.TaskType,
			Title:            t.Title,
			Status:           t.Status,
			EstimatedMinutes: t.EstimatedMinutes,
		})
	}

	mode := "full_20m"
	if totalMins <= 10 {
		mode = "fast_10m"
	}

	return &DailyPlanResult{
		PlanID:             plan.ID,
		Date:               today,
		Mode:               mode,
		TotalEstimatedMins: totalMins,
		Tasks:              taskResults,
	}, nil
}

func (s *Service) CompleteTask(ctx context.Context, userID, planID, taskID, completedAt string, durationSeconds *int) (*TaskResult, error) {
	task, err := s.repo.GetTaskWithOwner(ctx, taskID, planID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	if err := s.repo.CompleteTask(ctx, taskID, completedAt, durationSeconds); err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}

	return &TaskResult{
		TaskID:           task.ID,
		TaskType:         task.TaskType,
		Title:            task.Title,
		Status:           "completed",
		EstimatedMinutes: task.EstimatedMinutes,
	}, nil
}

// generateDayTasks creates a default set of tasks for a day.
func (s *Service) generateDayTasks(planID, date string, dailyMinutes int, level string) []PlanTask {
	var tasks []PlanTask

	// Input task (reading/listening)
	inputMins := dailyMinutes / 2
	if inputMins < 5 {
		inputMins = 5
	}
	tasks = append(tasks, PlanTask{
		ID:               uuid.New().String(),
		PlanID:           planID,
		TaskDate:         date,
		TaskType:         "input",
		Title:            fmt.Sprintf("%s level input practice", level),
		Status:           "pending",
		EstimatedMinutes: inputMins,
	})

	// Review task
	reviewMins := dailyMinutes / 3
	if reviewMins < 5 {
		reviewMins = 5
	}
	tasks = append(tasks, PlanTask{
		ID:               uuid.New().String(),
		PlanID:           planID,
		TaskDate:         date,
		TaskType:         "review",
		Title:            "Spaced repetition review",
		Status:           "pending",
		EstimatedMinutes: reviewMins,
	})

	return tasks
}

// getWeekStart returns the Monday of the week for the given time.
func getWeekStart(t time.Time) time.Time {
	weekday := t.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	return t.AddDate(0, 0, -int(weekday-time.Monday)).Truncate(24 * time.Hour)
}
