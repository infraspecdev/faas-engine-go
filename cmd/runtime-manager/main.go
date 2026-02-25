package main

import (
	"context"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/service"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {

	r := mux.NewRouter()
	_, cli, cancel, err := sdk.Init(context.Background())
	if err != nil {
		panic(err)
	}
	defer cancel()

	realDeployer := &service.Deployer{CLI: cli}
	invokeDeployer := &service.FunctionInvoker{}

	r.HandleFunc("/health", api.HealthHandler).Methods("GET")
	r.HandleFunc("/greet", api.GreetHandler).Methods("GET")

	// main api s
	r.HandleFunc("/functions", api.DeployHandler(realDeployer)).Methods("POST")
	r.HandleFunc("/functions/{functionName}/invoke", api.InvokeHandler(invokeDeployer)).Methods("POST")
	r.HandleFunc("/functions", api.GetFunctionsHandler).Methods("GET")
	r.HandleFunc("/functions/{functionName}", api.DeleteFunctionHandler).Methods("DELETE")

	http.ListenAndServe(":8080", r)
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Runtime Manager!")
}
