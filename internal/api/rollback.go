package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type RollbackRequest struct {
	TargetVersion string `json:"target_version,omitempty"`
}

type Roller interface {
	Rollback(ctx context.Context, functionName, targetVersion string) (*service.RollbackResult, error)
	GetRollbackHistory(functionName string, limit int) ([]map[string]interface{}, error)
}

// RollbackHandler handles HTTP requests to rollback a function to a previous version.
// POST /functions/{functionName}/rollback
// Body: { "target_version": "v1" } or empty for previous version
func RollbackHandler(roller Roller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		var req RollbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && r.ContentLength > 0 {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		result, err := roller.Rollback(r.Context(), functionName, req.TargetVersion)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// RollbackHistoryHandler handles HTTP requests to fetch rollback history.
// GET /functions/{functionName}/history?limit=20
func RollbackHistoryHandler(roller Roller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		functionName := strings.TrimSpace(vars["functionName"])
		if functionName == "" {
			http.Error(w, "functionName is required", http.StatusBadRequest)
			return
		}

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		history, err := roller.GetRollbackHistory(functionName, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(history); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
