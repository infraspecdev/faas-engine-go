package api

import (
	"encoding/json"
	"net/http"
)

type DeleteResponse struct {
	Message string `json:"message"`
}

func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	response := DeleteResponse{
		Message: "Function Deleted (still working)",
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
