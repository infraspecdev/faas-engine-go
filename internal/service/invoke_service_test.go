package service

import (
	"context"
	"errors"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

func TestInvoke_ReuseSuccess(t *testing.T) {

	store := &fakeStore{
		fn: &models.Function{ID: 1},
		container: &models.Container{
			ID:       "c1",
			HostPort: "8080",
		},
	}

	con := &fakeContainerClient{
		healthy: true,
		port:    "8080",
	}

	invoker := NewInvokeService(con, &fakeImageClient{}, store)

	res, err := invoker.Invoke(context.Background(), "test", []byte("{}"), "http")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatal("expected result")
	}
}

func TestInvoke_PullFail(t *testing.T) {

	store := &fakeStore{
		fn:        &models.Function{ID: 1},
		container: nil, // 🔥 IMPORTANT
	}

	img := &fakeImageClient{
		pullErr: errors.New("fail"),
	}

	invoker := NewInvokeService(&fakeContainerClient{}, img, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_CreateContainerFail(t *testing.T) {

	store := &fakeStore{
		fn:        &models.Function{ID: 1},
		container: nil, // 🔥 IMPORTANT
	}

	con := &fakeContainerClient{
		createErr: errors.New("create failed"),
	}

	invoker := NewInvokeService(con, &fakeImageClient{}, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_StartContainerFail(t *testing.T) {

	store := &fakeStore{
		fn:        &models.Function{ID: 1},
		container: nil, // 🔥 IMPORTANT
	}

	con := &fakeContainerClient{
		startErr: errors.New("start failed"),
	}

	invoker := NewInvokeService(con, &fakeImageClient{}, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInvoke_UnhealthyContainer(t *testing.T) {

	store := &fakeStore{
		fn:        &models.Function{ID: 1},
		container: nil, // 🔥 IMPORTANT
	}

	con := &fakeContainerClient{
		healthy: false,
		port:    "8080",
	}

	invoker := NewInvokeService(con, &fakeImageClient{}, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"), "http")

	if err == nil {
		t.Fatal("expected unhealthy error")
	}
}
