package service

import (
	"context"
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
	"log/slog"

	"github.com/robfig/cron/v3"
)

type FunctionStore interface {
	GetFunctionByID(id int) (*models.Function, error)
}

type SchedulerService struct {
	cron      *cron.Cron
	invoker   core.Invoker
	entries   map[string]cron.EntryID // scheduleID → entryID
	semaphore chan struct{}
}

func NewSchedulerService(invoker core.Invoker) *SchedulerService {
	return &SchedulerService{
		cron:      cron.New(),
		invoker:   invoker,
		entries:   make(map[string]cron.EntryID),
		semaphore: make(chan struct{}, 1),
	}
}

// ---------- START ----------
func (s *SchedulerService) Start() {
	s.cron.Start()
	slog.Info("scheduler_started")
}

// ---------- STOP ----------core.Invoker
func (s *SchedulerService) Stop() {
	s.cron.Stop()
	slog.Info("scheduler_stopped")
}

// ---------- LOAD FROM DB ----------
func (s *SchedulerService) LoadSchedules() error {

	schedules, err := store.ListSchedules(sqlite.DB)
	if err != nil {
		return err
	}

	for _, sch := range schedules {
		if err := s.RegisterSchedule(sch); err != nil {
			slog.Error("failed_to_register_schedule",
				"id", sch.ID,
				"error", err,
			)
		}
	}

	slog.Info("schedules_loaded", "count", len(schedules))
	return nil
}

// ---------- REGISTER ----------
func (s *SchedulerService) RegisterSchedule(sch models.Schedule) error {

	// prevent duplicate
	if _, exists := s.entries[sch.ID]; exists {
		return nil
	}

	entryID, err := s.cron.AddFunc(sch.CronExpr, func() {

		s.semaphore <- struct{}{}
		defer func() { <-s.semaphore }()

		ctx := context.Background()

		// fetch latest function (safe)
		fn, err := store.GetFunctionByID(sqlite.DB, sch.FunctionID)
		if err != nil || fn == nil {
			slog.Error("function_not_found",
				"function_id", sch.FunctionID,
			)
			return
		}

		slog.Info("schedule_triggered",
			"schedule_id", sch.ID,
			"function", fn.Name,
		)

		_, err = s.invoker.Invoke(ctx, fn.Name, sch.Payload, "cron")
		if err != nil {
			slog.Error("schedule_invoke_failed",
				"schedule_id", sch.ID,
				"error", err,
			)
		}
	})

	if err != nil {
		return err
	}

	s.entries[sch.ID] = entryID
	return nil
}

// ---------- REMOVE ----------
func (s *SchedulerService) RemoveSchedule(scheduleID string) {

	if entryID, ok := s.entries[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, scheduleID)

		slog.Info("schedule_removed", "id", scheduleID)
	}
}
