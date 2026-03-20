package main

import (
	"context"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/service"
	"faas-engine-go/internal/sqlite"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env (optional for local dev)
	if err := godotenv.Load(); err != nil {
		slog.Warn("could not load .env file, using default configuration")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize SDK
	_, cli, cancel, err := sdk.Init(context.Background())
	if err != nil {
		slog.Error("failed to initialize sdk", "error", err)
		os.Exit(1)
	}
	defer cancel()

	docker := sdk.NewDockerClient(cli)

	// Start background container cleanup worker
	service.ContainerSpleen(docker)

	// Initialize database (if needed)
	if err := sqlite.InitDB(); err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	if err := sqlite.InitTables(); err != nil {
		slog.Error("failed to initialize database tables", "error", err)
		os.Exit(1)
	}

	// Setup router
	r := mux.NewRouter()

	realDeployer := service.NewDeployer(docker)
	realStore := api.NewFunctionStore()

	invokeInvoker := service.NewFunctionInvoker(docker, docker)

	scheduler := service.NewSchedulerService(invokeInvoker)

	if err := scheduler.LoadSchedules(); err != nil {
		slog.Error("failed to load schedules", "error", err)
		os.Exit(1)
	}

	scheduler.Start()

	r.HandleFunc("/health", api.HealthHandler).Methods("GET")
	r.HandleFunc("/greet", api.GreetHandler).Methods("GET")
	r.HandleFunc("/functions", api.DeployHandler(realDeployer, realStore)).Methods("POST")
	r.HandleFunc("/functions/{functionName}/invoke", api.InvokeHandler(invokeInvoker)).Methods("POST")
	r.HandleFunc("/functions", api.GetFunctionsHandler).Methods("GET")
	r.HandleFunc("/functions/{functionName}", api.DeleteFunctionHandler).Methods("DELETE")

	r.HandleFunc("/schedules/{functionName}", api.CreateScheduleHandler(scheduler)).Methods("POST")
	r.HandleFunc("/schedules", api.ListSchedulesHandler()).Methods("GET")
	r.HandleFunc("/schedules/{functionName}", api.ListScheduleByFunctionNameHandler(scheduler)).Methods("GET")
	r.HandleFunc("/schedules/{id}", api.DeleteScheduleHandler(scheduler)).Methods("DELETE")
	// Create server instance
	srv := &http.Server{
		Addr:    ":" + port, // ":"  = 0.0.0.0
		Handler: r,
	}

	// Run server in background
	go func() {
		slog.Info("starting server", "port", port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("shutdown signal received")

	scheduler.Stop()
	// Create timeout context for graceful shutdown
	ctx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server exited gracefully")
	}
}
