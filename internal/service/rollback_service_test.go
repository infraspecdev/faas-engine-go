package service

import (
	"context"
	"errors"
	"faas-engine-go/internal/sqlite/models"
	"fmt"
	"strings"
	"testing"
	"time"
)

type fakeRollbackStore struct {
	listVersionsErr     error
	listVersionsResult  []models.Function
	rollbackToErr       error
	rollbackToPrevious  string
	getHistoryErr       error
	getHistoryResult    []models.VersionHistory
	getContainersErr    error
	getContainersResult []models.Container
	removeContainerErr  error
	getActiveFnResult   *models.Function // Added for tests
}

func (f *fakeRollbackStore) GetActiveFunction(name string) (*models.Function, error) {
	if f.getActiveFnResult != nil {
		return f.getActiveFnResult, nil
	}
	// Default: return first version as active for testing
	if len(f.listVersionsResult) > 0 {
		result := f.listVersionsResult[0]
		result.Status = "active"
		return &result, nil
	}
	return nil, fmt.Errorf("function not found")
}

func (f *fakeRollbackStore) CreateInvocation(inv *models.Invocation) error {
	return nil
}

func (f *fakeRollbackStore) MarkInvocationRunning(invID string, containerID string) error {
	return nil
}

func (f *fakeRollbackStore) CompleteInvocation(invID string, status string, exitCode int, responsePayload []byte, logs string, startedAt time.Time) error {
	return nil
}

func (f *fakeRollbackStore) AcquireFreeContainer(functionID int) (*models.Container, error) {
	return nil, nil
}

func (f *fakeRollbackStore) MarkContainerFree(containerID string) error {
	return nil
}

func (f *fakeRollbackStore) RemoveContainer(containerID string) error {
	return f.removeContainerErr
}

func (f *fakeRollbackStore) CreateContainer(c *models.Container) error {
	return nil
}

func (f *fakeRollbackStore) GetContainersByFunction(functionID int) ([]models.Container, error) {
	return f.getContainersResult, f.getContainersErr
}

func (f *fakeRollbackStore) ListFunctionVersions(name string) ([]models.Function, error) {
	return f.listVersionsResult, f.listVersionsErr
}

func (f *fakeRollbackStore) DeleteFunction(name string) error {
	return nil
}

func (f *fakeRollbackStore) ListFunctions() ([]models.Function, error) {
	return nil, nil
}

func (f *fakeRollbackStore) RollbackToVersion(functionName, targetVersion, requestID string) (string, error) {
	return f.rollbackToPrevious, f.rollbackToErr
}

func (f *fakeRollbackStore) RollbackToVersionWithID(functionName, targetVersion, requestID string) (string, int, error) {
	// Return the version, a default functionID (1), and any error
	return f.rollbackToPrevious, 1, f.rollbackToErr
}

func (f *fakeRollbackStore) GetVersionHistory(functionName string, limit int) ([]models.VersionHistory, error) {
	return f.getHistoryResult, f.getHistoryErr
}

func (f *fakeRollbackStore) GetPreviousVersion(functionName string) (string, error) {
	return "", nil
}

func (f *fakeRollbackStore) UpdateCleanupStatus(functionID int, requestID string, status, errMsg string) error {
	return nil
}

type fakeRollbackContainerClient struct {
	stopErr     error
	deleteErr   error
	stopCount   int
	deleteCalls []string
}

func (f *fakeRollbackContainerClient) DeleteContainer(ctx context.Context, containerID string) error {
	f.deleteCalls = append(f.deleteCalls, containerID)
	return f.deleteErr
}

func (f *fakeRollbackContainerClient) StopContainer(ctx context.Context, containerID string) error {
	f.stopCount++
	return f.stopErr
}

func TestRollbackService_InvalidFunctionName(t *testing.T) {
	store := &fakeRollbackStore{}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "", "v1")

	if err == nil {
		t.Fatal("expected error for empty function name")
	}

	if err.Error() != "function name is required" {
		t.Fatalf("expected 'function name is required', got '%v'", err)
	}
}

func TestRollbackService_FunctionNotFound(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v1")

	if err == nil {
		t.Fatal("expected error for not found")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "image validation failed") {
		t.Fatalf("expected 'not found' error, got '%v'", err)
	}
}

func TestRollbackService_TargetVersionNotFound(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v1", Status: "active", Image: "myfunction:v1"},
			{ID: 2, Name: "myfunction", Version: "v2", Status: "inactive", Image: "myfunction:v2"},
		},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v3")

	if err == nil {
		t.Fatal("expected error for target version not found")
	}

	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "image validation failed") {
		t.Fatalf("expected 'not found' error, got '%v'", err)
	}
}

func TestRollbackService_ExplicitVersion(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v2", Status: "active", CreatedAt: time.Now(), Image: "myfunction:v2"},
			{ID: 2, Name: "myfunction", Version: "v1", Status: "inactive", CreatedAt: time.Now().Add(-time.Hour), Image: "myfunction:v1"},
		},
		rollbackToPrevious: "v2",
		// After rollback to v1, v1 should be active
		getActiveFnResult: &models.Function{ID: 2, Name: "myfunction", Version: "v1", Status: "active", Image: "myfunction:v1"},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	result, err := service.Rollback(context.Background(), "myfunction", "v1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FunctionName != "myfunction" {
		t.Fatalf("expected function name 'myfunction', got '%s'", result.FunctionName)
	}

	if result.PreviousVersion != "v2" {
		t.Fatalf("expected previous version 'v2', got '%s'", result.PreviousVersion)
	}

	if result.CurrentVersion != "v1" {
		t.Fatalf("expected current version 'v1', got '%s'", result.CurrentVersion)
	}

	if result.RolledBackAt.IsZero() {
		t.Fatal("expected RolledBackAt to be set")
	}
}

func TestRollbackService_ImplicitVersion(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v3", Status: "active", CreatedAt: time.Now(), Image: "myfunction:v3"},
			{ID: 2, Name: "myfunction", Version: "v2", Status: "inactive", CreatedAt: time.Now().Add(-time.Hour), Image: "myfunction:v2"},
			{ID: 3, Name: "myfunction", Version: "v1", Status: "inactive", CreatedAt: time.Now().Add(-2 * time.Hour), Image: "myfunction:v1"},
		},
		rollbackToPrevious: "v3",
		// After rollback, v2 should be active
		getActiveFnResult: &models.Function{ID: 2, Name: "myfunction", Version: "v2", Status: "active", Image: "myfunction:v2"},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	result, err := service.Rollback(context.Background(), "myfunction", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PreviousVersion != "v3" {
		t.Fatalf("expected previous version 'v3', got '%s'", result.PreviousVersion)
	}

	if result.CurrentVersion != "v2" {
		t.Fatalf("expected current version 'v2', got '%s'", result.CurrentVersion)
	}
}

func TestRollbackService_NoPreviousVersion(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v1", Status: "active", Image: "myfunction:v1"},
		},
		rollbackToErr: fmt.Errorf("no previous version to rollback to"),
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "")

	if err == nil {
		t.Fatal("expected error for no previous version")
	}

	if err.Error() != "rollback failed: no previous version to rollback to" {
		t.Fatalf("expected 'rollback failed: no previous version to rollback to', got '%v'", err)
	}
}

func TestRollbackService_RollbackError(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v2", Status: "active", Image: "myfunction:v2"},
			{ID: 2, Name: "myfunction", Version: "v1", Status: "inactive", Image: "myfunction:v1"},
		},
		rollbackToErr: errors.New("database error"),
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v1")

	if err == nil {
		t.Fatal("expected error from rollback")
	}

	if !errors.Is(err, errors.New("rollback failed")) && err.Error() != "rollback failed: database error" {
		t.Fatalf("expected rollback error, got %v", err)
	}
}

func TestRollbackService_GetRollbackHistory_Success(t *testing.T) {
	now := time.Now()
	history := []models.VersionHistory{
		{
			ID:          1,
			FunctionID:  1,
			FromVersion: "v3",
			ToVersion:   "v2",
			TriggeredAt: now.Add(-5 * time.Minute),
		},
		{
			ID:          2,
			FunctionID:  1,
			FromVersion: "v2",
			ToVersion:   "v1",
			TriggeredAt: now.Add(-10 * time.Minute),
		},
	}

	store := &fakeRollbackStore{
		getHistoryResult: history,
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	result, err := service.GetRollbackHistory("myfunction", 20)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(result))
	}

	if result[0]["from"] != "v3" {
		t.Fatalf("expected first from version 'v3', got '%v'", result[0]["from"])
	}

	if result[0]["to"] != "v2" {
		t.Fatalf("expected first to version 'v2', got '%v'", result[0]["to"])
	}

	if result[1]["from"] != "v2" {
		t.Fatalf("expected second from version 'v2', got '%v'", result[1]["from"])
	}

	if result[1]["to"] != "v1" {
		t.Fatalf("expected second to version 'v1', got '%v'", result[1]["to"])
	}
}

func TestRollbackService_GetRollbackHistory_Empty(t *testing.T) {
	store := &fakeRollbackStore{
		getHistoryResult: []models.VersionHistory{},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	result, err := service.GetRollbackHistory("myfunction", 20)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty history, got %d items", len(result))
	}
}

func TestRollbackService_GetRollbackHistory_DefaultLimit(t *testing.T) {
	store := &fakeRollbackStore{
		getHistoryResult: []models.VersionHistory{},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	// When limit is 0 or negative, should default to 20
	result, err := service.GetRollbackHistory("myfunction", 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Fatalf("expected empty history, got %d items", len(result))
	}
}

func TestRollbackService_GetRollbackHistory_Error(t *testing.T) {
	store := &fakeRollbackStore{
		getHistoryErr: errors.New("database error"),
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.GetRollbackHistory("myfunction", 20)

	if err == nil {
		t.Fatal("expected error from history query")
	}
}

func TestRollbackService_ContainerCleanupAsync(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v2", Status: "active", CreatedAt: time.Now(), Image: "myfunction:v2"},
			{ID: 2, Name: "myfunction", Version: "v1", Status: "inactive", CreatedAt: time.Now().Add(-time.Hour), Image: "myfunction:v1"},
		},
		rollbackToPrevious: "v2",
		getActiveFnResult:  &models.Function{ID: 1, Name: "myfunction", Version: "v2", Status: "active", Image: "myfunction:v2"},
		getContainersResult: []models.Container{
			{ID: "c1", FunctionID: 2, Status: "free", CreatedAt: time.Now().Add(-time.Hour)},
			{ID: "c2", FunctionID: 2, Status: "busy", CreatedAt: time.Now().Add(-30 * time.Minute)},
		},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Allow some time for async cleanup goroutine
	time.Sleep(100 * time.Millisecond)

	// Async cleanup should have been called
	if client.stopCount == 0 {
		t.Fatal("expected StopContainer to be called during cleanup")
	}
}

func TestRollbackService_ContainerCleanupSkipsUnhealthyContainers(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v2", Status: "active", CreatedAt: time.Now(), Image: "myfunction:v2"},
			{ID: 2, Name: "myfunction", Version: "v1", Status: "inactive", CreatedAt: time.Now().Add(-time.Hour), Image: "myfunction:v1"},
		},
		rollbackToPrevious: "v2",
		getContainersResult: []models.Container{
			{ID: "c1", FunctionID: 2, Status: "terminated", CreatedAt: time.Now().Add(-time.Hour)},
			{ID: "c2", FunctionID: 2, Status: "failed", CreatedAt: time.Now().Add(-30 * time.Minute)},
		},
	}
	client := &fakeRollbackContainerClient{}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Allow some time for async cleanup goroutine
	time.Sleep(100 * time.Millisecond)

	// Should not clean up containers that are not in 'free' or 'busy' status
	if client.stopCount > 0 {
		t.Fatalf("expected no StopContainer calls for unhealthy containers, got %d", client.stopCount)
	}
}

func TestRollbackService_ContainerCleanupHandlesErrors(t *testing.T) {
	store := &fakeRollbackStore{
		listVersionsResult: []models.Function{
			{ID: 1, Name: "myfunction", Version: "v2", Status: "active", CreatedAt: time.Now(), Image: "myfunction:v2"},
			{ID: 2, Name: "myfunction", Version: "v1", Status: "inactive", CreatedAt: time.Now().Add(-time.Hour), Image: "myfunction:v1"},
		},
		rollbackToPrevious: "v2",
		getContainersResult: []models.Container{
			{ID: "c1", FunctionID: 2, Status: "free", CreatedAt: time.Now().Add(-time.Hour)},
		},
		removeContainerErr: errors.New("removal failed"),
	}
	client := &fakeRollbackContainerClient{
		deleteErr: errors.New("delete failed"),
	}
	service := NewRollbackService(store, client)

	_, err := service.Rollback(context.Background(), "myfunction", "v1")

	// Rollback should succeed even if cleanup has errors
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Allow some time for async cleanup goroutine
	time.Sleep(100 * time.Millisecond)

	// Even with errors, cleanup should attempt to process containers
	if client.stopCount == 0 {
		t.Fatal("expected cleanup to attempt StopContainer")
	}
}
