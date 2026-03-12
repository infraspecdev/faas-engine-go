package sdk

import (
	"faas-engine-go/internal/buildcontext"
	"faas-engine-go/internal/config"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

func TestPullImage_Fail(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.PullImage(ctx, "")
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
}

func TestDockerEngineRunning(t *testing.T) {
	t.Parallel()

	ctx, _, cancel := setupDocker(t)
	defer cancel()

	cli, err := client.New(client.FromEnv, client.WithAPIVersionFromEnv())
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	_, err = cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		t.Fatalf("Docker engine not running: %v", err)
	}

	t.Log("Docker engine is running")
}

func TestPullImage_Success(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.PullImage(ctx, "alpine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullImage_InvalidImage(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.PullImage(ctx, "invalidimagename:latest")
	if err == nil {
		t.Fatal("expected error for invalid image")
	}
}

func TestBuildImage_Success(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	imageName := fmt.Sprintf("testimage-%d", time.Now().UnixNano())

	tarstream, err := buildcontext.CreateTarStream("../../samples/hello")
	if err != nil {
		t.Fatalf("failed to create Tar stream: %v", err)
	}

	err = docker.BuildImage(ctx, imageName, tarstream, io.Discard)
	if err != nil {
		t.Fatalf("failed to build image: %v", err)
	}

	t.Cleanup(func() {
		_ = docker.RemoveImage(ctx, imageName)
	})
}

func TestBuildImage_InvalidDirectory(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	tarstream, err := buildcontext.CreateTarStream("../test_samples/invalid")
	if err == nil {
		t.Fatal("expected error creating tar stream")
	}

	err = docker.BuildImage(ctx, "testimage", tarstream, io.Discard)
	if err == nil {
		t.Fatal("expected build to fail")
	}
}

func TestBuildImage_duplicateImageName(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.PullImage(ctx, "alpine")
	if err != nil {
		t.Fatalf("failed to pull alpine: %v", err)
	}

	tarstream, err := buildcontext.CreateTarStream("../../samples/hello")
	if err != nil {
		t.Skipf("failed to create tar stream: %v", err)
	}

	err = docker.BuildImage(ctx, "alpine", tarstream, io.Discard)
	if err == nil {
		t.Fatal("expected duplicate image name error")
	}
}

func TestTagImage_Success(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	source := "alpine:latest"
	target := fmt.Sprintf("alpine:testtag-%d", time.Now().UnixNano())

	if err := docker.PullImage(ctx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	if err := docker.TagImage(ctx, source, target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	images, err := docker.ListImages(ctx)
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

	t.Cleanup(func() {
		_ = docker.RemoveImage(ctx, target)
	})
}

func TestTagImage_InvalidSource(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.TagImage(ctx, "nonexistent:image", "test:tag")
	if err == nil {
		t.Fatal("expected error for invalid source image")
	}
}

func TestRemoveImage_Success(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	if err := docker.PullImage(ctx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	target := fmt.Sprintf("alpine:test-remove-%d", time.Now().UnixNano())

	if err := docker.TagImage(ctx, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	if err := docker.RemoveImage(ctx, target); err != nil {
		t.Fatalf("failed to remove image: %v", err)
	}
}

func TestRemoveImage_NonExistent(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	err := docker.RemoveImage(ctx, "nonexistent:image")
	if err == nil {
		t.Fatal("expected error removing non-existent image")
	}
}

func TestTagImage_InvalidTarget(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	if err := docker.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	err := docker.TagImage(ctx, "alpine:latest", "INVALID IMAGE NAME")
	if err == nil {
		t.Fatal("expected error for invalid target image")
	}
}

func TestPushImage_Success(t *testing.T) {
	t.Parallel()

	ctx, docker, cancel := setupDocker(t)
	defer cancel()

	if err := docker.PullImage(ctx, "alpine"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	target := config.ImageRef(config.FunctionsRepo, "alpine", fmt.Sprintf("testpush-%d", time.Now().UnixNano()))

	if err := docker.TagImage(ctx, "alpine:latest", target); err != nil {
		t.Fatalf("failed to tag image: %v", err)
	}

	if err := docker.PushImage(ctx, target); err != nil {
		t.Fatalf("failed to push image: %v", err)
	}

	t.Cleanup(func() {
		_ = docker.RemoveImage(ctx, target)
	})
}
