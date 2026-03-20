package service

import (
	"context"
	"encoding/json"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	"faas-engine-go/internal/sqlite/store"
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

func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte, triggerType string) (any, error) {

	fn, err := store.GetActiveFunction(sqlite.DB, functionName)
	if err != nil || fn == nil {
		return nil, fmt.Errorf("function not found")
	}

	inv := &models.Invocation{
		FunctionID:     fn.ID,
		TriggerType:    triggerType,
		Status:         "pending",
		RequestPayload: payload,
		StartedAt:      time.Now(),
	}

	if err := store.CreateInvocation(sqlite.DB, inv); err != nil {
		return nil, err
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

	container, err := store.GetFreeContainer(sqlite.DB, fn.ID)
	if err != nil {
		return nil, false, err
	}
	if container == nil {
		return nil, false, nil
	}

	inspect, err := f.containerClient.InspectContainer(ctx, container.ID)
	if err != nil || inspect.Container.State == nil || !inspect.Container.State.Running {
		slog.Warn("dead_container_detected", "id", container.ID)

		_ = f.containerClient.DeleteContainer(ctx, container.ID)
		_ = store.RemoveContainer(sqlite.DB, container.ID)

		return nil, false, nil
	}

	logger := slog.With("container_id", container.ID, "function", fn.Name)
	logger.Info("container_lifecycle", "stage", "reusing")

	store.MarkContainerBusy(sqlite.DB, container.ID)
	store.MarkInvocationRunning(sqlite.DB, inv.ID, container.ID)

	defer store.MarkContainerFree(sqlite.DB, container.ID)

	res, err := f.invokeFunc(ctx, container.HostPort, payload)

	f.completeInvocation(inv, container.ID, res, err)

	return res, true, err
}

func (f *FunctionInvoker) coldStartInvokeWithInvocation(
	ctx context.Context,
	fn *models.Function,
	payload []byte,
	inv *models.Invocation,
) (any, error) {

	image := config.ImageRef(config.FunctionsRepo, fn.Name, fn.Version)

	slog.Info("container_lifecycle", "stage", "pulling", "function", fn.Name)

	if err := f.imageClient.PullImage(ctx, image); err != nil {
		return nil, err
	}

	containerID, err := f.createAndStart(ctx, fn.Name, image)
	if err != nil {
		return nil, err
	}

	inspect, err := f.containerClient.InspectContainer(ctx, containerID)
	if err != nil || inspect.Container.State == nil || !inspect.Container.State.Running {
		slog.Error("container_exited_early", "id", containerID)

		_ = f.containerClient.DeleteContainer(ctx, containerID)
		return nil, fmt.Errorf("container exited early")
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

	store.MarkInvocationRunning(sqlite.DB, inv.ID, containerID)

	res, err := f.invokeFunc(ctx, hostPort, payload)

	logger.Info("container_lifecycle", "stage", "invoking")

	if err == nil {
		f.persistContainer(fn.ID, containerID, hostPort)
	}

	if err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
	}

	defer store.MarkContainerFree(sqlite.DB, containerID)

	f.completeInvocation(inv, containerID, res, err)

	return res, err
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
		responsePayload, _ = json.Marshal(res)
	}

	if err != nil {
		status = "failed"
		exitCode = 1
	} else {
		status = "success"
		exitCode = 0
	}

	logs := ""

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if containerID != "" {
		l, logErr := f.containerClient.LogContainer(ctx, containerID)
		if logErr != nil {
			slog.Warn("log_fetch_failed", "container_id", containerID, "error", logErr)
		} else {
			logs = l

			if len(logs) > 5000 {
				logs = logs[:5000] + "...(truncated)"
			}
		}
	}

	store.CompleteInvocation(
		sqlite.DB,
		inv.ID,
		status,
		exitCode,
		responsePayload,
		logs,
		inv.StartedAt,
	)
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

func (f *FunctionInvoker) persistContainer(fnID int, containerID, hostPort string) {

	store.CreateContainer(sqlite.DB, &models.Container{
		ID:         containerID,
		FunctionID: fnID,
		Status:     "busy",
		HostPort:   hostPort,
		LastUsedAt: time.Now(),
		CreatedAt:  time.Now(),
	})
}
