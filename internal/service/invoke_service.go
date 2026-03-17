package service

import (
	"context"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/db"
	"faas-engine-go/internal/sdk"
	"fmt"
	"log/slog"
	"time"

	"github.com/moby/moby/api/types/network"
)

type FunctionInvoker struct {
	containerClient sdk.ContainerClient
	imageClient     sdk.ImageClient
	invokeFunc      func(ctx context.Context, hostPort string, payload []byte) (map[string]any, error)
}

func NewFunctionInvoker(c sdk.ContainerClient, i sdk.ImageClient) *FunctionInvoker {
	return &FunctionInvoker{
		containerClient: c,
		imageClient:     i,
		invokeFunc:      sdk.InvokeContainer,
	}
}

// Invoke invokes a function by creating and starting a container from the function's image.
// It waits for the container to become healthy before sending the invocation request.
// The container is cleaned up asynchronously after invocation.
func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte) (any, error) {

	invID := db.CreateInvocation(functionName, payload, "http")
	start := time.Now()

	db.PrintContainerMap()
	container := db.GetFreeContainer(functionName)

	// ----------------------
	// REUSE CONTAINER
	// ----------------------
	if container != nil {

		logger := slog.With(
			"container_id", container.ID,
			"function", functionName,
		)

		logger.Info("container_lifecycle", "stage", "reusing")

		db.MarkBusy(container.ID)
		defer func() {
			db.MarkFree(container.ID)
			db.PrintContainerMap()
		}()

		res, err := f.invokeFunc(ctx, container.HostPort, payload)

		duration := time.Since(start)

		if err != nil {
			db.FailInvocation(invID, err.Error(), duration)
		} else {
			db.CompleteInvocation(invID, res, duration)
		}

		return res, err
	}

	// ----------------------
	// COLD START PATH
	// ----------------------

	target := config.ImageRef(config.FunctionsRepo, functionName, "")

	slog.Info("container_lifecycle", "stage", "pulling", "function", functionName)

	if err := f.imageClient.PullImage(ctx, target); err != nil {
		db.FailInvocation(invID, err.Error(), time.Since(start))
		slog.Error("image_pull_failed", "function", functionName, "error", err)
		return nil, err
	}

	containerId, err := f.containerClient.CreateContainer(ctx, functionName, target, nil)
	if err != nil {
		db.FailInvocation(invID, err.Error(), time.Since(start))
		slog.Error("container_create_failed", "function", functionName, "error", err)
		return nil, err
	}

	logger := slog.With("container_id", containerId, "function", functionName)

	logger.Info("container_lifecycle", "stage", "created")

	if err := f.containerClient.StartContainer(ctx, containerId); err != nil {
		db.FailInvocation(invID, err.Error(), time.Since(start))
		logger.Error("container_start_failed", "error", err)
		return nil, err
	}

	logger.Info("container_lifecycle", "stage", "starting")

	port, err := network.ParsePort(config.ContainerPort)
	if err != nil {
		db.FailInvocation(invID, err.Error(), time.Since(start))
		return nil, fmt.Errorf("failed to parse port: %w", err)
	}

	var hostPort string
	portDeadline := time.Now().Add(config.PortTimeout)

	for time.Now().Before(portDeadline) {
		inspect, err := f.containerClient.InspectContainer(ctx, containerId)

		if err == nil && inspect.Container.NetworkSettings != nil {
			bindings := inspect.Container.NetworkSettings.Ports[port]
			if len(bindings) > 0 {
				hostPort = bindings[0].HostPort
				break
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	healthDeadline := time.Now().Add(config.HealthTimeout)
	healthy := false

	for time.Now().Before(healthDeadline) {
		inspect, err := f.containerClient.InspectContainer(ctx, containerId)

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
		err := fmt.Errorf("container did not become healthy in time")
		db.FailInvocation(invID, err.Error(), time.Since(start))
		logger.Error("container_unhealthy", "timeout", config.HealthTimeout)
		return nil, err
	}

	logger.Info("container_lifecycle", "stage", "healthy")

	db.AddContainer(&db.Container{
		ID:           containerId,
		FunctionName: functionName,
		Status:       "busy",
		HostPort:     hostPort,
	})

	logger.Info("container_lifecycle", "stage", "invoking")

	res, err := f.invokeFunc(ctx, hostPort, payload)

	duration := time.Since(start)

	if err != nil {
		db.FailInvocation(invID, err.Error(), duration)
	} else {
		db.CompleteInvocation(invID, res, duration)
	}

	defer db.MarkFree(containerId)

	return res, err
}
