package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"faas-engine-go/internal/sqlite/models"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

//
// FAKE STORE
//

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

func (f *fakeStore) AcquireFreeContainer(functionID int) (*models.Container, error) {
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
				Health: &container.Health{
					Status: container.HealthStatus(health),
				},
			},
		},
	}, nil
}

//
// TESTS
//

func TestInvoke_ReuseSuccess(t *testing.T) {

	store := &fakeStore{
		fn: &models.Function{ID: 1},
		container: &models.Container{
			ID:       "c1",
			HostPort: "8080",
		},
	}

	invoker := NewFunctionInvoker(&fakeContainerClient{}, &fakeImageClient{}, store)

	invoker.invokeFunc = func(ctx context.Context, port string, payload []byte) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}

	res, err := invoker.Invoke(context.Background(), "test", []byte("{}"))

	if err != nil || res == nil {
		t.Fatal("expected success")
	}
}

func TestInvoke_PullFail(t *testing.T) {

	store := &fakeStore{
		fn: &models.Function{ID: 1},
	}

	img := &fakeImageClient{
		pullErr: errors.New("fail"),
	}

	invoker := NewFunctionInvoker(&fakeContainerClient{}, img, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"))

	if err == nil {
		t.Fatal("expected error")
	}
}
