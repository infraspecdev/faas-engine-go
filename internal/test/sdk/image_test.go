package test

import (
	"context"
	"faas-engine-go/internal/buildcontext"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

var (
	testCtx    context.Context
	testCli    *client.Client
	testCancel context.CancelFunc
)

func setupDockerGlobal() (context.Context, *client.Client, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	cli, err := client.New(
		client.FromEnv,
		client.WithAPIVersionFromEnv(),
	)
	if err != nil {
		cancel()
		panic("failed to create docker client: " + err.Error())
	}

	return ctx, cli, cancel
}

func TestMain(m *testing.M) {

	// initialize docker once
	testCtx, testCli, testCancel = setupDockerGlobal()
	defer testCancel()

	// verify docker engine
	_, err := testCli.Ping(testCtx, client.PingOptions{})
	if err != nil {
		panic("docker engine not running: " + err.Error())
	}

	// pull alpine once for all tests
	if err := sdk.PullImage(testCtx, testCli, "alpine"); err != nil {
		panic("failed to pull alpine image: " + err.Error())
	}

	code := m.Run()

	os.Exit(code)
}

func TestPullImage_Fail(t *testing.T) {
	t.Parallel()

	err := sdk.PullImage(testCtx, testCli, "")

	// err = nil {no error case}
	// err = "failed to pull image " {Error case}
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestDockerEngineRunning(t *testing.T) {
	t.Parallel()

	_, err := testCli.Ping(testCtx, client.PingOptions{})
	if err != nil {
		t.Fatalf("Docker engine not running: %v", err)
	}

	t.Log("Docker engine is running")
}

func TestPullImage_Success(t *testing.T) {
	t.Parallel()

	err := sdk.PullImage(testCtx, testCli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	t.Parallel()

	err := sdk.PullImage(testCtx, testCli, "invalidimagename:latest")
	if err != nil {
		t.Log("expected error for invalid image:", err)
	} else {
		t.Fatal("expected error for invalid image, got nil")
	}
}

func TestBuildImage_Success(t *testing.T) {
	t.Parallel()

	// data, err := os.ReadFile("test_samples/function.tar")
	tarstream, err := buildcontext.CreateTarStream("../../../samples/hello")
	if err != nil {
		t.Fatalf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = sdk.BuildImage(testCtx, testCli, "testimage:latest", tarstream)
	if err != nil {
		t.Fatalf("unexpected error - failed to build image: %v", err)
	}
}

func TestBuildImage_InvalidDirectory(t *testing.T) {
	t.Parallel()

	tarstream, err := buildcontext.CreateTarStream("../test_samples/invalid")
	if err == nil {
		t.Fatal("expected error creating tar stream for invalid directory, got nil")
	}

	err = sdk.BuildImage(testCtx, testCli, "testimage:latest", tarstream)
	if err == nil {
		t.Fatal("expected build to fail for invalid directory, got nil")
	}
}

func TestBuildImage_duplicateImageName(t *testing.T) {
	t.Parallel()

	err := sdk.PullImage(testCtx, testCli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error pulling alpine image: %v", err)
	}
	t.Log("Pulled image successfully")

	defer func() {
		result, err := testCli.ImageRemove(testCtx, "alpine:latest", client.ImageRemoveOptions{Force: true})
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

	err = sdk.BuildImage(testCtx, testCli, "alpine", tarstream)
	if err == nil {
		t.Fatal("expected error for duplicate image name, got nil")
	}

	t.Logf("received expected error: %v", err)
}

func TestTagImage_Success(t *testing.T) {
	t.Parallel()

	source := "alpine:latest"
	target := "alpine:testtag"

	if err := sdk.PullImage(testCtx, testCli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	if err := sdk.TagImage(testCtx, testCli, source, target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	images, err := testCli.ImageList(testCtx, client.ImageListOptions{})
	if err != nil {
		t.Fatalf("failed to list images: %v", err)
	}

	found := false
	for _, img := range images.Items {
		for _, tag := range img.RepoTags {
			if tag == target {
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatalf("expected tagged image %s not found", target)
	}

	_, _ = testCli.ImageRemove(testCtx, target, client.ImageRemoveOptions{Force: true})
}

func TestTagImage_InvalidSource(t *testing.T) {
	t.Parallel()

	err := sdk.TagImage(testCtx, testCli, "nonexistent:image", "test:tag")
	if err == nil {
		t.Fatal("expected error for invalid source image, got nil")
	}
}

func TestRemoveImage_Success(t *testing.T) {
	t.Parallel()

	// Ensure alpine exists
	if err := sdk.PullImage(testCtx, testCli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	// Create unique tag
	target := fmt.Sprintf("alpine:test-remove-%d", time.Now().UnixNano())

	// Tag it first
	if err := sdk.TagImage(testCtx, testCli, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	// Remove tagged image
	if err := sdk.RemoveImage(testCtx, testCli, target); err != nil {
		t.Fatalf("failed to remove image: %v", err)
	}

	// Verify removal
	images, err := testCli.ImageList(testCtx, client.ImageListOptions{})
	if err != nil {
		t.Fatalf("failed to list images: %v", err)
	}

	for _, img := range images.Items {
		for _, tag := range img.RepoTags {
			if tag == target {
				t.Fatalf("image %s still exists after removal", target)
			}
		}
	}
}

func TestRemoveImage_NonExistent(t *testing.T) {
	t.Parallel()

	err := sdk.RemoveImage(testCtx, testCli, "nonexistent:image")
	if err == nil {
		t.Fatal("expected error removing non-existent image, got nil")
	}
}

func TestTagImage_InvalidTarget(t *testing.T) {
	t.Parallel()

	err := sdk.PullImage(testCtx, testCli, "alpine:latest")
	if err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	err = sdk.TagImage(testCtx, testCli, "alpine:latest", "INVALID IMAGE NAME")
	if err == nil {
		t.Fatal("expected error for invalid target image, got nil")
	}
}

func TestPushImage_Success(t *testing.T) {
	t.Parallel()

	if err := sdk.PullImage(testCtx, testCli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	target := config.ImageRef(config.FunctionsRepo, "alpine", "testpush")
	if err := sdk.TagImage(testCtx, testCli, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	if err := sdk.PushImage(testCtx, testCli, target); err != nil {
		t.Fatalf("failed to push image: %v", err)
	}
}
