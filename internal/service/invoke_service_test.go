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

	invoker := NewInvokeService(&fakeContainerClient{}, &fakeImageClient{}, store)

	res, err := invoker.Invoke(context.Background(), "test", []byte("{}"))

	if err != nil || res == nil {
		t.Fatal("expected success")
	}
}

func TestInvoke_PullFail(t *testing.T) {

	store := &fakeStore{
		fn: &models.Function{ID: 1},
	}

	img := &fakeImageClient{
		pullErr: errors.New("fail"),
	}

	invoker := NewInvokeService(&fakeContainerClient{}, img, store)

	_, err := invoker.Invoke(context.Background(), "test", []byte("{}"))

	if err == nil {
		t.Fatal("expected error")
	}
}
