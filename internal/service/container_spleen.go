package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/db"
	"faas-engine-go/internal/sdk"
	"log/slog"
	"time"
)

func ContainerSpleen(containerClient sdk.ContainerClient) {

	ticker := time.NewTicker(10 * time.Second)

	go func() {

		for range ticker.C {

			db.CleanupIdleContainers(
				config.ContainerIdleTimeout,
				func(containerID string) {

					ctx, cancel := context.WithTimeout(
						context.Background(),
						config.CleanUpTimeout,
					)
					defer cancel()

					slog.Info(
						"container_lifecycle",
						"stage", "spleen_cleanup",
						"container_id", containerID,
					)

					if err := containerClient.StopContainer(ctx, containerID); err != nil {
						slog.Error("container_stop_failed",
							"container_id", containerID,
							"error", err,
						)
					}

					if err := containerClient.DeleteContainer(ctx, containerID); err != nil {
						slog.Error("container_delete_failed",
							"container_id", containerID,
							"error", err,
						)
					}

					db.RemoveContainer(containerID)
				},
			)

		}

	}()
}
