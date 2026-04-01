package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type mockVersionService struct {
	getFn func(name string) ([]service.VersionInfo, error)
}

func (m *mockVersionService) GetVersions(name string) ([]service.VersionInfo, error) {
	return m.getFn(name)
}

func newVersionRequest(name string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/functions/"+name+"/versions", nil)
	return mux.SetURLVars(req, map[string]string{
		"functionName": name,
	})
}

func TestFunctionVersionsHandler_Success(t *testing.T) {

	mock := &mockVersionService{
		getFn: func(name string) ([]service.VersionInfo, error) {
			return []service.VersionInfo{
				{Version: "v1", Active: false},
				{Version: "v2", Active: true},
			}, nil
		},
	}

	handler := FunctionVersionsHandler(mock)

	req := newVersionRequest("test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestFunctionVersionsHandler_NotFound(t *testing.T) {

	mock := &mockVersionService{
		getFn: func(name string) ([]service.VersionInfo, error) {
			return nil, service.ErrFunctionNotFound
		},
	}

	handler := FunctionVersionsHandler(mock)

	req := newVersionRequest("missing")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestFunctionVersionsHandler_InternalError(t *testing.T) {

	mock := &mockVersionService{
		getFn: func(name string) ([]service.VersionInfo, error) {
			return nil, errors.New("db failure")
		},
	}

	handler := FunctionVersionsHandler(mock)

	req := newVersionRequest("test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestFunctionVersionsHandler_MissingName(t *testing.T) {

	mock := &mockVersionService{}

	handler := FunctionVersionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions//versions", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
