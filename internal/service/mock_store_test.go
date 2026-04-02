package service

import (
	"context"
	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite/models"
	"io"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type fakeStore struct {
	fn        *models.Function
	container *models.Container

	markedRunning bool
	markedFree    bool
	createdCont   bool
}

func (f *fakeStore) GetActiveFunction(name string) (*models.Function, error) {
	return f.fn, nil
}

func (f *fakeStore) CreateInvocation(inv *models.Invocation) error {
	inv.ID = "inv-1"
	return nil
}

func (f *fakeStore) MarkInvocationRunning(invID string, containerID string) error {
	f.markedRunning = true
	return nil
}

func (f *fakeStore) CompleteInvocation(
	invID string,
	status string,
	exitCode int,
	responsePayload []byte,
	logs string,
	startedAt time.Time,
) error {
	return nil
}

func (f *fakeStore) CompleteInvocationAndMarkFree(
	invID string,
	status string,
	exitCode int,
	responsePayload []byte,
	logs string,
	startedAt time.Time,
	containerID string,
	shouldMarkFree bool,
) error {
	f.markedFree = true
	return nil
}

func (f *fakeStore) AcquireFreeContainer(functionID string) (*models.Container, error) {
	return f.container, nil
}

func (f *fakeStore) MarkContainerFree(containerID string) error {
	f.markedFree = true
	return nil
}

func (f *fakeStore) RemoveContainer(containerID string) error {
	return nil
}

func (f *fakeStore) CreateContainer(c *models.Container) error {
	f.createdCont = true
	return nil
}

func (f *fakeStore) StreamContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return nil, nil
}

// GetContainersByFunction(string) ([]models.Container, error)
func (f *fakeStore) GetContainersByFunction(functionID string) ([]models.Container, error) {
	if f.container == nil {
		return []models.Container{}, nil
	}
	return []models.Container{*f.container}, nil
}

func (f *fakeStore) ListFunctionVersions(name string) ([]models.Function, error) {
	// used by tests in stream_service_test and other service tests
	return nil, nil
}

func (f *fakeStore) ListFunctions() ([]models.Function, error) {
	return nil, nil
}

func (f *fakeStore) DeleteFunction(name string) error {
	return nil
}

func (f *fakeStore) GetInvocationLogs(functionID string, limit int) ([]models.Invocation, error) {
	return []models.Invocation{}, nil
}

type fakeContainerClient struct {
	createErr error
	startErr  error
	healthy   bool
	port      string
}

func (f *fakeContainerClient) CreateContainer(ctx context.Context, name, image string, cmd []string) (string, error) {
	return "c1", f.createErr
}

func (f *fakeContainerClient) StartContainer(ctx context.Context, containerID string) error {
	return f.startErr
}

func (f *fakeContainerClient) DeleteContainer(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeContainerClient) StopContainer(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeContainerClient) StatsContainer(ctx context.Context, containerID string) ([]byte, error) {
	return nil, nil
}

func (f *fakeContainerClient) LogContainer(ctx context.Context, containerID string) (string, error) {
	return "", nil
}

func (f *fakeContainerClient) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	return 0, nil
}

func (f *fakeContainerClient) StreamContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *fakeContainerClient) InvokeContainer(ctx context.Context, hostPort string, body []byte) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func (f *fakeContainerClient) InspectContainer(ctx context.Context, containerID string) (client.ContainerInspectResult, error) {

	portMap := network.PortMap{}

	if f.port != "" {
		p, _ := network.ParsePort(f.port + "/tcp")
		portMap[p] = []network.PortBinding{{HostPort: f.port}}
	}

	health := "starting"
	if f.healthy {
		health = "healthy"
	}

	return client.ContainerInspectResult{
		Container: container.InspectResponse{
			NetworkSettings: &container.NetworkSettings{
				Ports: portMap,
			},
			State: &container.State{
				Running: true, // 🔧 FIX: Must set Running to true
				Health: &container.Health{
					Status: container.HealthStatus(health),
				},
			},
		},
	}, nil
}

func (f *fakeContainerClient) ListContainers(ctx context.Context) ([]sdk.ContainerInfo, error) {
	return []sdk.ContainerInfo{}, nil
}
