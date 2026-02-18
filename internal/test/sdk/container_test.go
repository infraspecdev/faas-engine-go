package test

import (
	"context"
	"faas-engine-go/internal/sdk"
	"strings"
	"testing"

	"github.com/moby/moby/client"
)

func SetupDocker(t *testing.T) (context.Context, *client.Client, func()) {
	t.Helper()

	ctx, cli, cancel, err := sdk.Init(context.Background())
	if err != nil {
		t.Fatalf("failed to init sdk: %v", err)
	}

	return ctx, cli, cancel
}

func createTestContainer(t *testing.T, ctx context.Context, cli *client.Client, name string) string {
	t.Helper()

	if err := sdk.PullImage(ctx, cli, "alpine"); err != nil {
		t.Fatalf("pull failed: %v", err)
	}

	id, err := sdk.CreateContainer(ctx, cli, name, "alpine", []string{"echo", "hello world"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	t.Cleanup(func() {
		_ = sdk.DeleteContainer(context.Background(), cli, id)
	})

	return id
}

func TestCreateContainer_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine1")

	if id == "" {
		t.Fatal("expected container ID to not be empty")
	}
}

func TestCreateContainer_InvalidImage(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	_, err := sdk.CreateContainer(ctx, cli, "alpine", "", []string{"echo", "hello world"})
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateContainer_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	_, err := sdk.CreateContainer(ctx, cli, "invalid/name/with/slashes", "alpine", []string{"echo", "hello world"})
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContainer_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine")

	err := sdk.DeleteContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteContainer_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.DeleteContainer(ctx, cli, "alpine360")
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartContainer_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine")

	err := sdk.StartContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartContainer_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.StartContainer(ctx, cli, "alpine123")
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopContainer_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine")

	err := sdk.StopContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopContainer_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.StopContainer(ctx, cli, "")
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainerLogs_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine1")

	err := sdk.StartContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs, err := sdk.LogContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = sdk.WaitContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello world\n"
	if logs != expected {
		t.Fatalf("expected logs to be '%s', got '%s'", expected, strings.Trim(logs, "\n"))
	}
}

func TestContainerLogs_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()
	_, err := sdk.LogContainer(ctx, cli, "nonexistent")
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitContainer_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	id := createTestContainer(t, ctx, cli, "alpine1")

	err := sdk.StartContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statuscode, err := sdk.WaitContainer(ctx, cli, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statuscode != 0 {
		t.Fatalf("expected status code 0, got %d", statuscode)
	}
}

func TestWaitContainer_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()
	_, err := sdk.WaitContainer(ctx, cli, "nonexistent")
	if err == nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
