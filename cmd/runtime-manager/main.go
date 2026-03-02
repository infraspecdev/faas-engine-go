package main

import (
	"context"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/service"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		slog.Warn("could not load .env file, using default configuration")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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

	server := &http.Server{Addr: ":" + port, Handler: r}
	slog.Info("starting server", "port", port)

	if err = server.ListenAndServe(); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
