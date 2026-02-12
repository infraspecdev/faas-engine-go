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

func TestInit_ContextTimeout(t *testing.T) {

	ctx, _, cancel, err := sdk.Init(context.Background())
	defer cancel()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}

	if time.Until(deadline) > 10*time.Second {
		t.Fatal("deadline exceeds expected timeout")
	}
}
