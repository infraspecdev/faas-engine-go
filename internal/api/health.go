package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func GreetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	type response struct {
		Message string `json:"message"`
	}

	w.Header().Set("Content-Type", "application/json")

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)

		if err := json.NewEncoder(w).Encode(response{
			Message: "Missing 'name' query parameter",
		}); err != nil {
			slog.Error("failed to encode error response", "error", err)
		}

		return
	}

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response{
		Message: fmt.Sprintf("Hello, %s!", name),
	}); err != nil {
		slog.Error("failed to encode greet response", "error", err)
	}
}
