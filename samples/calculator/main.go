package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", calculator).Methods("POST")

	log.Println("Listening on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func calculator(w http.ResponseWriter, r *http.Request) {

	type request struct {
		A  int    `json:"a"`
		B  int    `json:"b"`
		Op string `json:"op"`
	}

	type response struct {
		Result int `json:"result"`
	}

	var data request

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch data.Op {
	case "+":
		data.A += data.B
	case "-":
		data.A -= data.B
	case "*":
		data.A *= data.B
	case "/":
		if data.B == 0 {
			http.Error(w, "cannot divide by zero", http.StatusBadRequest)
			return
		}
		data.A /= data.B
	case "%":
		if data.B == 0 {
			http.Error(w, "cannot divide by zero", http.StatusBadRequest)
			return
		}
		data.A %= data.B
	default:
		http.Error(w, "invalid operator", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{
		Result: data.A,
	})
}
