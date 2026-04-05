package service

import (
	"context"
	"database/sql"
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"
)

type SchedulerService struct {
	cron               *cron.Cron
	invoker            core.Invoker
	db                 *sql.DB                  // injected database connection
	entries            map[string]cron.EntryID  // scheduleID → entryID
	scheduleSemaphores map[string]chan struct{} // scheduleID → semaphore (per-schedule concurrency guard)
	semaphoreMu        sync.Mutex               // protects scheduleSemaphores map
}

func NewSchedulerService(invoker core.Invoker, db *sql.DB) *SchedulerService {
	// Configure cron to accept 6-field format (with seconds) to match API validation
	// This ensures schedules loaded from DB are registered correctly
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return &SchedulerService{
		cron:               cron.New(cron.WithParser(parser)),
		invoker:            invoker,
		db:                 db,
		entries:            make(map[string]cron.EntryID),
		scheduleSemaphores: make(map[string]chan struct{}),
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

	schedules, err := store.ListSchedules(s.db)
	if err != nil {
		return err
	}

	var orphanedCount int
	for _, sch := range schedules {
		// Validate that the function still exists before registering the schedule
		fn, err := store.GetFunctionByID(s.db, sch.FunctionID)
		if err != nil || fn == nil {
			// Function does not exist - this is an orphaned schedule entry
			slog.Warn("orphaned_schedule_detected",
				"schedule_id", sch.ID,
				"function_id", sch.FunctionID,
				"reason", "function_not_found",
			)

			// Delete the orphaned schedule from the database
			if err := store.DeleteSchedule(s.db, sch.ID); err != nil {
				slog.Error("failed_to_delete_orphaned_schedule",
					"schedule_id", sch.ID,
					"error", err,
				)
			} else {
				orphanedCount++
			}
			continue
		}

		// Function exists - safe to register the schedule
		if err := s.RegisterSchedule(sch); err != nil {
			slog.Error("failed_to_register_schedule",
				"id", sch.ID,
				"error", err,
			)
		}
	}

	slog.Info("schedules_loaded",
		"total", len(schedules),
		"registered", len(schedules)-orphanedCount,
		"orphaned_deleted", orphanedCount,
	)
	return nil
}

// ---------- REGISTER ----------
func (s *SchedulerService) RegisterSchedule(sch models.Schedule) error {

	// prevent duplicate
	if _, exists := s.entries[sch.ID]; exists {
		return nil
	}

	// Create per-schedule semaphore to prevent concurrent invocations
	// of the same schedule (e.g., if cron fires every 1s but invocation takes 5s)
	s.semaphoreMu.Lock()
	if _, exists := s.scheduleSemaphores[sch.ID]; !exists {
		s.scheduleSemaphores[sch.ID] = make(chan struct{}, 1)
	}
	semaphore := s.scheduleSemaphores[sch.ID]
	s.semaphoreMu.Unlock()

	entryID, err := s.cron.AddFunc(sch.CronExpr, func() {

		// Try to acquire the semaphore without blocking if already busy.
		// If the previous invocation is still running, skip this execution.
		select {
		case semaphore <- struct{}{}:
			defer func() { <-semaphore }()
		default:
			slog.Warn("schedule_skipped_busy",
				"schedule_id", sch.ID,
				"reason", "previous_invocation_still_running",
			)
			return
		}

		ctx := context.Background()

		// fetch latest function (safe)
		fn, err := store.GetFunctionByID(s.db, sch.FunctionID)
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
	slog.Info("schedule_registered", "schedule_id", sch.ID, "cron", sch.CronExpr)
	return nil
}

// ---------- REMOVE ----------
func (s *SchedulerService) RemoveSchedule(scheduleID string) {

	if entryID, ok := s.entries[scheduleID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, scheduleID)

		// Clean up the per-schedule semaphore
		s.semaphoreMu.Lock()
		delete(s.scheduleSemaphores, scheduleID)
		s.semaphoreMu.Unlock()

		slog.Info("schedule_removed", "id", scheduleID)
	}
}
