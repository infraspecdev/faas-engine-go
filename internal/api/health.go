package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func GreetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	type response struct {
		Message string `json:"message"`
	}

	w.Header().Set("Content-Type", "application/json")
	if name == "" {

		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response{
			Message: "Missing 'name' query parameter",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Message: fmt.Sprintf("Hello, %s!", name),
	})
}
