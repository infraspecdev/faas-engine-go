package service

import (
	"errors"
	"testing"

	"faas-engine-go/internal/sqlite/models"
)

type mockDeleteStore struct {
	listFn   func(name string) ([]models.Function, error)
	deleteFn func(name string) error
}

func (m *mockDeleteStore) ListFunctionVersions(name string) ([]models.Function, error) {
	return m.listFn(name)
}

func (m *mockDeleteStore) DeleteFunction(name string) error {
	return m.deleteFn(name)
}

type mockRegistry struct {
	getFn func(name, version string) (string, error)
	delFn func(name, digest string) error
}

func (m *mockRegistry) GetDigest(name, version string) (string, error) {
	return m.getFn(name, version)
}

func (m *mockRegistry) DeleteImage(name, digest string) error {
	return m.delFn(name, digest)
}

func newTestService(store DeleteStore, reg RegistryClient) *functionDeleteService {
	return &functionDeleteService{
		store:    store,
		registry: reg,
		retry: func(attempts int, fn func() error) error {
			return fn()
		},
	}
}

func TestDeleteFunction_Success(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{{Version: "v1"}, {Version: "v2"}}, nil
		},
		deleteFn: func(name string) error { return nil },
	}

	reg := &mockRegistry{
		getFn: func(name, version string) (string, error) { return "digest", nil },
		delFn: func(name, digest string) error { return nil },
	}

	svc := newTestService(store, reg)

	failed, err := svc.DeleteFunction("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if failed != nil {
		t.Fatalf("expected no failures")
	}
}

func TestDeleteFunction_NotFound(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{}, nil
		},
	}

	svc := newTestService(store, nil)

	_, err := svc.DeleteFunction("missing")

	if !errors.Is(err, ErrFunctionNotFound) {
		t.Fatalf("expected ErrFunctionNotFound")
	}
}

func TestDeleteFunction_ListError(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return nil, errors.New("db error")
		},
	}

	svc := newTestService(store, nil)

	_, err := svc.DeleteFunction("test")

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteFunction_PartialFailure(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{{Version: "v1"}, {Version: "v2"}}, nil
		},
		deleteFn: func(name string) error { return nil },
	}

	reg := &mockRegistry{
		getFn: func(name, version string) (string, error) { return "digest", nil },
		delFn: func(name, digest string) error { return errors.New("fail") },
	}

	svc := newTestService(store, reg)

	failed, err := svc.DeleteFunction("test")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(failed) == 0 {
		t.Fatal("expected failed versions")
	}
}

func TestDeleteFunction_AlreadyDeleted(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{{Version: "v1"}}, nil
		},
		deleteFn: func(name string) error { return nil },
	}

	reg := &mockRegistry{
		getFn: func(name, version string) (string, error) { return "", errNotFound },
		delFn: func(name, digest string) error { return nil },
	}

	svc := newTestService(store, reg)

	failed, err := svc.DeleteFunction("test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if failed != nil {
		t.Fatalf("expected no failures")
	}
}

func TestDeleteFunction_DBDeleteFail(t *testing.T) {

	store := &mockDeleteStore{
		listFn: func(name string) ([]models.Function, error) {
			return []models.Function{{Version: "v1"}}, nil
		},
		deleteFn: func(name string) error {
			return errors.New("db delete failed")
		},
	}

	reg := &mockRegistry{
		getFn: func(name, version string) (string, error) { return "digest", nil },
		delFn: func(name, digest string) error { return nil },
	}

	svc := newTestService(store, reg)

	_, err := svc.DeleteFunction("test")

	if err == nil {
		t.Fatal("expected error")
	}
}
