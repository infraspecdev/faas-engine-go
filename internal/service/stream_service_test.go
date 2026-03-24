package service

import (
	"context"
	"testing"
	"time"

	"faas-engine-go/internal/sqlite/models"
)

// Shared test helpers are now moved to common_test_helpers.go and mock_store.go.

func TestStreamFunctionLogs_Success(t *testing.T) {

	store := &fakeStore{
		container: &models.Container{ID: "abc123"},
	}

	client := &fakeLogContainerClient{
		running: true,
		logs:    []string{"hello", "world"},
	}

	service := NewLogStreamService(client, store)

	out := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = service.StreamFunctionLogs(ctx, 1, out)
	}()

	time.Sleep(500 * time.Millisecond)
	cancel()

	var logs []string
	for len(out) > 0 {
		logs = append(logs, <-out)
	}

	if len(logs) == 0 {
		t.Fatal("expected logs")
	}
}

func TestStreamFunctionLogs_NoRunningContainer(t *testing.T) {

	store := &fakeStore{
		container: &models.Container{ID: "abc123"},
	}

	client := &fakeLogContainerClient{
		running: false,
	}

	service := NewLogStreamService(client, store)

	out := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = service.StreamFunctionLogs(ctx, 1, out)
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()

	if len(out) != 0 {
		t.Fatal("expected no logs")
	}
}

func TestStreamFunctionLogs_EmptyLogs(t *testing.T) {

	store := &fakeStore{
		container: &models.Container{ID: "abc123"},
	}

	client := &fakeLogContainerClient{
		running: true,
		logs:    []string{""},
	}

	service := NewLogStreamService(client, store)

	out := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = service.StreamFunctionLogs(ctx, 1, out)
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()

	if len(out) != 0 {
		t.Fatal("expected no logs for empty input")
	}
}

func TestGetFunctionID_Success(t *testing.T) {

	store := &fakeStore{
		fn: &models.Function{ID: 42},
	}

	service := NewLogStreamService(nil, store)

	id, err := service.GetFunctionID("test")

	if err != nil {
		t.Fatal(err)
	}

	if id != 42 {
		t.Fatalf("expected 42, got %d", id)
	}
}

func TestGetFunctionID_NotFound(t *testing.T) {

	store := &fakeStore{
		fn: nil,
	}

	service := NewLogStreamService(nil, store)

	_, err := service.GetFunctionID("test")

	if err == nil {
		t.Fatal("expected error")
	}
}
