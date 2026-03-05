package sdk

import (
	"context"
	"testing"
)

func TestInit_Success(t *testing.T) {
	parent := context.Background()

	ctx, cli, cancel, err := Init(parent)
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

	ctx, _, cancel, err := Init(parent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cancel()

	cancelParent()

	<-ctx.Done()

	if ctx.Err() != context.Canceled {
		t.Fatalf("expected context canceled, got %v", ctx.Err())
	}
}
