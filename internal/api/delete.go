package api

import (
	"encoding/json"
	"net/http"
)

type DeleteResponse struct {
	Message string `json:"message"`
}

// DeleteFunctionHandler handles HTTP requests for deleting a function.
// Currently returns a placeholder success response.
func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	response := DeleteResponse{
		Message: "Function Deleted (still working)",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
