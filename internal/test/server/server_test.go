package test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestServerGracefulShutdown(t *testing.T) {
	srv := &http.Server{
		Addr: ":0",
	}

	// start server
	go func() {
		_ = srv.ListenAndServe()
	}()

	// allow server to start
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("expected graceful shutdown, got error: %v", err)
	}
}
