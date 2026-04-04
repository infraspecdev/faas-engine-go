package service

import (
	"database/sql"
	"faas-engine-go/internal/sqlite"
	"faas-engine-go/internal/sqlite/models"
	sqlstore "faas-engine-go/internal/sqlite/store"
	"time"
)

type Store interface {
	GetActiveFunction(name string) (*models.Function, error)
	CreateInvocation(inv *models.Invocation) error
	MarkInvocationRunning(invID string, containerID string) error
	CompleteInvocation(
		invID string,
		status string,
		exitCode int,
		responsePayload []byte,
		logs string,
		startedAt time.Time,
	) error
	AcquireFreeContainer(functionID int) (*models.Container, error)
	MarkContainerFree(containerID string) error
	RemoveContainer(containerID string) error
	CreateContainer(c *models.Container) error
	GetContainersByFunction(functionID int) ([]models.Container, error)
	ListFunctionVersions(name string) ([]models.Function, error)
	DeleteFunction(name string) error
	ListFunctions() ([]models.Function, error)
	// RollbackToVersionWithID performs atomic rollback and returns (previousVersion, deactivatedFunctionID, error)
	// The deactivatedFunctionID is needed for UpdateCleanupStatus to track cleanup results correctly (Fix #7)
	RollbackToVersionWithID(functionName, targetVersion, requestID string) (string, int, error)
	RollbackToVersion(functionName, targetVersion, requestID string) (string, error) // Fix #6: requestID for idempotency
	GetPreviousVersion(functionName string) (string, error)                          // Fix #1, #2: Get previous version by id ordering
	GetVersionHistory(functionName string, limit int) ([]models.VersionHistory, error)
	// Rollback stack operations (LIFO for version management)
	PushRollbackStack(functionName string, version string) error
	PopRollbackStack(functionName string) (string, error)
	GetNextRollbackVersion(functionName string) (string, error)
	UpdateCleanupStatus(functionID int, requestID string, status, errMsg string) error // Fix #7: Persist cleanup results
}

type realStore struct {
	db *sql.DB
}

var _ Store = (*realStore)(nil)

func NewStore() Store {
	return &realStore{db: sqlite.DB}
}

func (s *realStore) GetActiveFunction(name string) (*models.Function, error) {
	return sqlstore.GetActiveFunction(s.db, name)
}

func (s *realStore) CreateInvocation(inv *models.Invocation) error {
	return sqlstore.CreateInvocation(s.db, inv)
}

func (s *realStore) MarkInvocationRunning(invID string, containerID string) error {
	return sqlstore.MarkInvocationRunning(s.db, invID, containerID)
}

func (s *realStore) CompleteInvocation(
	invID string,
	status string,
	exitCode int,
	responsePayload []byte,
	logs string,
	startedAt time.Time,
) error {
	return sqlstore.CompleteInvocation(
		s.db,
		invID,
		status,
		exitCode,
		responsePayload,
		logs,
		startedAt,
	)
}

func (s *realStore) AcquireFreeContainer(functionID int) (*models.Container, error) {
	return sqlstore.AcquireFreeContainer(s.db, functionID)
}

func (s *realStore) MarkContainerFree(containerID string) error {
	return sqlstore.MarkContainerFree(s.db, containerID)
}

func (s *realStore) RemoveContainer(containerID string) error {
	return sqlstore.RemoveContainer(s.db, containerID)
}

func (s *realStore) CreateContainer(c *models.Container) error {
	return sqlstore.CreateContainer(s.db, c)
}

func (s *realStore) GetContainersByFunction(functionID int) ([]models.Container, error) {
	return sqlstore.GetContainersByFunction(s.db, functionID)
}

func (s *realStore) ListFunctionVersions(name string) ([]models.Function, error) {
	return sqlstore.ListFunctionVersions(s.db, name)
}

func (s *realStore) DeleteFunction(name string) error {
	return sqlstore.DeleteFunction(s.db, name)
}

func (s *realStore) ListFunctions() ([]models.Function, error) {
	return sqlstore.ListFunctions(s.db)
}

func (s *realStore) RollbackToVersionWithID(functionName, targetVersion, requestID string) (string, int, error) {
	return sqlstore.RollbackToVersionWithID(s.db, functionName, targetVersion, requestID)
}

func (s *realStore) RollbackToVersion(functionName, targetVersion, requestID string) (string, error) {
	return sqlstore.RollbackToVersion(s.db, functionName, targetVersion, requestID)
}

func (s *realStore) GetPreviousVersion(functionName string) (string, error) {
	return sqlstore.GetPreviousVersion(s.db, functionName)
}

func (s *realStore) GetVersionHistory(functionName string, limit int) ([]models.VersionHistory, error) {
	return sqlstore.GetVersionHistory(s.db, functionName, limit)
}

func (s *realStore) PushRollbackStack(functionName string, version string) error {
	return sqlstore.PushRollbackStack(s.db, functionName, version)
}

func (s *realStore) PopRollbackStack(functionName string) (string, error) {
	return sqlstore.PopRollbackStack(s.db, functionName)
}

func (s *realStore) GetNextRollbackVersion(functionName string) (string, error) {
	return sqlstore.GetNextRollbackVersion(s.db, functionName)
}

func (s *realStore) UpdateCleanupStatus(functionID int, requestID string, status, errMsg string) error {
	return sqlstore.UpdateCleanupStatus(s.db, functionID, requestID, status, errMsg)
}
