package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type Logger interface {
	GetLogsByName(functionName string, limit int) ([]service.LogEntry, error)
	GetLogsByNameAndVersion(functionName, version string, limit int) ([]service.LogEntry, error)
}

func LogHandler(logger Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		version := strings.TrimSpace(r.URL.Query().Get("version"))

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		var logs []service.LogEntry
		var err error

		if version != "" {
			logs, err = logger.GetLogsByNameAndVersion(functionName, version, limit)
		} else {
			logs, err = logger.GetLogsByName(functionName, limit)
		}

		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, "failed to fetch logs", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(logs)
	}
}
