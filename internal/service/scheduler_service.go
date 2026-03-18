package service

import (
	"context"
	"fmt"
	"log/slog"

	"faas-engine-go/internal/db"
	"faas-engine-go/internal/types"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type SchedulerService struct {
	cron        *cron.Cron
	invoker     *FunctionInvoker
	schedules   map[string]types.Schedule
	cronEntries map[string]cron.EntryID
}

func NewSchedulerService(invoker *FunctionInvoker) *SchedulerService {
	return &SchedulerService{
		cron:        cron.New(),
		invoker:     invoker,
		schedules:   make(map[string]types.Schedule),
		cronEntries: make(map[string]cron.EntryID),
	}
}

func (s *SchedulerService) Start() {
	s.cron.Start()
	schedules := db.ListSchedules()
	for _, schedule := range schedules {
		if err := s.registerSchedule(schedule); err != nil {
			slog.Error("failed_to_load_schedule", "schedule_id", schedule.ID, "error", err)
		}
	}
	slog.Info("scheduler_started", "loaded_schedules", len(schedules))
}

func (s *SchedulerService) AddSchedule(functionName, cronExpr string, payload []byte) error {
	if functionName == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if cronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	id := uuid.New().String()
	schedule := types.Schedule{
		ID:           id,
		FunctionName: functionName,
		CronExpr:     cronExpr,
		Payload:      payload,
	}

	// Validate and register first — only persist if successful
	if err := s.registerSchedule(schedule); err != nil {
		return err
	}

	db.AddSchedule(schedule)
	slog.Info("schedule_created",
		"schedule_id", id,
		"function", functionName,
		"cron", cronExpr,
	)
	return nil
}
func (s *SchedulerService) DeleteSchedule(id string) error {

	entryID, ok := s.cronEntries[id]
	if !ok {
		return fmt.Errorf("schedule not found")
	}

	// stop cron job
	s.cron.Remove(entryID)

	// remove from memory
	delete(s.cronEntries, id)
	delete(s.schedules, id)

	// ✅ remove from DB layer
	db.DeleteSchedule(id)

	slog.Info("schedule_deleted",
		"schedule_id", id,
	)

	return nil
}

func (s *SchedulerService) ListSchedules() []types.Schedule {
	return db.ListSchedules()
}

func (s *SchedulerService) GetSchedulesByFunction(functionName string) []types.Schedule {
	return db.GetSchedulesByFunction(functionName)
}

func (s *SchedulerService) registerSchedule(schedule types.Schedule) error {
	id := schedule.ID
	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		ctx := context.Background()
		slog.Info("scheduler_trigger",
			"schedule_id", id,
			"function", schedule.FunctionName,
			"cron", schedule.CronExpr,
		)
		_, err := s.invoker.Invoke(ctx, schedule.FunctionName, schedule.Payload)
		if err != nil {
			slog.Error("scheduled_invocation_failed",
				"schedule_id", id,
				"function", schedule.FunctionName,
				"error", err,
			)
		}
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.schedules[id] = schedule
	s.cronEntries[id] = entryID
	return nil
}
