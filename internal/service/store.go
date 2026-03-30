package service

import (
	"database/sql"
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
}

type realStore struct {
	db *sql.DB
}

var _ Store = (*realStore)(nil)

func NewStore(db *sql.DB) Store {
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
