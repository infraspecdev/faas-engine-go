package service

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type fakeContainerClient struct {
	createCalled bool
	startCalled  bool
	stopCalled   bool

	createErr error
	startErr  error

	healthy bool
	port    string
}

func (f *fakeContainerClient) CreateContainer(ctx context.Context, name, image string, cmd []string) (string, error) {
	f.createCalled = true

	if f.createErr != nil {
		return "", f.createErr
	}

	return "test-container", nil
}

func (f *fakeContainerClient) StartContainer(ctx context.Context, containerID string) error {
	f.startCalled = true
	return f.startErr
}

func (f *fakeContainerClient) StopContainer(ctx context.Context, containerID string) error {
	f.stopCalled = true
	return nil
}

func (f *fakeContainerClient) DeleteContainer(ctx context.Context, containerID string) error {
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
		p, err := network.ParsePort(f.port + "/tcp")

		if err != nil {
			return client.ContainerInspectResult{}, err
		}

		portMap[p] = []network.PortBinding{
			{HostPort: f.port},
		}
	}

	health := "starting"
	if f.healthy {
		health = "healthy"
	}

	resp := container.InspectResponse{
		NetworkSettings: &container.NetworkSettings{
			Ports: portMap,
		},
		State: &container.State{
			Health: &container.Health{
				Status: container.HealthStatus(health),
			},
		},
	}

	return client.ContainerInspectResult{
		Container: resp,
	}, nil
}

func TestInvoke_Success(t *testing.T) {

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		healthy: true,
		port:    "8080",
	}

	invoker := NewFunctionInvoker(con, img)

	invoker.invokeFunc = func(ctx context.Context, port string, payload []byte) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	}

	result, err := invoker.Invoke(context.Background(), "hello", []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result but got nil")
	}

	if !img.pullCalled {
		t.Fatal("PullImage should be called")
	}

	if !con.createCalled {
		t.Fatal("CreateContainer should be called")
	}

	if !con.startCalled {
		t.Fatal("StartContainer should be called")
	}
}

func TestInvoke_PullImageFail(t *testing.T) {

	img := &fakeImageClient{
		pullErr: errors.New("pull failed"),
	}

	con := &fakeContainerClient{}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"))

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if con.createCalled {
		t.Fatal("container should not be created when pull fails")
	}
}

func TestInvoke_CreateContainerFail(t *testing.T) {

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		createErr: errors.New("create failed"),
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"))

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !img.pullCalled {
		t.Fatal("PullImage should be called")
	}

	if !con.createCalled {
		t.Fatal("CreateContainer should be called")
	}
}

func TestInvoke_StartContainerFail(t *testing.T) {

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		startErr: errors.New("start failed"),
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"))

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !con.startCalled {
		t.Fatal("StartContainer should be called")
	}
}

func TestInvoke_UnhealthyContainer(t *testing.T) {

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		healthy: false,
		port:    "9000",
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"))

	if err == nil {
		t.Fatal("expected container unhealthy error")
	}
}
