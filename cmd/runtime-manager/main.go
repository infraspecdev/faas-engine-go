package main

import (
	"context"
	"faas-engine-go/internal/api"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/service"
	"faas-engine-go/internal/sqlite"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
	ctx, cli, cancel, err := sdk.Init(context.Background())
	if err != nil {
		slog.Error("failed to initialize sdk", "error", err)
		os.Exit(1)
	}
	defer cancel()

	docker := sdk.NewDockerClient(cli, ctx)

	// Initialize database (if needed)
	db, err := sqlite.InitDB()
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	if err := sqlite.InitTables(); err != nil {
		slog.Error("failed to initialize database tables", "error", err)
		os.Exit(1)
	}

	// Start background container cleanup worker
	store := service.NewStore(db)
	spleenStopCh := service.ContainerSpleen(ctx, docker, store)

	// Setup router
	r := mux.NewRouter()

	deployService := service.NewDeployService(docker, db)
	functionStore := api.NewFunctionStore(db)

	invokeService := service.NewInvokeService(docker, docker, store)

	scheduler := service.NewSchedulerService(invokeService)

	if err := scheduler.LoadSchedules(); err != nil {
		slog.Error("failed to load schedules", "error", err)
		os.Exit(1)
	}

	scheduler.Start()

	logService := service.NewLogService(db)
	functionVersionService := service.NewFunctionVersionService(store)

	logStreamService := service.NewLogStreamService(docker, store)

	registryClient := &service.HTTPRegistryClient{}
	deleteService := service.NewFunctionDeleteService(store, registryClient)

	listService := service.NewListService(store)

	r.HandleFunc("/functions", api.ListFunctionsHandler(listService)).Methods("GET")
	r.HandleFunc("/functions/{functionName}/logs", api.LogHandler(logService)).Methods("GET")
	r.HandleFunc("/functions/{functionName}/logs/stream", api.LogStreamHandler(logStreamService)).Methods("GET")
	r.HandleFunc("/functions/{functionName}/versions", api.FunctionVersionsHandler(functionVersionService)).Methods("GET")

	r.HandleFunc("/functions", api.DeployHandler(deployService, functionStore)).Methods("POST")
	r.HandleFunc("/functions/{functionName}/invoke", api.InvokeHandler(invokeService)).Methods("POST")

	r.HandleFunc("/functions/{functionName}", api.DeleteFunctionHandler(deleteService)).Methods("DELETE")

	r.HandleFunc("/schedules/{functionName}", api.CreateScheduleHandler(scheduler)).Methods("POST")
	r.HandleFunc("/schedules", api.ListSchedulesHandler()).Methods("GET")
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
	slog.Info("shutdown signal received, initiating graceful shutdown")

	// Phase 1: Stop spleen cleanup worker
	close(spleenStopCh)
	slog.Info("container_spleen stopped")

	// Phase 2: Stop scheduler (prevents new triggers)
	scheduler.Stop()
	slog.Info("scheduler stopped")

	// Phase 3: Shutdown HTTP server (waits for in-flight requests)
	ctx, shutdownCancel := context.WithTimeout(context.Background(), config.ServerShutdownTimeout)
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("http server exited gracefully")
	}
	shutdownCancel()

	// Phase 4: Graceful container cleanup (stop all running containers and remove exited ones)
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), config.GracefulShutdownTimeout)
	defer cleanupCancel()

	if err := service.GracefulShutdown(cleanupCtx, docker, store); err != nil {
		slog.Error("container cleanup failed", "error", err)
	}

	slog.Info("graceful shutdown completed")
}
