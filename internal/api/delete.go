package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type DeleteResponse struct {
	Message string `json:"message"`
}

func DeleteFunctionHandler(w http.ResponseWriter, r *http.Request) {
	response := DeleteResponse{
		Message: "Function Deleted (still working)",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode delete response", "error", err)
	}
}
