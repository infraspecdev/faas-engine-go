package service

import (
	"errors"
	"testing"
	"time"

	"faas-engine-go/internal/sqlite/models"
)

type mockStoreForLogs struct {
	getActiveFunction    func(name string) (*models.Function, error)
	listFunctionVersions func(name string) ([]models.Function, error)
	getInvocationLogs    func(functionID string, limit int) ([]models.Invocation, error)
}

func (m *mockStoreForLogs) GetActiveFunction(name string) (*models.Function, error) {
	if m.getActiveFunction != nil {
		return m.getActiveFunction(name)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreForLogs) ListFunctionVersions(name string) ([]models.Function, error) {
	if m.listFunctionVersions != nil {
		return m.listFunctionVersions(name)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreForLogs) GetInvocationLogs(functionID string, limit int) ([]models.Invocation, error) {
	if m.getInvocationLogs != nil {
		return m.getInvocationLogs(functionID, limit)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStoreForLogs) CreateInvocation(inv *models.Invocation) error                { return nil }
func (m *mockStoreForLogs) MarkInvocationRunning(invID string, containerID string) error { return nil }
func (m *mockStoreForLogs) CompleteInvocation(invID string, status string, exitCode int, responsePayload []byte, logs string, startedAt time.Time) error {
	return nil
}
func (m *mockStoreForLogs) CompleteInvocationAndMarkFree(invID, status string, exitCode int, responsePayload []byte, logs string, startedAt time.Time, containerID string, shouldMarkFree bool) error {
	return nil
}
func (m *mockStoreForLogs) AcquireFreeContainer(functionID string) (*models.Container, error) {
	return nil, nil
}
func (m *mockStoreForLogs) MarkContainerFree(containerID string) error { return nil }
func (m *mockStoreForLogs) RemoveContainer(containerID string) error   { return nil }
func (m *mockStoreForLogs) CreateContainer(c *models.Container) error  { return nil }
func (m *mockStoreForLogs) GetContainersByFunction(functionID string) ([]models.Container, error) {
	return nil, nil
}
func (m *mockStoreForLogs) DeleteFunction(name string) error          { return nil }
func (m *mockStoreForLogs) ListFunctions() ([]models.Function, error) { return nil, nil }
func (m *mockStoreForLogs) RollbackToVersion(functionName, targetVersion, requestID string) (string, error) {
	return "", nil
}
func (m *mockStoreForLogs) RollbackToVersionWithID(functionName, targetVersion, requestID string) (string, string, error) {
	return "", "", nil
}
func (m *mockStoreForLogs) GetVersionHistory(functionName string, limit int) ([]models.VersionHistory, error) {
	return nil, nil
}
func (m *mockStoreForLogs) GetPreviousVersion(functionName string) (string, error) {
	return "", nil
}
func (m *mockStoreForLogs) UpdateCleanupStatus(functionID string, requestID string, status, errMsg string) error {
	return nil
}

func TestGetLogsByName_Success(t *testing.T) {
	mock := &mockStoreForLogs{
		getActiveFunction: func(name string) (*models.Function, error) {
			return &models.Function{ID: "fn-1", Name: "test"}, nil
		},
		getInvocationLogs: func(functionID string, limit int) ([]models.Invocation, error) {
			return []models.Invocation{
				{ID: "inv1", Status: "success", Logs: "ok", DurationMs: 50},
			}, nil
		},
	}

	logService := &LogService{store: mock}
	res, err := logService.GetLogsByName("test", 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatal("expected 1 log")
	}
}

func TestGetLogsByName_FunctionNotFound(t *testing.T) {
	mock := &mockStoreForLogs{
		getActiveFunction: func(name string) (*models.Function, error) {
			return nil, nil
		},
	}

	logService := &LogService{store: mock}
	_, err := logService.GetLogsByName("test", 10)

	if err != ErrFunctionNotFound {
		t.Fatalf("expected ErrFunctionNotFound, got %v", err)
	}
}

func TestGetLogsByNameAndVersion_Success(t *testing.T) {
	mock := &mockStoreForLogs{
		listFunctionVersions: func(name string) ([]models.Function, error) {
			return []models.Function{
				{ID: "fn-2", Name: "test", Version: "v2"},
			}, nil
		},
		getInvocationLogs: func(functionID string, limit int) ([]models.Invocation, error) {
			return []models.Invocation{
				{ID: "invX", Status: "success", Logs: "v2", DurationMs: 70},
			}, nil
		},
	}

	logService := &LogService{store: mock}
	res, err := logService.GetLogsByNameAndVersion("test", "v2", 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 1 {
		t.Fatal("expected 1 log")
	}

	if res[0].ID != "invX" {
		t.Fatal("unexpected log ID")
	}
}

func TestGetLogsByNameAndVersion_NotFound(t *testing.T) {
	mock := &mockStoreForLogs{
		listFunctionVersions: func(name string) ([]models.Function, error) {
			return []models.Function{}, nil
		},
	}

	logService := &LogService{store: mock}
	_, err := logService.GetLogsByNameAndVersion("test", "v2", 10)

	if err != ErrFunctionNotFound {
		t.Fatalf("expected ErrFunctionNotFound, got %v", err)
	}
}
