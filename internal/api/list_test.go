package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

type mockListService struct {
	listFn func() ([]models.Function, error)
}

func (m *mockListService) ListFunctions() ([]models.Function, error) {
	return m.listFn()
}

func TestListFunctionsHandler_Success(t *testing.T) {

	mock := &mockListService{
		listFn: func() ([]models.Function, error) {
			return []models.Function{
				{Name: "calc", Version: "v1", Status: "active"},
				{Name: "auth", Version: "v2", Status: "inactive"},
			}, nil
		},
	}

	handler := ListFunctionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListFunctionsHandler_Empty(t *testing.T) {

	mock := &mockListService{
		listFn: func() ([]models.Function, error) {
			return []models.Function{}, nil
		},
	}

	handler := ListFunctionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestListFunctionsHandler_Error(t *testing.T) {

	mock := &mockListService{
		listFn: func() ([]models.Function, error) {
			return nil, errors.New("db error")
		},
	}

	handler := ListFunctionsHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/functions", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
