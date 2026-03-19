package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	sqlstore "faas-engine-go/internal/sqlite/store"

	"github.com/moby/moby/api/types/network"
)

type Store interface {
	GetActiveFunction(name string) (*models.Function, error)
	CreateInvocation(inv *models.Invocation) error
	MarkInvocationRunning(invID string, containerID string) error
	CompleteInvocation(
		invID string,
		status string,
		exitCode int,
		responsePayload []byte,
		logs string,
		startedAt time.Time,
	) error
	AcquireFreeContainer(functionID int) (*models.Container, error)
	MarkContainerFree(containerID string) error
	RemoveContainer(containerID string) error
	CreateContainer(c *models.Container) error
}

type realStore struct {
	db *sql.DB
}

var _ Store = (*realStore)(nil)

func NewStore() Store {
	return &realStore{db: sqlite.DB}
}

func (s *realStore) GetActiveFunction(name string) (*models.Function, error) {
	return sqlstore.GetActiveFunction(s.db, name)
}

func (s *realStore) CreateInvocation(inv *models.Invocation) error {
	return sqlstore.CreateInvocation(s.db, inv)
}

func (s *realStore) MarkInvocationRunning(invID string, containerID string) error {
	return sqlstore.MarkInvocationRunning(s.db, invID, containerID)
}

func (s *realStore) CompleteInvocation(
	invID string,
	status string,
	exitCode int,
	responsePayload []byte,
	logs string,
	startedAt time.Time,
) error {
	return sqlstore.CompleteInvocation(
		s.db,
		invID,
		status,
		exitCode,
		responsePayload,
		logs,
		startedAt,
	)
}

func (s *realStore) AcquireFreeContainer(functionID int) (*models.Container, error) {
	return sqlstore.AcquireFreeContainer(s.db, functionID)
}

func (s *realStore) MarkContainerFree(containerID string) error {
	return sqlstore.MarkContainerFree(s.db, containerID)
}

func (s *realStore) RemoveContainer(containerID string) error {
	return sqlstore.RemoveContainer(s.db, containerID)
}

func (s *realStore) CreateContainer(c *models.Container) error {
	return sqlstore.CreateContainer(s.db, c)
}

//
// 🔥 FUNCTION INVOKER
//

type FunctionInvoker struct {
	containerClient sdk.ContainerClient
	imageClient     sdk.ImageClient
	store           Store
	invokeFunc      func(ctx context.Context, hostPort string, payload []byte) (map[string]any, error)
}

func NewFunctionInvoker(c sdk.ContainerClient, i sdk.ImageClient, s Store) *FunctionInvoker {
	return &FunctionInvoker{
		containerClient: c,
		imageClient:     i,
		store:           s,
		invokeFunc:      sdk.InvokeContainer,
	}
}

func (f *FunctionInvoker) Invoke(ctx context.Context, functionName string, payload []byte) (any, error) {

	fn, err := f.store.GetActiveFunction(functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch function: %w", err)
	}
	if fn == nil {
		return nil, fmt.Errorf("function not found")
	}

	inv := &models.Invocation{
		FunctionID:     fn.ID,
		TriggerType:    "http",
		Status:         "pending",
		RequestPayload: payload,
		StartedAt:      time.Now(),
	}

	if err := f.store.CreateInvocation(inv); err != nil {
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

	container, err := f.store.AcquireFreeContainer(fn.ID)
	if err != nil {
		return nil, false, err
	}

	if container == nil {
		slog.Warn("no free container found", "function", fn.Name)
		return nil, false, nil
	}

	if err := f.store.MarkInvocationRunning(inv.ID, container.ID); err != nil {
		slog.Error("mark invocation running failed", "error", err)
	}

	res, err := f.invokeFunc(ctx, container.HostPort, payload)
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

	if err := f.store.MarkContainerFree(container.ID); err != nil {
		slog.Error("mark container free failed", "error", err)
	}

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

	hostPort, err := f.waitForPort(ctx, containerID)
	if err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		return nil, err
	}

	if err := f.waitForHealthy(ctx, containerID); err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		return nil, err
	}

	if err := f.store.MarkInvocationRunning(inv.ID, containerID); err != nil {
		return nil, err
	}

	res, err := f.invokeFunc(ctx, hostPort, payload)
	if err != nil {
		_ = f.containerClient.DeleteContainer(ctx, containerID)
		_ = f.store.RemoveContainer(containerID)

		f.completeInvocation(inv, containerID, nil, err)
		return nil, err
	}

	if err := f.store.CreateContainer(&models.Container{
		ID:         containerID,
		FunctionID: fn.ID,
		Status:     "free",
		HostPort:   hostPort,
		LastUsedAt: time.Now(),
		CreatedAt:  time.Now(),
	}); err != nil {
		slog.Warn("failed to persist container", "error", err)
	}

	f.completeInvocation(inv, containerID, res, nil)
	return res, nil
}

func (f *FunctionInvoker) createAndStart(ctx context.Context, name, image string) (string, error) {

	containerID, err := f.containerClient.CreateContainer(ctx, name, image, nil)
	if err != nil {
		return "", fmt.Errorf("create container failed: %w", err)
	}

	if err := f.containerClient.StartContainer(ctx, containerID); err != nil {
		return "", fmt.Errorf("start container failed: %w", err)
	}

	return containerID, nil
}

func (f *FunctionInvoker) waitForPort(ctx context.Context, containerID string) (string, error) {

	port, err := network.ParsePort(config.ContainerPort)
	if err != nil {
		return "", fmt.Errorf("parse port failed: %w", err)
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
				return fmt.Errorf("container exited early")
			}
			if inspect.Container.State.Health != nil &&
				inspect.Container.State.Health.Status == "healthy" {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	return fmt.Errorf("container not healthy in time")
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var logs string
	if containerID != "" {
		if l, logErr := f.containerClient.LogContainer(ctx, containerID); logErr == nil {
			logs = l
		} else {
			slog.Warn("failed to fetch logs", "error", logErr)
		}
	}

	if err := f.store.CompleteInvocation(
		inv.ID,
		status,
		exitCode,
		responsePayload,
		logs,
		inv.StartedAt,
	); err != nil {
		slog.Error("complete invocation failed", "error", err)
	}
}
