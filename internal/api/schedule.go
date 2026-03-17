package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Scheduler interface {
	AddSchedule(functionName string, cronExpr string, payload []byte) error
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
