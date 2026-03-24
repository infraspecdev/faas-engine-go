package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type GetFunctionResponse struct {
	Message string `json:"message"`
}

func GetFunctionsHandler(w http.ResponseWriter, r *http.Request) {
	response := GetFunctionResponse{
		Message: "Hello world (still working)",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
