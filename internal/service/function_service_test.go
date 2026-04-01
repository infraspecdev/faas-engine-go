package service

import (
	"errors"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

type mockFunctionStore struct {
	listFn func(name string) ([]models.Function, error)
}

func (m *mockFunctionStore) ListFunctionVersions(name string) ([]models.Function, error) {
	return m.listFn(name)
}

func TestGetVersions_Success(t *testing.T) {

	mock := &mockFunctionStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{
				{Version: "v1", Status: "active"},
				{Version: "v2", Status: "inactive"},
			}, nil
		},
	}

	svc := NewFunctionVersionService(mock)

	res, err := svc.GetVersions("test-fn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(res))
	}

	if !res[0].Active {
		t.Fatalf("expected first version to be active")
	}

	if res[1].Active {
		t.Fatalf("expected second version to be inactive")
	}
}

func TestGetVersions_FunctionNotFound(t *testing.T) {

	mock := &mockFunctionStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{}, nil
		},
	}

	svc := NewFunctionVersionService(mock)

	_, err := svc.GetVersions("missing-fn")

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "function not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetVersions_StoreError(t *testing.T) {

	mock := &mockFunctionStore{
		listFn: func(name string) ([]models.Function, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewFunctionVersionService(mock)

	_, err := svc.GetVersions("test-fn")

	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "db error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetVersions_AllInactive(t *testing.T) {

	mock := &mockFunctionStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{
				{Version: "v1", Status: "inactive"},
				{Version: "v2", Status: "inactive"},
			}, nil
		},
	}

	svc := NewFunctionVersionService(mock)

	res, err := svc.GetVersions("test-fn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, v := range res {
		if v.Active {
			t.Fatalf("expected all versions to be inactive")
		}
	}
}
