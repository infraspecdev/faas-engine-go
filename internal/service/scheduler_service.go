package service

import (
	"context"
	"fmt"
	"log/slog"

	"faas-engine-go/internal/types"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type SchedulerService struct {
	cron        *cron.Cron
	invoker     *FunctionInvoker
	schedules   map[string]types.Schedule // schedule data
	cronEntries map[string]cron.EntryID   // cron job IDs
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
	slog.Info("scheduler_started")
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

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		ctx := context.Background()

		slog.Info("scheduler_trigger",
			"function", functionName,
			"cron", cronExpr,
		)

		_, err := s.invoker.Invoke(ctx, functionName, payload)
		if err != nil {
			slog.Error("scheduled_invocation_failed",
				"function", functionName,
				"error", err,
			)
		}
	})

	if err != nil {
		return err
	}

	s.schedules[id] = schedule
	s.cronEntries[id] = entryID

	slog.Info("schedule_created",
		"schedule_id", id,
		"function", functionName,
		"cron", cronExpr,
	)

	return nil
}
