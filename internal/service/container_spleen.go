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

					ctx, cancel := context.WithTimeout(
						context.Background(),
						config.CleanUpTimeout,
					)
					defer cancel()

					slog.Info(
						"container_lifecycle",
						"container_id", containerID,
						"stage", "spleen_cleanup",
					)

					if err := containerClient.StopContainer(ctx, containerID); err != nil {
						slog.Error("container_stop_failed", "container_id", containerID, "error", err)
					}

					if err := containerClient.DeleteContainer(ctx, containerID); err != nil {
						slog.Error("container_delete_failed", "container_id", containerID, "error", err)
						return
					}

					if err := store.RemoveContainer(sqlite.DB, containerID); err != nil {
						slog.Error("db_remove_failed", "container_id", containerID, "error", err)
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
