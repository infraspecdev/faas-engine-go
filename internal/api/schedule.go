package api

import (
	"encoding/json"
	"net/http"

	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"

	"github.com/gorilla/mux"
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

		db := sqlite.GetDB()

		fn, err := store.GetActiveFunction(db, functionName)
		if err != nil || fn == nil {
			writeError(w, http.StatusBadRequest, "function not found or inactive")
			return
		}

		s := models.Schedule{
			FunctionID: fn.ID,
			CronExpr:   req.CronExpr,
			Payload:    req.Payload,
		}

		if err := scheduler.RegisterSchedule(s); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := store.CreateSchedule(db, &s); err != nil {
			scheduler.RemoveSchedule(s.ID)
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

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

		id := mux.Vars(r)["id"]
		if id == "" {
			writeError(w, http.StatusBadRequest, "schedule id required")
			return
		}

		db := sqlite.GetDB()

		if err := store.DeleteSchedule(db, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		scheduler.RemoveSchedule(id)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "deleted",
		})
	}
}

func ListSchedulesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		db := sqlite.GetDB()

		schedules, err := store.ListSchedules(db)
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

func ListScheduleByFunctionNameHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		functionName := mux.Vars(r)["functionName"]
		if functionName == "" {
			writeError(w, http.StatusBadRequest, "functionName is required")
			return
		}

		db := sqlite.GetDB()

		schedules, err := store.ListSchedulesByFunctionName(db, functionName)
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
