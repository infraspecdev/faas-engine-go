package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--health" {
		resp, err := http.Get("http://localhost:8080/health")
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}
	r := mux.NewRouter()

	r.HandleFunc("/", calculator).Methods("POST")
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
	log.Println("Listening on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func calculator(w http.ResponseWriter, r *http.Request) {

	type request struct {
		A  int    `json:"a"`
		B  int    `json:"b"`
		Op string `json:"operation"`
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
