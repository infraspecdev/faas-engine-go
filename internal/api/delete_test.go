package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"faas-engine-go/internal/service"

	"github.com/gorilla/mux"
)

type mockDeleter struct {
	deleteFn func(name string) ([]string, error)
}

func (m *mockDeleter) DeleteFunction(name string) ([]string, error) {
	return m.deleteFn(name)
}

func newRequest(method, url string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	vars := map[string]string{
		"functionName": "test-fn",
	}
	return mux.SetURLVars(req, vars)
}

func TestDeleteHandler_Success(t *testing.T) {

	mock := &mockDeleter{
		deleteFn: func(name string) ([]string, error) {
			return nil, nil
		},
	}

	handler := DeleteFunctionHandler(mock)

	req := newRequest(http.MethodDelete, "/functions/test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestDeleteHandler_NotFound(t *testing.T) {

	mock := &mockDeleter{
		deleteFn: func(name string) ([]string, error) {
			return nil, service.ErrFunctionNotFound
		},
	}

	handler := DeleteFunctionHandler(mock)

	req := newRequest(http.MethodDelete, "/functions/test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteHandler_PartialFailure(t *testing.T) {

	mock := &mockDeleter{
		deleteFn: func(name string) ([]string, error) {
			return []string{"v1", "v2"}, errors.New("partial delete failure")
		},
	}

	handler := DeleteFunctionHandler(mock)

	req := newRequest(http.MethodDelete, "/functions/test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDeleteHandler_InternalError(t *testing.T) {

	mock := &mockDeleter{
		deleteFn: func(name string) ([]string, error) {
			return nil, errors.New("some error")
		},
	}

	handler := DeleteFunctionHandler(mock)

	req := newRequest(http.MethodDelete, "/functions/test-fn")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDeleteHandler_MissingName(t *testing.T) {

	mock := &mockDeleter{}

	handler := DeleteFunctionHandler(mock)

	req := httptest.NewRequest(http.MethodDelete, "/functions/", nil)
	req = mux.SetURLVars(req, map[string]string{
		"functionName": "",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
