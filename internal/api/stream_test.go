package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type mockStreamer struct {
	getIDFn  func(name string) (int, error)
	streamFn func(ctx context.Context, id int, out chan<- string) error
}

func (m *mockStreamer) GetFunctionID(name string) (int, error) {
	return m.getIDFn(name)
}

func (m *mockStreamer) StreamFunctionLogs(ctx context.Context, id int, out chan<- string) error {
	return m.streamFn(ctx, id, out)
}

func newStreamRequest(name string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/functions/"+name+"/logs/stream", nil)
	return mux.SetURLVars(req, map[string]string{
		"functionName": name,
	})
}

func TestLogStreamHandler_Success(t *testing.T) {

	mock := &mockStreamer{
		getIDFn: func(name string) (int, error) {
			return 1, nil
		},
		streamFn: func(ctx context.Context, id int, out chan<- string) error {
			out <- "log1"
			out <- "log2"
			close(out)
			return nil
		},
	}

	handler := LogStreamHandler(mock)

	req := newStreamRequest("test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "log1") || !strings.Contains(body, "log2") {
		t.Fatalf("expected logs in response, got: %s", body)
	}
}

func TestLogStreamHandler_FunctionNotFound(t *testing.T) {

	mock := &mockStreamer{
		getIDFn: func(name string) (int, error) {
			return 0, errors.New("not found")
		},
	}

	handler := LogStreamHandler(mock)

	req := newStreamRequest("missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestLogStreamHandler_MissingName(t *testing.T) {

	mock := &mockStreamer{}

	handler := LogStreamHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions//logs/stream", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLogStreamHandler_StreamError(t *testing.T) {

	mock := &mockStreamer{
		getIDFn: func(name string) (int, error) {
			return 1, nil
		},
		streamFn: func(ctx context.Context, id int, out chan<- string) error {
			return errors.New("stream failed")
		},
	}

	handler := LogStreamHandler(mock)

	req := newStreamRequest("test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "error:") {
		t.Fatalf("expected error in stream, got: %s", body)
	}
}
