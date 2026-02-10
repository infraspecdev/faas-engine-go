package main

import (
	"faas-engine-go/cmd/runtime-manager/api"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/health", api.HealthHandler).Methods("GET")
	r.HandleFunc("/greet", api.GreetHandler).Methods("GET")

	// main api s
	r.HandleFunc("/functions", api.DeployHandler).Methods("POST")
	r.HandleFunc("/functions/{function_name}/invoke", api.InvokeHandler).Methods("POST")
	r.HandleFunc("/functions", api.GetFunctionsHandler).Methods("GET")
	r.HandleFunc("/functions/{function_name}", api.DeleteFunctionHandler).Methods("DELETE")

	http.ListenAndServe(":8080", r)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Runtime Manager!")
}
