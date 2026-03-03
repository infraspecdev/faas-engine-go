package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"fmt"
	"log/slog"
	"time"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type FunctionInvoker struct{}

// Invoke invokes a function by creating and starting a container from the function's image.
// It waits for the container to become healthy before sending the invocation request.
// The container is cleaned up asynchronously after invocation.
func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte) (any, error) {

	ctx, cli, cancel, err := sdk.Init(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()

	target := config.ImageRef(config.FunctionsRepo, functionName, "")

	slog.Info("container_lifecycle", "stage", "pulling", "function", functionName)

	if err := sdk.PullImage(ctx, cli, target); err != nil {
		slog.Error("image_pull_failed", "function", functionName, "error", err)
		return nil, err
	}

	containerId, err := sdk.CreateContainer(ctx, cli, functionName, target, nil)
	if err != nil {
		slog.Error("container_create_failed", "function", functionName, "error", err)
		return nil, err
	}

	logger := slog.With("container_id", containerId, "function", functionName)

	logger.Info("container_lifecycle", "stage", "created")

	defer func() {
		go func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), config.CleanUpTimeout)
			defer cancel()

			logger.Info("container_lifecycle", "stage", "stopping")

			if err := sdk.StopContainer(cleanupCtx, cli, containerId); err != nil {
				logger.Error("container_stop_failed", "error", err)
			} else {
				logger.Info("container_lifecycle", "stage", "stopped")
			}
		}()
	}()

	if err := sdk.StartContainer(ctx, cli, containerId); err != nil {
		logger.Error("container_start_failed", "error", err)
		return nil, err
	}

	logger.Info("container_lifecycle", "stage", "starting")

	// Wait for port binding
	port, err := network.ParsePort(config.ContainerPort)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port: %w", err)
	}

	var hostPort string
	portDeadline := time.Now().Add(config.PortTimeout)

	for time.Now().Before(portDeadline) {
		inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{})

		if err == nil && inspect.Container.NetworkSettings != nil {
			bindings := inspect.Container.NetworkSettings.Ports[port]
			if len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Wait for healthy
	healthDeadline := time.Now().Add(config.HealthTimeout)
	healthy := false

	for time.Now().Before(healthDeadline) {
		inspect, err := cli.ContainerInspect(ctx, containerId, client.ContainerInspectOptions{})

		if err == nil &&
			inspect.Container.State != nil &&
			inspect.Container.State.Health != nil &&
			inspect.Container.State.Health.Status == "healthy" {

			healthy = true
			break
		}

		time.Sleep(300 * time.Millisecond)
	}

	if !healthy {
		logger.Error("container_unhealthy",
			"timeout", config.HealthTimeout,
		)
		return nil, fmt.Errorf("container did not become healthy in time")
	}

	logger.Info("container_lifecycle", "stage", "healthy")

	logger.Info("container_lifecycle", "stage", "invoking")

	return sdk.InvokeContainer(ctx, hostPort, payload)
}
