package api

import (
	"encoding/json"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
	"net/http"

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

// ---------- CREATE ----------
func CreateScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := vars["functionName"]

		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		var req struct {
			CronExpr string          `json:"cron"`
			Payload  json.RawMessage `json:"payload"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.CronExpr == "" {
			http.Error(w, "cron is required", http.StatusBadRequest)
			return
		}

		fn, err := store.GetActiveFunction(sqlite.DB, functionName)
		if err != nil || fn == nil {
			http.Error(w, "function not found or inactive", http.StatusBadRequest)
			return
		}

		s := models.Schedule{
			FunctionID: fn.ID,
			CronExpr:   req.CronExpr,
			Payload:    req.Payload,
		}

		// save to DB
		if err := store.CreateSchedule(sqlite.DB, &s); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// register in cron
		if err := scheduler.RegisterSchedule(s); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(s)
	}
}

// ---------- DELETE ----------
func DeleteScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := vars["id"]

		if id == "" {
			http.Error(w, "schedule id required", http.StatusBadRequest)
			return
		}

		// delete from DB
		if err := store.DeleteSchedule(sqlite.DB, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// remove from cron
		scheduler.RemoveSchedule(id)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"deleted"}`))
	}
}

// ---------- LIST ----------
func ListSchedulesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		schedules, err := store.ListSchedules(sqlite.DB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var response []ScheduleResponse
		for _, s := range schedules {
			response = append(response, ScheduleResponse{
				ID:           s.ID,
				FunctionName: s.FunctionName, // comes from JOIN
				Cron:         s.CronExpr,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func ListScheduleByFunctionNameHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		functionName := vars["functionName"]

		schedules, err := store.ListSchedulesByFunctionName(sqlite.DB, functionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
		json.NewEncoder(w).Encode(response)
	}
}
