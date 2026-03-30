package api

import (
	"encoding/json"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"
	"net/http"
)

type GetFunctionsResponse struct {
	Functions any `json:"functions"`
}

// GetFunctionsHandler returns a list of all deployed functions.
// Queries the database and returns function metadata.
func GetFunctionsHandler(w http.ResponseWriter, r *http.Request) {

	functions, err := store.ListFunctions(sqlite.DB)
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
