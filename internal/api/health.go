package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// HealthHandler returns "OK" to indicate the service is running.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprintf(w, "OK"); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

// GreetHandler returns a JSON greeting message.
// Requires a "name" query parameter.
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response{
		Message: fmt.Sprintf("Hello, %s!", name),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
