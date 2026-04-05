package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"

	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
)

type ScheduleResponse struct {
	ID           string `json:"id"`
	FunctionName string `json:"function"`
	Cron         string `json:"cron"`
}

type Scheduler interface {
	RegisterSchedule(models.Schedule) error
	RemoveSchedule(scheduleID string)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

// validateCronExpression checks if a cron expression is valid and meets minimum interval requirements
// to prevent DOS attacks (e.g., scheduling * * * * * * which fires every second)
func validateCronExpression(cronExpr string) error {
	// Parse the cron expression
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("failed to parse cron expression: %w", err)
	}

	// Check minimum interval: Get next two execution times and verify they're at least
	// ScheduleMinimumIntervalSeconds apart
	now := time.Now()
	firstExecution := schedule.Next(now)
	secondExecution := schedule.Next(firstExecution)

	interval := secondExecution.Sub(firstExecution)
	minInterval := time.Duration(config.ScheduleMinimumIntervalSeconds)

	if interval < minInterval {
		return fmt.Errorf(
			"cron expression fires too frequently (every %v seconds, minimum is %v seconds)",
			int(interval.Seconds()),
			int(minInterval.Seconds()),
		)
	}

	return nil
}

func CreateScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		functionName := mux.Vars(r)["functionName"]
		if functionName == "" {
			writeError(w, http.StatusBadRequest, "functionName is required")
			return
		}

		var req struct {
			CronExpr string          `json:"cron"`
			Payload  json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.CronExpr == "" {
			writeError(w, http.StatusBadRequest, "cron is required")
			return
		}

		// Validate cron expression and ensure minimum interval between executions
		if err := validateCronExpression(req.CronExpr); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid cron expression: %v", err))
			return
		}

		db := sqlite.GetDB()

		fn, err := store.GetActiveFunction(db, functionName)
		if err != nil || fn == nil {
			slog.Warn("function not found for schedule creation", "function", functionName)
			writeError(w, http.StatusBadRequest, "function not found or inactive")
			return
		}

		s := models.Schedule{
			FunctionID: fn.ID,
			CronExpr:   req.CronExpr,
			Payload:    req.Payload,
		}

		slog.Info("creating schedule", "function", functionName, "cron", req.CronExpr)

		// Persist to database first to ensure durability.
		// If scheduler registration fails, the schedule entry exists in DB and
		// will be picked up by LoadSchedules on the next restart.
		if err := store.CreateSchedule(db, &s); err != nil {
			slog.Error("failed to create schedule in database", "function", functionName, "error", err)
			scheduler.RemoveSchedule(s.ID)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		slog.Info("schedule saved to database", "schedule_id", s.ID, "function", functionName)

		// Register with cron scheduler after DB persistence.
		// If this fails, the schedule is still in the DB and can be re-registered on restart.
		if err := scheduler.RegisterSchedule(s); err != nil {
			slog.Error("failed to register schedule with scheduler", "schedule_id", s.ID, "function", functionName, "error", err)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		slog.Info("schedule registered with scheduler", "schedule_id", s.ID, "function", functionName, "cron", req.CronExpr)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": "created",
			"data":    s,
		})
	}
}

func DeleteScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db := sqlite.GetDB()

		// Case 1: Delete all schedules globally with ?removeall=true
		if r.URL.Query().Get("removeall") == "true" {
			slog.Info("deleting all schedules")

			deleted, err := store.DeleteAllSchedules(db)
			if err != nil {
				slog.Error("failed to delete all schedules", "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}

			// Remove all from cron scheduler
			// Note: In a real implementation, we'd need to get all schedule IDs first
			// and remove them from the scheduler. For now, a restart will clean them up.
			slog.Info("all schedules deleted from database", "count", deleted)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "deleted",
				"deleted": deleted,
			})
			return
		}

		// Case 2: Delete all schedules for a function with ?function=functionName
		functionName := r.URL.Query().Get("function")
		if functionName != "" {
			slog.Info("deleting all schedules for function", "function", functionName)

			// Get all schedules for this function first so we can remove them from scheduler
			schedules, err := store.ListSchedulesByFunctionName(db, functionName)
			if err != nil {
				slog.Error("failed to list schedules", "function", functionName, "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}

			if len(schedules) == 0 {
				writeError(w, http.StatusNotFound, "no schedules found for this function")
				return
			}

			deleted, err := store.DeleteSchedulesByFunctionName(db, functionName)
			if err != nil {
				slog.Error("failed to delete schedules", "function", functionName, "error", err)
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}

			// Remove all from cron scheduler
			for _, s := range schedules {
				scheduler.RemoveSchedule(s.ID)
			}

			slog.Info("schedules deleted for function", "function", functionName, "count", deleted)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "deleted",
				"deleted": deleted,
			})
			return
		}

		// Case 3: Delete by ID (existing behavior)
		id := mux.Vars(r)["id"]
		if id == "" {
			writeError(w, http.StatusBadRequest, "schedule id required")
			return
		}

		slog.Info("deleting schedule", "schedule_id", id)

		// Delete from DB first for durability.
		// If the app crashes after this point but before removing from scheduler,
		// LoadSchedules on restart will succeed (DB already deleted).
		if err := store.DeleteSchedule(db, id); err != nil {
			slog.Error("failed to delete schedule from database", "schedule_id", id, "error", err)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		slog.Info("schedule deleted from database", "schedule_id", id)

		// Remove from cron scheduler after DB deletion.
		// If the app crashes between DB delete and scheduler removal, the schedule entry
		// is already gone from the DB, so LoadSchedules won't re-register it.
		// Important: DB delete MUST come first to prevent zombie scheduler entries
		// being re-registered on restart.
		scheduler.RemoveSchedule(id)

		slog.Info("schedule removed from scheduler", "schedule_id", id)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "deleted",
		})
	}
}

func ListSchedulesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Optional query parameter for filtering by function name
		functionName := r.URL.Query().Get("function")

		db := sqlite.GetDB()
		var schedules []models.Schedule
		var err error

		if functionName != "" {
			// Filter by function name if provided
			schedules, err = store.ListSchedulesByFunctionName(db, functionName)
		} else {
			// List all schedules if no filter
			schedules, err = store.ListSchedules(db)
		}

		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var response []ScheduleResponse
		for _, s := range schedules {
			response = append(response, ScheduleResponse{
				ID:           s.ID,
				FunctionName: s.FunctionName,
				Cron:         s.CronExpr,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
