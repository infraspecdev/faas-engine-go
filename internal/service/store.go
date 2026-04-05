package service

import (
	"database/sql"
	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sqlite/models"
	sqlstore "faas-engine-go/internal/sqlite/store"
	"time"
)

type realStore struct {
	db *sql.DB
}

var _ core.Store = (*realStore)(nil)

func NewStore(db *sql.DB) core.Store {
	return &realStore{db: db}
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

func (s *realStore) CompleteInvocationAndMarkFree(
	invID string,
	status string,
	exitCode int,
	responsePayload []byte,
	logs string,
	startedAt time.Time,
	containerID string,
	shouldMarkFree bool,
) error {
	return sqlstore.CompleteInvocationAndMarkFree(
		s.db,
		invID,
		status,
		exitCode,
		responsePayload,
		logs,
		startedAt,
		containerID,
		shouldMarkFree,
	)
}

func (s *realStore) AcquireFreeContainer(functionID string) (*models.Container, error) {
	return sqlstore.GetFreeContainer(s.db, functionID)
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

func (s *realStore) GetContainersByFunction(functionID string) ([]models.Container, error) {
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

func (s *realStore) GetInvocationLogs(functionID string, limit int) ([]models.Invocation, error) {
	return sqlstore.GetInvocationLogsByFunction(s.db, functionID, limit)
}

func (s *realStore) RollbackToVersionWithID(functionName, targetVersion, requestID string) (string, string, error) {
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

func (s *realStore) UpdateCleanupStatus(functionID string, requestID string, status, errMsg string) error {
	return sqlstore.UpdateCleanupStatus(s.db, functionID, requestID, status, errMsg)
}
