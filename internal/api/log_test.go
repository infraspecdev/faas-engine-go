package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type mockLogger struct {
	byNameFn        func(name string, limit int) ([]service.LogEntry, error)
	byNameVersionFn func(name, version string, limit int) ([]service.LogEntry, error)
}

func (m *mockLogger) GetLogsByName(name string, limit int) ([]service.LogEntry, error) {
	return m.byNameFn(name, limit)
}

func (m *mockLogger) GetLogsByNameAndVersion(name, version string, limit int) ([]service.LogEntry, error) {
	return m.byNameVersionFn(name, version, limit)
}

func newLogRequest(url string, functionName string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	return mux.SetURLVars(req, map[string]string{
		"functionName": functionName,
	})
}

func TestLogHandler_ByName_Success(t *testing.T) {

	mock := &mockLogger{
		byNameFn: func(name string, limit int) ([]service.LogEntry, error) {
			if limit != 20 {
				t.Fatalf("expected default limit 20, got %d", limit)
			}
			return []service.LogEntry{
				{ID: "1", Status: "ok"},
			}, nil
		},
	}

	handler := LogHandler(mock)

	req := newLogRequest("/functions/test/logs", "test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLogHandler_ByVersion_Success(t *testing.T) {

	mock := &mockLogger{
		byNameVersionFn: func(name, version string, limit int) ([]service.LogEntry, error) {
			if version != "v1" {
				t.Fatalf("expected version v1, got %s", version)
			}
			return []service.LogEntry{
				{ID: "1", Status: "ok"},
			}, nil
		},
	}

	handler := LogHandler(mock)

	req := newLogRequest("/functions/test/logs?version=v1", "test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLogHandler_CustomLimit(t *testing.T) {

	mock := &mockLogger{
		byNameFn: func(name string, limit int) ([]service.LogEntry, error) {
			if limit != 5 {
				t.Fatalf("expected limit 5, got %d", limit)
			}
			return []service.LogEntry{}, nil
		},
	}

	handler := LogHandler(mock)

	req := newLogRequest("/functions/test/logs?limit=5", "test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLogHandler_NotFoundError(t *testing.T) {

	mock := &mockLogger{
		byNameFn: func(name string, limit int) ([]service.LogEntry, error) {
			return nil, service.ErrFunctionNotFound
		},
	}

	handler := LogHandler(mock)

	req := newLogRequest("/functions/test/logs", "test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestLogHandler_InternalError(t *testing.T) {

	mock := &mockLogger{
		byNameFn: func(name string, limit int) ([]service.LogEntry, error) {
			return nil, errors.New("db failed")
		},
	}

	handler := LogHandler(mock)

	req := newLogRequest("/functions/test/logs", "test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestLogHandler_MissingFunctionName(t *testing.T) {

	mock := &mockLogger{}

	handler := LogHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions//logs", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
