package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"

	"github.com/moby/moby/api/types/network"
)

type FunctionInvoker struct {
	containerClient sdk.ContainerClient
	imageClient     sdk.ImageClient
	store           core.Store
}

func NewInvokeService(c sdk.ContainerClient, i sdk.ImageClient, s core.Store) *FunctionInvoker {
	return &FunctionInvoker{
		containerClient: c,
		imageClient:     i,
		store:           s,
	}
}

// retryWithBackoff is a wrapper around the unified store retry logic
// Uses: 50ms, 100ms, 200ms, 400ms, 800ms backoff with 5 max retries
func retryWithBackoff(operation func() error) error {
	return store.RetryWithBackoff(operation, 50*time.Millisecond, 5)
}

func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte, triggerType string) (any, error) {

	fn, err := f.store.GetActiveFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch function: %w", err)
	}
	if fn == nil {
		return nil, ErrFunctionNotFound
	}

	inv := &models.Invocation{
		FunctionID:     fn.ID,
		TriggerType:    triggerType,
		Status:         "pending",
		RequestPayload: payload,
		StartedAt:      time.Now(),
	}

	// Retry create invocation with backoff to handle transient locks
	if err := retryWithBackoff(func() error {
		return f.store.CreateInvocation(inv)
	}); err != nil {
		return nil, fmt.Errorf("failed to create invocation: %w", err)
	}

	if res, ok, err := f.tryReuseWithInvocation(ctx, fn, payload, inv); ok {
		return res, err
	}

	return f.coldStartInvokeWithInvocation(ctx, fn, payload, inv)
}

func (f *FunctionInvoker) tryReuseWithInvocation(
	ctx context.Context,
	fn *models.Function,
	payload []byte,
	inv *models.Invocation,
) (any, bool, error) {

	var container *models.Container
	err := retryWithBackoff(func() error {
		var acquireErr error
		container, acquireErr = f.store.AcquireFreeContainer(fn.ID)
		return acquireErr
	})
	if err != nil {
		return nil, false, err
	}

	if container == nil {
		slog.Warn("no free container found", "function", fn.Name)
		return nil, false, nil
	}

	// Retry marking invocation as running
	if err := retryWithBackoff(func() error {
		return f.store.MarkInvocationRunning(inv.ID, container.ID)
	}); err != nil {
		slog.Error("mark invocation running failed", "error", err)
	}

	slog.Info(
		"container_lifecycle",
		"container_id", container.ID,
		"function", fn.Name,
		"stage", "reusing",
	)

	res, err := f.containerClient.InvokeContainer(ctx, container.HostPort, payload)
	if err != nil {
		if delErr := f.containerClient.DeleteContainer(ctx, container.ID); delErr != nil {
			slog.Warn("failed to delete container", "error", delErr)
		}
		if rmErr := f.store.RemoveContainer(container.ID); rmErr != nil {
			slog.Warn("failed to remove container from db", "error", rmErr)
		}

		f.completeInvocation(inv, container.ID, nil, err)
		return nil, false, nil
	}

	// completeInvocation handles MarkContainerFree, don't call it here
	f.completeInvocation(inv, container.ID, res, nil)
	return res, true, nil
}

func (f *FunctionInvoker) coldStartInvokeWithInvocation(
	ctx context.Context,
	fn *models.Function,
	payload []byte,
	inv *models.Invocation,
) (any, error) {

	image := config.ImageRef(config.FunctionsRepo, fn.Name, fn.Version)

	if err := f.imageClient.PullImage(ctx, image); err != nil {
		return nil, fmt.Errorf("pull image failed: %w", err)
	}

	containerID, err := f.createAndStart(ctx, fn.Name, image)
	if err != nil {
		return nil, err
	}

	logger := slog.With("container_id", containerID, "function", fn.Name)
	logger.Info("container_lifecycle", "stage", "created")

	hostPort, err := f.waitForPort(ctx, containerID)
	if err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		return nil, err
	}

	if err := f.waitForHealthy(ctx, containerID); err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		return nil, err
	}

	logger.Info("container_lifecycle", "stage", "healthy")

	if err := f.store.CreateContainer(&models.Container{
		ID:         containerID,
		FunctionID: fn.ID,
		Status:     "busy",
		HostPort:   hostPort,
		LastUsedAt: time.Now(),
		CreatedAt:  time.Now(),
	}); err != nil {
		slog.Warn("failed to persist container", "error", err)
	}

	if err := f.store.MarkInvocationRunning(inv.ID, containerID); err != nil {
		return nil, err
	}

	res, err := f.containerClient.InvokeContainer(ctx, hostPort, payload)
	if err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		_ = f.store.RemoveContainer(containerID)

		f.completeInvocation(inv, containerID, nil, err)
		return nil, err
	}

	logger.Info("container_lifecycle", "stage", "invoking")

	// completeInvocation handles MarkContainerFree, don't call it here
	f.completeInvocation(inv, containerID, res, nil)

	return res, nil
}

func (f *FunctionInvoker) completeInvocation(
	inv *models.Invocation,
	containerID string,
	res map[string]any,
	err error,
) {

	var status string
	var exitCode int
	var responsePayload []byte

	if res != nil {
		if b, marshalErr := json.Marshal(res); marshalErr == nil {
			responsePayload = b
		}
	}

	if err != nil {
		status = "failed"
		exitCode = 1
	} else {
		status = "success"
		exitCode = 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.LogRetrievalTimeout)
	defer cancel()

	var logs string
	if containerID != "" {
		if l, logErr := f.containerClient.LogContainer(ctx, containerID); logErr == nil {
			logs = l
		} else {
			slog.Warn("failed to fetch logs", "error", logErr)
		}
	}

	// Complete invocation and mark container free in atomic operation with retries
	// This consolidates two write operations into one transaction
	shouldMarkFree := containerID != "" && err == nil
	if completeErr := retryWithBackoff(func() error {
		return f.store.CompleteInvocationAndMarkFree(
			inv.ID,
			status,
			exitCode,
			responsePayload,
			logs,
			inv.StartedAt,
			containerID,
			shouldMarkFree,
		)
	}); completeErr != nil {
		slog.Error("complete invocation failed", "error", completeErr, "container_id", containerID)
	}
}

func (f *FunctionInvoker) createAndStart(ctx context.Context, name, image string) (string, error) {

	containerID, err := f.containerClient.CreateContainer(ctx, name, image, nil)
	if err != nil {
		slog.Error("container_create_failed", "function", name, "error", err)
		return "", err
	}

	if err := f.containerClient.StartContainer(ctx, containerID); err != nil {
		slog.Error("container_start_failed", "container_id", containerID, "error", err)
		return "", err
	}

	slog.Info("container_lifecycle", "stage", "starting", "container_id", containerID)

	return containerID, nil
}

func (f *FunctionInvoker) waitForPort(ctx context.Context, containerID string) (string, error) {

	port, err := network.ParsePort(config.ContainerPort)
	if err != nil {
		return "", fmt.Errorf("failed to parse port: %w", err)
	}

	deadline := time.Now().Add(config.PortTimeout)

	for time.Now().Before(deadline) {

		inspect, err := f.containerClient.InspectContainer(ctx, containerID)

		if err == nil && inspect.Container.NetworkSettings != nil {
			bindings := inspect.Container.NetworkSettings.Ports[port]
			if len(bindings) > 0 {
				return bindings[0].HostPort, nil
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return "", fmt.Errorf("port not available in time")
}

func (f *FunctionInvoker) waitForHealthy(ctx context.Context, containerID string) error {

	deadline := time.Now().Add(config.HealthTimeout)

	for time.Now().Before(deadline) {

		inspect, err := f.containerClient.InspectContainer(ctx, containerID)

		if err == nil && inspect.Container.State != nil {

			if !inspect.Container.State.Running {
				return fmt.Errorf("container exited before becoming healthy")
			}

			if inspect.Container.State.Health != nil &&
				inspect.Container.State.Health.Status == "healthy" {
				return nil
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	return fmt.Errorf("container did not become healthy in time")
}
