package sdk

import (
	"context"
	"faas-engine-go/internal/buildcontext"
	"faas-engine-go/internal/config"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

var (
	testCtx    context.Context
	testCli    *client.Client
	testDocker *DockerClient
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
	testDocker = NewDockerClient(testCli)
	defer testCancel()

	// verify docker engine
	_, err := testCli.Ping(testCtx, client.PingOptions{})
	if err != nil {
		panic("docker engine not running: " + err.Error())
	}

	// pull alpine once for all tests
	if err := testDocker.PullImage(testCtx, "alpine"); err != nil {
		panic("failed to pull alpine image: " + err.Error())
	}

	code := m.Run()

	os.Exit(code)
}

func TestPullImage_Fail(t *testing.T) {
	t.Parallel()

	err := testDocker.PullImage(testCtx, "")

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

	err := testDocker.PullImage(testCtx, "alpine")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	t.Parallel()

	err := testDocker.PullImage(testCtx, "invalidimagename:latest")
	if err != nil {
		t.Log("expected error for invalid image:", err)
	} else {
		t.Fatal("expected error for invalid image, got nil")
	}
}

func TestBuildImage_Success(t *testing.T) {
	t.Parallel()

	// data, err := os.ReadFile("test_samples/function.tar")
	tarstream, err := buildcontext.CreateTarStream("../../samples/hello")
	if err != nil {
		t.Fatalf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = testDocker.BuildImage(testCtx, "testimage:latest", tarstream)
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

	err = testDocker.BuildImage(testCtx, "testimage:latest", tarstream)
	if err == nil {
		t.Fatal("expected build to fail for invalid directory, got nil")
	}
}

func TestBuildImage_duplicateImageName(t *testing.T) {
	t.Parallel()

	err := testDocker.PullImage(testCtx, "alpine")
	if err != nil {
		t.Fatalf("unexpected error pulling alpine image: %v", err)
	}
	t.Log("Pulled image successfully")

	defer func() {
		err := testDocker.RemoveImage(testCtx, "alpine:latest")
		if err != nil {
			t.Logf("failed to remove image: %v", err)
		}
	}()

	tarstream, err := buildcontext.CreateTarStream("../../samples/hello")
	if err != nil {
		t.Skipf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = testDocker.BuildImage(testCtx, "alpine", tarstream)
	if err == nil {
		t.Fatal("expected error for duplicate image name, got nil")
	}

	t.Logf("received expected error: %v", err)
}

func TestTagImage_Success(t *testing.T) {
	t.Parallel()

	source := "alpine:latest"
	target := "alpine:testtag"

	if err := testDocker.PullImage(testCtx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	if err := testDocker.TagImage(testCtx, source, target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	images, err := testDocker.ListImages(testCtx)
	if err != nil {
		t.Fatalf("failed to list images: %v", err)
	}

	found := false
	for _, img := range images {
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

	err := testDocker.TagImage(testCtx, "nonexistent:image", "test:tag")
	if err == nil {
		t.Fatal("expected error for invalid source image, got nil")
	}
}

func TestRemoveImage_Success(t *testing.T) {
	t.Parallel()

	// Ensure alpine exists
	if err := testDocker.PullImage(testCtx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	// Create unique tag
	target := fmt.Sprintf("alpine:test-remove-%d", time.Now().UnixNano())

	// Tag it first
	if err := testDocker.TagImage(testCtx, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	// Remove tagged image
	if err := testDocker.RemoveImage(testCtx, target); err != nil {
		t.Fatalf("failed to remove image: %v", err)
	}

	// Verify removal
	images, err := testDocker.ListImages(testCtx)
	if err != nil {
		t.Fatalf("failed to list images: %v", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == target {
				t.Fatalf("image %s still exists after removal", target)
			}
		}
	}
}

func TestRemoveImage_NonExistent(t *testing.T) {
	t.Parallel()

	err := testDocker.RemoveImage(testCtx, "nonexistent:image")
	if err == nil {
		t.Fatal("expected error removing non-existent image, got nil")
	}
}

func TestTagImage_InvalidTarget(t *testing.T) {
	t.Parallel()

	err := testDocker.PullImage(testCtx, "alpine:latest")
	if err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	err = testDocker.TagImage(testCtx, "alpine:latest", "INVALID IMAGE NAME")
	if err == nil {
		t.Fatal("expected error for invalid target image, got nil")
	}
}

func TestPushImage_Success(t *testing.T) {
	t.Parallel()

	if err := testDocker.PullImage(testCtx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	target := config.ImageRef(config.FunctionsRepo, "alpine", "testpush")
	if err := testDocker.TagImage(testCtx, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	if err := testDocker.PushImage(testCtx, target); err != nil {
		t.Fatalf("failed to push image: %v", err)
	}
}
