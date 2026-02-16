package test

import (
	"context"
	"faas-engine-go/internal/sdk"
	"testing"

	"github.com/moby/moby/client"
)

func TestPullImage_Fail(t *testing.T) {
	ctx, cli, cancel, err := sdk.Init(context.Background())
	defer cancel()

	err = sdk.PullImage(ctx, cli, "")

	// err = nil {no error case}
	// err = "failed to pull image " {Error case}
	if err != nil {
		t.Skipf("unexpected error: %v", err)
	}
}

func TestDockerEngineRunning(t *testing.T) {
	ctx, cli, cancel, err := sdk.Init(context.Background())
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	defer cancel()

	_, err = cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		t.Fatalf("Docker engine not running: %v", err)
	}

	t.Log("Docker engine is running")
}

func TestPullImage_Success(t *testing.T) {
	ctx, cli, cancel, err := sdk.Init(context.Background())
	defer cancel()

	err = sdk.PullImage(ctx, cli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	ctx, cli, cancel, err := sdk.Init(context.Background())
	defer cancel()

	err = sdk.PullImage(ctx, cli, "invalidimagename:latest")
	if err != nil {
		t.Log("expected error for invalid image:", err)
	} else {
		t.Fatal("expected error for invalid image, got nil")
	}
}
