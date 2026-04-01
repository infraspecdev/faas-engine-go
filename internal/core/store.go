package core

import (
	"faas-engine-go/internal/sqlite/models"
	"time"
)

// Store is the data access interface for the application.
// It abstracts all database operations.
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
	CompleteInvocationAndMarkFree(
		invID string,
		status string,
		exitCode int,
		responsePayload []byte,
		logs string,
		startedAt time.Time,
		containerID string,
		shouldMarkFree bool,
	) error
	AcquireFreeContainer(functionID string) (*models.Container, error)
	MarkContainerFree(containerID string) error
	RemoveContainer(containerID string) error
	CreateContainer(c *models.Container) error
	GetContainersByFunction(functionID string) ([]models.Container, error)
	ListFunctionVersions(name string) ([]models.Function, error)
	DeleteFunction(name string) error
	ListFunctions() ([]models.Function, error)
	GetInvocationLogs(functionID string, limit int) ([]models.Invocation, error)
}
