package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"
	"fmt"
	"log/slog"
	"time"
)

func ContainerSpleen(containerClient sdk.ContainerClient) {

	ticker := time.NewTicker(10 * time.Second)

	go func() {

		for range ticker.C {

			store.CleanupIdleContainers(
				config.ContainerIdleTimeout,
				func(containerID string) {

					slog.Info(
						"container_lifecycle",
						"container_id", containerID,
						"stage", "spleen_cleanup",
					)

					// Create separate contexts for stop and delete with timeouts from config
					// Both of these Docker operations need time when containers are still active
					stopCtx, stopCancel := context.WithTimeout(
						context.Background(),
						config.ContainerCleanupStopTimeout,
					)
					if err := containerClient.StopContainer(stopCtx, containerID); err != nil {
						slog.Error(
							"container_stop_failed",
							"container_id", containerID,
							"error", err,
						)
					}
					stopCancel()

					// Delete in separate context with own timeout
					deleteCtx, deleteCancel := context.WithTimeout(
						context.Background(),
						config.ContainerCleanupDeleteTimeout,
					)
					defer deleteCancel()

					if err := containerClient.DeleteContainer(deleteCtx, containerID); err != nil {
						slog.Error(
							"container_delete_failed",
							"container_id", containerID,
							"error", err,
						)
						return
					}

					if err := store.RemoveContainer(sqlite.DB, containerID); err != nil {
						slog.Error(
							"db_remove_failed",
							"container_id", containerID,
							"error", err,
						)
					}

					slog.Info(
						"container_lifecycle",
						"container_id", containerID,
						"stage", "deleted",
					)
				},
			)
		}
	}()
}

// GracefulShutdown stops all running containers and removes exited containers
// This should be called during application shutdown to ensure proper cleanup
// Only targets containers with the "faas-engine=true" label to protect external services
func GracefulShutdown(ctx context.Context, containerClient sdk.ContainerClient) error {
	startTime := time.Now()
	slog.Info("graceful shutdown started", "total_timeout", config.GracefulShutdownTimeout.String())

	// List all faas-engine labeled containers
	containers, err := containerClient.ListContainers(ctx)
	if err != nil {
		slog.Error("failed to list containers during shutdown", "error", err)
		return err
	}

	if len(containers) == 0 {
		slog.Info("no faas-engine containers to shutdown")
		return nil
	}

	// Count containers by state
	running := 0
	exited := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		} else {
			exited++
		}
	}
	slog.Info("container cleanup scope", "total_labeled", len(containers), "running", running, "stopped_or_exited", exited)

	stopped := 0
	stopFailed := 0

	// Phase 1: Stop all running containers
	phase1Start := time.Now()
	slog.Info("phase 1: stopping running containers", "count", running, "timeout_per_container", config.ShutdownContainerStopTime.String())
	for _, container := range containers {
		containerID := container.ID[:12] // Short ID for logging

		if container.State == "running" {
			stopCtx, stopCancel := context.WithTimeout(ctx, config.ShutdownContainerStopTime)
			if err := containerClient.StopContainer(stopCtx, container.ID); err != nil {
				stopFailed++
				slog.Warn("failed to stop container", "container_id", containerID, "error", err)
			} else {
				stopped++
				slog.Info("container stopped", "container_id", containerID)
			}
			stopCancel()
		}
	}
	phase1Duration := time.Since(phase1Start)
	slog.Info("phase 1 completed", "stopped", stopped, "failed", stopFailed, "duration", phase1Duration.String())

	// Phase 2: Remove all exited/stopped containers
	phase2Start := time.Now()
	slog.Info("phase 2: removing stopped/exited containers", "timeout_per_container", config.ShutdownContainerDeleteTime.String())

	// Re-list to get updated states
	containers, err = containerClient.ListContainers(ctx)
	if err != nil {
		slog.Error("failed to re-list containers for removal", "error", err)
		return err
	}

	removed := 0
	removeFailed := 0
	for _, container := range containers {
		if container.State == "exited" || container.State == "stopped" {
			deleteCtx, deleteCancel := context.WithTimeout(ctx, config.ShutdownContainerDeleteTime)
			if err := containerClient.DeleteContainer(deleteCtx, container.ID); err != nil {
				removeFailed++
				slog.Warn("failed to delete container", "container_id", container.ID, "state", container.State, "error", err)
				deleteCancel()
				continue
			}
			deleteCancel()

			// Remove from database after successful Docker deletion
			if err := store.RemoveContainer(sqlite.DB, container.ID); err != nil {
				slog.Warn("failed to remove container from db", "container_id", container.ID, "error", err)
			} else {
				removed++
				slog.Info("container removed", "container_id", container.ID, "state", container.State)
			}
		}
	}
	phase2Duration := time.Since(phase2Start)
	slog.Info("phase 2 completed", "removed", removed, "failed", removeFailed, "duration", phase2Duration.String())

	totalDuration := time.Since(startTime)
	slog.Info("graceful shutdown completed", "total_duration", totalDuration.String(), "containers_stopped", stopped, "containers_removed", removed, "stop_failures", stopFailed, "remove_failures", removeFailed)

	if stopFailed > 0 || removeFailed > 0 {
		slog.Warn("shutdown completed with errors", "stop_errors", stopFailed, "remove_errors", removeFailed)
		return fmt.Errorf("shutdown had failures: %d stop errors, %d remove errors", stopFailed, removeFailed)
	}

	return nil
}
