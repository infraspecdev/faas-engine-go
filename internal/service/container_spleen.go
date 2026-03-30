package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/store"
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
