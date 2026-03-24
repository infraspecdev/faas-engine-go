package service

import (
	"bytes"
	"context"
	"errors"
	"faas-engine-go/internal/config"
	"io"
	"testing"
)

// fakeImageClient is now provided by common_test_helpers.go

func TestDeploy_Success(t *testing.T) {
	t.Parallel()

	fake := &fakeImageClient{}

	deployer := NewDeployService(fake)
	deployer.getVersion = func(name string) (string, error) {
		return "v1", nil
	}

	err := deployer.Deploy(context.Background(), "hello", bytes.NewReader([]byte("test")), io.Discard)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fake.buildCalled {
		t.Fatal("BuildImage was not called")
	}

	if !fake.tagCalled {
		t.Fatal("TagImage was not called")
	}

	if !fake.pushCalled {
		t.Fatal("PushImage was not called")
	}

	if !fake.removeCalled {
		t.Fatal("RemoveImage was not called")
	}

	expectedTarget := config.ImageRef(config.FunctionsRepo, "hello", "v1")

	if fake.lastTagTarget != expectedTarget {
		t.Fatalf("expected tag target %s, got %s", expectedTarget, fake.lastTagTarget)
	}
}

func TestDeploy_BuildImageFail(t *testing.T) {
	t.Parallel()

	fake := &fakeImageClient{
		buildErr: errors.New("build failed"),
	}

	deployer := NewDeployService(fake)
	deployer.getVersion = func(name string) (string, error) {
		return "v1", nil
	}

	err := deployer.Deploy(context.Background(), "hello", bytes.NewReader([]byte("test")), io.Discard)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !fake.buildCalled {
		t.Fatal("BuildImage should be called")
	}

	if fake.tagCalled {
		t.Fatal("TagImage should NOT be called")
	}

	if fake.pushCalled {
		t.Fatal("PushImage should NOT be called")
	}

	if fake.removeCalled {
		t.Fatal("RemoveImage should NOT be called")
	}
}

func TestDeploy_TagImageFail(t *testing.T) {
	t.Parallel()

	fake := &fakeImageClient{
		tagErr: errors.New("tag failed"),
	}

	deployer := NewDeployService(fake)
	deployer.getVersion = func(name string) (string, error) {
		return "v1", nil
	}

	err := deployer.Deploy(context.Background(), "hello", bytes.NewReader([]byte("test")), io.Discard)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !fake.buildCalled {
		t.Fatal("BuildImage should be called")
	}

	if !fake.tagCalled {
		t.Fatal("TagImage should be called")
	}

	if fake.pushCalled {
		t.Fatal("PushImage should NOT be called")
	}

	if fake.removeCalled {
		t.Fatal("RemoveImage should NOT be called")
	}
}

func TestDeploy_PushImageFail(t *testing.T) {
	t.Parallel()

	fake := &fakeImageClient{
		pushErr: errors.New("push failed"),
	}

	deployer := NewDeployService(fake)
	deployer.getVersion = func(name string) (string, error) {
		return "v1", nil
	}

	err := deployer.Deploy(context.Background(), "hello", bytes.NewReader([]byte("test")), io.Discard)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !fake.buildCalled {
		t.Fatal("BuildImage should be called")
	}

	if !fake.tagCalled {
		t.Fatal("TagImage should be called")
	}

	if !fake.pushCalled {
		t.Fatal("PushImage should be called")
	}

	if fake.removeCalled {
		t.Fatal("RemoveImage should NOT be called")
	}
}

func TestDeploy_RemoveImageFail(t *testing.T) {
	t.Parallel()

	fake := &fakeImageClient{
		removeErr: errors.New("remove failed"),
	}

	deployer := NewDeployService(fake)
	deployer.getVersion = func(name string) (string, error) {
		return "v1", nil
	}

	err := deployer.Deploy(context.Background(), "hello", bytes.NewReader([]byte("test")), io.Discard)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !fake.removeCalled {
		t.Fatal("RemoveImage should be called")
	}
}
