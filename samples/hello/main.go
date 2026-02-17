package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	type response struct {
		Message string `json:"message"`
	}

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{
			Message: "Hello, World!",
		})
	})

	log.Println("Listening on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
