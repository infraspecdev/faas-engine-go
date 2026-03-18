package api

import (
	"encoding/json"
	"faas-engine-go/internal/types"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Scheduler interface {
	AddSchedule(functionName string, cronExpr string, payload []byte) error
	DeleteSchedule(scheduleID string) error
	ListSchedules() []types.Schedule
	GetSchedulesByFunction(functionName string) []types.Schedule
}

type ScheduleRequest struct {
	Cron    string          `json:"cron"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func ScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])

		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		var req ScheduleRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		cronExpr := strings.TrimSpace(req.Cron)
		if cronExpr == "" {
			http.Error(w, "cron expression is required", http.StatusBadRequest)
			return
		}

		payload := req.Payload
		if len(payload) == 0 {
			payload = []byte("{}")
		}

		err = scheduler.AddSchedule(functionName, cronExpr, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("schedule created"))
	}
}

func DeleteScheduleHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		id := strings.TrimSpace(vars["scheduleID"])

		if id == "" {
			http.Error(w, "scheduleID is required", http.StatusBadRequest)
			return
		}

		err := scheduler.DeleteSchedule(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("schedule deleted"))
	}
}

func ListSchedulesHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		schedules := scheduler.ListSchedules()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schedules)
	}
}

func GetSchedulesByFunctionHandler(scheduler Scheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])

		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		schedules := scheduler.GetSchedulesByFunction(functionName)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schedules)
	}
}
