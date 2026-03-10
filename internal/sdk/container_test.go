package sdk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupDocker(t *testing.T) (context.Context, *DockerClient, func()) {
	t.Helper()

	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cli, cancel, err := Init(context.Background())
	if err != nil {
		t.Fatalf("failed to init sdk: %v", err)
	}

	docker := NewDockerClient(cli)

	return ctx, docker, cancel
}

func createTestContainer(t *testing.T, ctx context.Context, docker *DockerClient, name string) string {
	t.Helper()

	id, err := docker.CreateContainer(
		ctx,
		name,
		"alpine:latest",
		[]string{"sh", "-c", "while true; do sleep 1; done"},
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	t.Cleanup(func() {
		_ = docker.DeleteContainer(context.Background(), id)
	})

	return id
}

func TestCreateContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-create-success")

	if id == "" {
		t.Fatal("expected container ID to not be empty")
	}
}

func TestCreateContainer_InvalidImage(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	_, err := docker.CreateContainer(ctx, "test-invalid-image", "", []string{"echo", "hello world"})
	if err == nil {
		t.Fatalf("expected error for invalid image")
	}
}

func TestCreateContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	_, err := docker.CreateContainer(ctx, "invalid/name/with/slashes", "alpine:latest", []string{"echo", "hello world"})
	if err == nil {
		t.Fatalf("expected error for invalid container name")
	}
}

func TestDeleteContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-delete-success")

	err := docker.DeleteContainer(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.DeleteContainer(ctx, "nonexistent-container")
	if err == nil {
		t.Fatalf("expected error when deleting nonexistent container")
	}
}

func TestStartContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-start-success")

	err := docker.StartContainer(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.StartContainer(ctx, "nonexistent-container")
	if err == nil {
		t.Fatalf("expected error when starting nonexistent container")
	}
}

func TestStopContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-stop-success")

	err := docker.StopContainer(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.StopContainer(ctx, "")
	if err == nil {
		t.Fatalf("expected error for empty container id")
	}
}

func TestStatsContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-stats-success")

	err := docker.StartContainer(ctx, id)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	fmt.Println("Container started, fetching stats...")

	// stats, err := docker.StatsContainer(ctx, id)
	// if err != nil {
	// 	t.Fatalf("unexpected error: %v", err)
	// }

	// if len(stats) == 0 {
	// 	t.Fatal("expected stats data but got empty")
	// }
}

func TestStatsContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	_, err := docker.StatsContainer(ctx, "invalid-container-id")

	if err == nil {
		t.Fatal("expected error for invalid container id")
	}
}

func TestWaitContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-wait-success")

	err := docker.StartContainer(ctx, id)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	code, err := docker.WaitContainer(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if code != 0 {
		t.Fatalf("expected exit code 0 but got %d", code)
	}
}

func TestWaitContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	_, err := docker.WaitContainer(ctx, "invalid-container")

	if err == nil {
		t.Fatal("expected error for invalid container")
	}
}

func TestInspectContainer_Success(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, docker, "test-inspect-success")

	info, err := docker.InspectContainer(ctx, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.Container.ID == "" {
		t.Fatal("expected valid container inspect result")
	}
}

func TestInspectContainer_Fail(t *testing.T) {
	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	_, err := docker.InspectContainer(ctx, "invalid-container")

	if err == nil {
		t.Fatal("expected error for invalid container")
	}
}

func TestInvokeContainer_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"message":"hello"}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	port := strings.Split(server.URL, ":")[2]

	resp, err := InvokeContainer(context.Background(), port, []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp["message"] != "hello" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestInvokeContainer_Fail(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	}))
	defer server.Close()

	port := strings.Split(server.URL, ":")[2]

	_, err := InvokeContainer(context.Background(), port, []byte(`{}`))

	if err == nil {
		t.Fatal("expected error from container")
	}
}
