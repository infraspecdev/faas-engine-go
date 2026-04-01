package api

import (
	"encoding/json"
	"faas-engine-go/internal/sqlite/models"
	"net/http"
)

type GetFunctionsResponse struct {
	Functions []models.Function `json:"functions"`
}

type FunctionLister interface {
	ListFunctions() ([]models.Function, error)
}

// ListFunctionsHandler returns a JSON list of available functions.
// Returns HTTP 200 with GetFunctionsResponse containing all active functions.
func ListFunctionsHandler(svc FunctionLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		functions, err := svc.ListFunctions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := GetFunctionsResponse{
			Functions: functions,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
