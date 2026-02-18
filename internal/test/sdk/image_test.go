package test

import (
	"faas-engine-go/internal/buildcontext"
	"faas-engine-go/internal/sdk"
	"testing"

	"github.com/moby/moby/client"
)

func TestPullImage_Fail(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "")

	// err = nil {no error case}
	// err = "failed to pull image " {Error case}
	if err != nil {
		t.Skipf("unexpected error: %v", err)
	}
}

func TestDockerEngineRunning(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	_, err := cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		t.Fatalf("Docker engine not running: %v", err)
	}

	t.Log("Docker engine is running")
}

func TestPullImage_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "invalidimagename:latest")
	if err != nil {
		t.Log("expected error for invalid image:", err)
	} else {
		t.Fatal("expected error for invalid image, got nil")
	}
}

func TestBuildImage_Success(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	// data, err := os.ReadFile("test_samples/function.tar")
	tarstream, err := buildcontext.CreateTarStream("../test_samples/hello")
	if err != nil {
		t.Skipf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = sdk.BuildImage(ctx, cli, "testimage:latest", tarstream)
	if err != nil {
		t.Fatalf("unexpected error - failed to build image: %v", err)
	}
}

func TestBuildImage_InvalidDirectory(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	tarstream, err := buildcontext.CreateTarStream("../test_samples/invalid")
	if err != nil {
		t.Skip("unexpected error - failed to create Tar stream:", err)
	}

	err = sdk.BuildImage(ctx, cli, "testimage:latest", tarstream)
	if err != nil {
		t.Log("expected error for invalid directory:", err)
	}
}

func TestBuildImage_duplicateImageName(t *testing.T) {
	ctx, cli, cancel := SetupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error pulling alpine image: %v", err)
	}
	t.Log("Pulled image successfully")

	defer func() {
		result, err := cli.ImageRemove(ctx, "alpine:latest", client.ImageRemoveOptions{Force: true})
		if err != nil {
			t.Logf("failed to remove image: %v", err)
		} else {
			t.Log("Cleaned up image successfully:", result)
		}
	}()

	tarstream, err := buildcontext.CreateTarStream("../../../samples/hello")
	if err != nil {
		t.Skipf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = sdk.BuildImage(ctx, cli, "alpine", tarstream)
	if err == nil {
		t.Fatalf("unexpected error for duplicate image name: %v", err)
	}

	t.Logf("received expected error: %v", err)
}
