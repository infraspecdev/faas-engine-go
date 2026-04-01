package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sdk"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

func ContainerSpleen(containerClient sdk.ContainerClient, s core.Store) {

	ticker := time.NewTicker(10 * time.Second)

	go func() {

		for range ticker.C {

			containers, err := s.GetContainersByFunction("") // Get all containers
			if err != nil {
				slog.Error("failed to get containers", "error", err)
				continue
			}

			for _, container := range containers {
				if time.Since(container.LastUsedAt) > config.ContainerIdleTimeout {
					slog.Info(
						"container_lifecycle",
						"container_id", container.ID,
						"stage", "spleen_cleanup",
					)

					// Try to stop container with timeout
					// Don't block on stop failure - proceed to delete
					stopCtx, stopCancel := context.WithTimeout(
						context.Background(),
						config.ContainerCleanupStopTimeout,
					)
					if err := containerClient.StopContainer(stopCtx, container.ID); err != nil {
						slog.Error(
							"container_stop_failed",
							"container_id", container.ID,
							"error", err,
						)
					}
					stopCancel()

					// Try to delete container with timeout
					// If Docker delete fails, still remove from DB to prevent orphaning
					deleteCtx, deleteCancel := context.WithTimeout(
						context.Background(),
						config.ContainerCleanupDeleteTimeout,
					)
					nameErr := containerClient.DeleteContainer(deleteCtx, container.ID)
					deleteCancel()

					if nameErr != nil {
						slog.Warn(
							"container_delete_failed_proceeding_to_db_removal",
							"container_id", container.ID,
							"error", nameErr,
						)
						// Continue to db removal even if Docker delete failed
					}

					// Always try to remove from database
					if err := s.RemoveContainer(container.ID); err != nil {
						slog.Error("db_remove_failed", "container_id", container.ID, "error", err)
					} else {
						slog.Info(
							"container_lifecycle",
							"container_id", container.ID,
							"stage", "deleted",
						)
					}
				}
			}
		}
	}()
}

// GracefulShutdown stops all running containers and removes exited containers
// This should be called during application shutdown to ensure proper cleanup
// Only targets containers with the "faas-engine=true" label to protect external services
func GracefulShutdown(ctx context.Context, containerClient sdk.ContainerClient, s core.Store) error {
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

	// Phase 1: Stop all running containers in parallel
	phase1Start := time.Now()
	slog.Info("phase 1: stopping running containers", "count", running, "timeout_per_container", config.ShutdownContainerStopTime.String())

	var mu sync.Mutex
	const numStopWorkers = 4
	containersCh := make(chan string, len(containers))
	var wg sync.WaitGroup

	// Start stop workers
	for i := 0; i < numStopWorkers && i < running; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for containerID := range containersCh {
				stopCtx, stopCancel := context.WithTimeout(ctx, config.ShutdownContainerStopTime)
				if err := containerClient.StopContainer(stopCtx, containerID); err != nil {
					mu.Lock()
					stopFailed++
					mu.Unlock()
					slog.Warn("failed to stop container", "container_id", containerID[:12], "error", err)
				} else {
					mu.Lock()
					stopped++
					mu.Unlock()
					slog.Info("container stopped", "container_id", containerID[:12])
				}
				stopCancel()
			}
		}()
	}

	// Send running containers to workers
	for _, container := range containers {
		if container.State == "running" {
			containersCh <- container.ID
		}
	}
	close(containersCh)

	// Wait for all stop workers to complete
	wg.Wait()
	phase1Duration := time.Since(phase1Start)
	slog.Info("phase 1 completed", "stopped", stopped, "failed", stopFailed, "duration", phase1Duration.String())

	// Phase 2: Remove all exited/stopped containers
	phase2Start := time.Now()
	slog.Info("phase 2: removing stopped/exited containers", "timeout_per_container", config.ShutdownContainerDeleteTime.String())

	// Re-list to get updated states (but don't fail if re-list times out)
	containers, listErr := containerClient.ListContainers(ctx)
	if listErr != nil {
		slog.Warn("failed to re-list containers for removal (will attempt db cleanup anyway)", "error", listErr)
		// Don't return - try to clean up from database at least
	}

	removed := 0
	removeFailed := 0

	// Collect containers to remove (exited or stopped)
	var containersToRemove []string
	for _, container := range containers {
		if container.State == "exited" || container.State == "stopped" {
			containersToRemove = append(containersToRemove, container.ID)
		}
	}

	// Phase 2a: Delete containers from Docker in parallel
	// Track results separately from DB operations to avoid SQLite lock contention
	type deleteResult struct {
		containerID string
		dockerErr   error
	}
	deleteResultsCh := make(chan deleteResult, len(containersToRemove))

	const numRemoveWorkers = 4
	removeCh := make(chan string, len(containersToRemove))
	var removeWg sync.WaitGroup

	// Start Docker delete workers
	for i := 0; i < numRemoveWorkers && i < len(containersToRemove); i++ {
		removeWg.Add(1)
		go func() {
			defer removeWg.Done()
			for containerID := range removeCh {
				deleteCtx, deleteCancel := context.WithTimeout(ctx, config.ShutdownContainerDeleteTime)
				nameErr := containerClient.DeleteContainer(deleteCtx, containerID)
				deleteCancel()

				deleteResultsCh <- deleteResult{containerID, nameErr}
			}
		}()
	}

	// Send containers to delete workers
	for _, containerID := range containersToRemove {
		removeCh <- containerID
	}
	close(removeCh)

	// Wait for all Docker delete workers to complete
	removeWg.Wait()
	close(deleteResultsCh)

	// Collect results from Docker delete operations
	dockerResults := make([]deleteResult, 0, len(containersToRemove))
	for result := range deleteResultsCh {
		if result.dockerErr != nil {
			removeFailed++
			slog.Warn("failed to delete container from Docker", "container_id", result.containerID[:12], "error", result.dockerErr)
		}
		dockerResults = append(dockerResults, result)
	}

	// Phase 2b: Remove all containers from database serially to avoid SQLite lock contention
	// This happens after all Docker operations are complete
	for _, result := range dockerResults {
		if err := s.RemoveContainer(result.containerID); err != nil {
			slog.Warn("failed to remove container from db", "container_id", result.containerID[:12], "error", err)
		} else {
			removed++
			slog.Info("container removed", "container_id", result.containerID[:12])
		}
	}

	// Phase 2c: Clean up orphaned database entries (containers that are still in DB but don't exist in Docker)
	// This also happens serially after Docker operations are complete
	dbContainers, err := s.GetContainersByFunction("")
	if err == nil {
		for _, dbContainer := range dbContainers {
			// Check if this container exists in Docker list
			found := false
			for _, dockerContainer := range containers {
				if dockerContainer.ID == dbContainer.ID {
					found = true
					break
				}
			}

			// If not found in Docker, it's orphaned - remove from DB
			if !found {
				if err := s.RemoveContainer(dbContainer.ID); err != nil {
					slog.Warn("failed to remove orphaned container from db", "container_id", dbContainer.ID, "error", err)
				} else {
					removed++
					slog.Info("orphaned container removed from db", "container_id", dbContainer.ID)
				}
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
