package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", echo).Methods("POST")

	log.Println("Listening on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func echo(w http.ResponseWriter, r *http.Request) {

	var data any

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
