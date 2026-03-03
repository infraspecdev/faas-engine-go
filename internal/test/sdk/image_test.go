package test

import (
	"faas-engine-go/internal/buildcontext"
	"faas-engine-go/internal/config"
	"faas-engine-go/internal/sdk"
	"fmt"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

func TestPullImage_Fail(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "")

	// err = nil {no error case}
	// err = "failed to pull image " {Error case}
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestDockerEngineRunning(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	_, err := cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		t.Fatalf("Docker engine not running: %v", err)
	}

	t.Log("Docker engine is running")
}

func TestPullImage_Success(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "invalidimagename:latest")
	if err != nil {
		t.Log("expected error for invalid image:", err)
	} else {
		t.Fatal("expected error for invalid image, got nil")
	}
}

func TestBuildImage_Success(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	// data, err := os.ReadFile("test_samples/function.tar")
	tarstream, err := buildcontext.CreateTarStream("../../../samples/hello")
	if err != nil {
		t.Fatalf("unexpected error - failed to create Tar stream: %v", err)
	}

	err = sdk.BuildImage(ctx, cli, "testimage:latest", tarstream)
	if err != nil {
		t.Fatalf("unexpected error - failed to build image: %v", err)
	}
}

func TestBuildImage_InvalidDirectory(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	tarstream, err := buildcontext.CreateTarStream("../test_samples/invalid")
	if err == nil {
		t.Fatal("expected error creating tar stream for invalid directory, got nil")
	}

	err = sdk.BuildImage(ctx, cli, "testimage:latest", tarstream)
	if err == nil {
		t.Fatal("expected build to fail for invalid directory, got nil")
	}
}

func TestBuildImage_duplicateImageName(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
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
		t.Fatal("expected error for duplicate image name, got nil")
	}

	t.Logf("received expected error: %v", err)
}

func TestTagImage_Success(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	source := "alpine:latest"
	target := "alpine:testtag"

	if err := sdk.PullImage(ctx, cli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	if err := sdk.TagImage(ctx, cli, source, target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	images, err := cli.ImageList(ctx, client.ImageListOptions{})
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

	_, _ = cli.ImageRemove(ctx, target, client.ImageRemoveOptions{Force: true})
}

func TestTagImage_InvalidSource(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.TagImage(ctx, cli, "nonexistent:image", "test:tag")
	if err == nil {
		t.Fatal("expected error for invalid source image, got nil")
	}
}

func TestRemoveImage_Success(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	// Ensure alpine exists
	if err := sdk.PullImage(ctx, cli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	// Create unique tag
	target := fmt.Sprintf("alpine:test-remove-%d", time.Now().UnixNano())

	// Tag it first
	if err := sdk.TagImage(ctx, cli, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	// Remove tagged image
	if err := sdk.RemoveImage(ctx, cli, target); err != nil {
		t.Fatalf("failed to remove image: %v", err)
	}

	// Verify removal
	images, err := cli.ImageList(ctx, client.ImageListOptions{})
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
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.RemoveImage(ctx, cli, "nonexistent:image")
	if err == nil {
		t.Fatal("expected error removing non-existent image, got nil")
	}
}

func TestTagImage_InvalidTarget(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	err := sdk.PullImage(ctx, cli, "alpine:latest")
	if err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	err = sdk.TagImage(ctx, cli, "alpine:latest", "INVALID IMAGE NAME")
	if err == nil {
		t.Fatal("expected error for invalid target image, got nil")
	}
}

func TestPushImage_Success(t *testing.T) {
	ctx, cli, cancel := setupDocker(t)
	defer cancel()

	if err := sdk.PullImage(ctx, cli, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	target := config.ImageRef(config.FunctionsRepo, "alpine", "testpush")
	if err := sdk.TagImage(ctx, cli, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	if err := sdk.PushImage(ctx, cli, target); err != nil {
		t.Fatalf("failed to push image: %v", err)
	}
}
