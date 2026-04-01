package service

import (
	"database/sql"

	"faas-engine-go/internal/core"
	"faas-engine-go/internal/sqlite/models"
)

type LogEntry struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Logs     string `json:"logs"`
	Duration int    `json:"duration"`
}

type LogService struct {
	store core.Store
}

func NewLogService(db *sql.DB) *LogService {
	return &LogService{
		store: NewStore(db),
	}
}

func (l *LogService) GetLogsByName(functionName string, limit int) ([]LogEntry, error) {

	functionID, err := l.store.GetActiveFunction(functionName)
	if err != nil {
		return nil, err
	}

	if functionID == nil {
		return nil, ErrFunctionNotFound
	}

	invocations, err := l.store.GetInvocationLogs(functionID.ID, limit)
	if err != nil {
		return nil, err
	}

	return l.convertInvocations(invocations), nil
}

func (l *LogService) GetLogsByNameAndVersion(functionName, version string, limit int) ([]LogEntry, error) {

	// Get function by name and version
	functions, err := l.store.ListFunctionVersions(functionName)
	if err != nil {
		return nil, err
	}

	var fn *models.Function
	for i := range functions {
		if functions[i].Version == version {
			fn = &functions[i]
			break
		}
	}

	if fn == nil {
		return nil, ErrFunctionNotFound
	}

	invocations, err := l.store.GetInvocationLogs(fn.ID, limit)
	if err != nil {
		return nil, err
	}

	return l.convertInvocations(invocations), nil
}

func (l *LogService) convertInvocations(invocations []models.Invocation) []LogEntry {
	var result []LogEntry

	for _, inv := range invocations {
		result = append(result, LogEntry{
			ID:       inv.ID,
			Status:   inv.Status,
			Logs:     inv.Logs,
			Duration: inv.DurationMs,
		})
	}

	return result
}
