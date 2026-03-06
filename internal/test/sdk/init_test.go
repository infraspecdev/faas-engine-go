package test

import (
	"context"
	"faas-engine-go/internal/sdk"
	"testing"
	"time"
)

func TestInit_Success(t *testing.T) {
	parent := context.Background()

	ctx, cli, cancel, err := sdk.Init(parent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ctx == nil {
		t.Fatal("expected ctx to not be nil")
	}

	if cli == nil {
		t.Fatal("expected client to not be nil")
	}

	if cancel == nil {
		t.Fatal("expected cancel func to not be nil")
	}

	cancel()
}

func TestInit_ContextCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())

	ctx, _, cancel, err := sdk.Init(parent)
	defer cancel()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancelParent()

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("expected child context to be cancelled")
	}
}
