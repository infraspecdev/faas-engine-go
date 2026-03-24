package api

import (
	"encoding/json"
	"faas-engine-go/internal/sqlite/models"
	"net/http"
)

type GetFunctionsResponse struct {
	Functions any `json:"functions"`
}

type FunctionLister interface {
	ListFunctions() ([]models.Function, error)
}

// GreetHandler returns a JSON greeting message.
// Requires a "name" query parameter.
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
