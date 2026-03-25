package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"faas-engine-go/internal/sqlite"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

/*
========================
TEST DB SETUP
========================
*/

func setupTestDB(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	sqlite.DB = db

	// Realistic schema
	_, err = db.Exec(`
	CREATE TABLE functions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		version TEXT,
		package_checksum TEXT,
		image TEXT,
		runtime TEXT,
		schedule_cron TEXT,
		endpoint TEXT,
		status TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		t.Fatalf("create functions: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE containers (
		id TEXT PRIMARY KEY,
		function_id INTEGER,
		status TEXT,
		host_port TEXT,
		last_used TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		t.Fatalf("create containers: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE invocations (
		id TEXT PRIMARY KEY,
		function_id INTEGER,
		container_id TEXT,
		trigger_type TEXT,
		status TEXT,
		exit_code INTEGER,
		duration_ms INTEGER,
		request_payload TEXT,
		response_payload TEXT,
		logs TEXT,
		started_at DATETIME,
		finished_at DATETIME
	);
	`)
	if err != nil {
		t.Fatalf("create invocations: %v", err)
	}

	_, err = db.Exec(`
INSERT INTO functions (
	id,
	name,
	version,
	package_checksum,
	image,
	runtime,
	schedule_cron,
	endpoint,
	status,
	created_at
) VALUES (
	1,
	'hello',
	'v1',
	'checksum',
	'hello:latest',
	'node',
	'',
	'',
	'active',
	CURRENT_TIMESTAMP
);
`)
	if err != nil {
		t.Fatalf("insert function: %v", err)
	}
}

/*
========================
FAKE CONTAINER CLIENT
========================
*/

type fakeContainerClient struct {
	createCalled bool
	startCalled  bool

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

func (f *fakeContainerClient) DeleteContainer(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeContainerClient) LogContainer(ctx context.Context, containerID string) (string, error) {
	return "", nil
}

func (f *fakeContainerClient) StatsContainer(ctx context.Context, containerID string) ([]byte, error) {
	return nil, nil
}

func (f *fakeContainerClient) StopContainer(ctx context.Context, containerID string) error {
	return nil
}

func (f *fakeContainerClient) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	return 0, nil
}

func (f *fakeContainerClient) InspectContainer(ctx context.Context, containerID string) (client.ContainerInspectResult, error) {

	portMap := network.PortMap{}

	if f.port != "" {
		p, _ := network.ParsePort(f.port + "/tcp")
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
			Running: true,
			Health: &container.Health{
				Status: container.HealthStatus(health),
			},
		},
	}

	return client.ContainerInspectResult{
		Container: resp,
	}, nil
}

/*
========================
TESTS
========================
*/

func TestInvoke_Success(t *testing.T) {
	setupTestDB(t)

	img := &fakeImageClient{}
	con := &fakeContainerClient{
		healthy: true,
		port:    "8080",
	}

	invoker := NewFunctionInvoker(con, img)

	invoker.invokeFunc = func(ctx context.Context, port string, payload []byte) (map[string]any, error) {
		return map[string]any{"result": "ok"}, nil
	}

	res, err := invoker.Invoke(context.Background(), "hello", []byte("{}"), "http")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatal("expected result")
	}
}

func TestInvoke_PullImageFail(t *testing.T) {
	setupTestDB(t)

	img := &fakeImageClient{
		pullErr: errors.New("pull failed"),
	}

	con := &fakeContainerClient{}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_CreateContainerFail(t *testing.T) {
	setupTestDB(t)

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		createErr: errors.New("create failed"),
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_StartContainerFail(t *testing.T) {
	setupTestDB(t)

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		startErr: errors.New("start failed"),
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_UnhealthyContainer(t *testing.T) {
	setupTestDB(t)

	img := &fakeImageClient{}

	con := &fakeContainerClient{
		healthy: false,
		port:    "8080",
	}

	invoker := NewFunctionInvoker(con, img)

	_, err := invoker.Invoke(context.Background(), "hello", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected unhealthy error")
	}
}
